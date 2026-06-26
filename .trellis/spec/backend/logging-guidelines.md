# 日志规范

> 日志用于排查业务动作和外部系统交互，不用于暴露请求全文、密钥、连接串或个人信息。

## 基本原则

- 优先使用 go-zero `logx` 或相邻服务已有日志模式。
- 日志应包含服务/模块、业务动作、关键 ID、外部系统名、失败阶段和错误原因。
- 不在每一层重复打印同一个错误；边界层或关键上下文层记录即可。
- 对高频路径保持克制，避免在数据采集、MQTT、Socket、SSE、Kafka 等路径制造过量日志。

## 推荐记录

- 服务启动、配置加载结果和依赖初始化状态，但不要打印敏感配置值。
- 外部系统调用失败：RPC、MQTT、Kafka、Redis、数据库、OSS、Docker、DJI Cloud API、Eino Provider 等。
- 状态机关键流转：任务调度、计划执行、设备上下线、航线/DRC 状态、Socket 房间和会话变更。
- 批处理摘要：批次 ID、数量、耗时、成功/失败数，不记录完整大 payload。

## 禁止记录

- 密码、Token、API Key、认证头、证书、SSH 凭据。
- 数据库连接串、对象存储配置、MQTT Broker 账号密码、AI Provider 密钥。
- 身份证号、手机号、个人隐私数据。
- 内网地址、远程服务器账号、个人本地路径、IDE 配置路径。
- 完整请求体、完整异常堆栈或大块协议报文，除非已脱敏且确实用于排障。

## 常见错误

- 为了方便调试直接打印配置结构体。
- 在错误消息中拼接账号、密码、连接串或远程地址。
- 多层重复 `Errorf`，导致同一失败被打印多次。**RPC 错误由 `LoggerInterceptor` 统一打印，Logic 层禁止重复打印。**
- 对流式、采集、消息消费路径逐条打印成功日志，造成噪声和性能压力。
- 调试时本能 "加更多日志"，但正确方向是 "减少日志 + 精确错误分类"。

## Scenario: 协议层上下文注入

### 1. Scope / Trigger

MQTT 协议层（`common/mqttx` → `common/djisdk`）收到设备上行消息时，将协议字段注入 `context.Context`，使下游 handler 和业务日志自动携带设备标识、事务 ID 和时间戳，无需逐行拼接。

### 2. Signatures

两层上下文注入：

**mqttx 基础层**（`processMessage`, `common/mqttx/client.go`）：所有 MQTT 消费入口注入：
```go
ctx = logx.ContextWithFields(ctx,
    logx.Field("client", c.GetClientID()),
    logx.Field("topic", msg.Topic()),
    logx.Field("topic_template", topicTemplate),
    logx.Field("payload_bytes", len(payload)),
    logx.Field("payload_size", tool.DecimalBytes(int64(len(payload)), 1)),
)
```

**djisdk 协议层**（`handler.go`）：各上行 handler 注入协议字段：

| Handler | 字段 |
|---------|------|
| `HandleEvents` | `gateway_sn`, `method`, `tid`, `bid`, `need_reply`, `ts`, `ts_fmt` |
| `HandleStatus` | `gateway_sn`, `method`, `tid`, `bid`, `ts`, `ts_fmt` |
| `HandleRequests` | `gateway_sn`, `method`, `tid`, `bid`, `ts`, `ts_fmt` |
| `HandleDrcUp` | `gateway_sn`, `method`, `tid`, `bid`, `ts`, `ts_fmt` |
| `HandleOsd` | `device_sn`, `tid`, `bid`, `ts`, `ts_fmt` |
| `HandleState` | `device_sn`, `tid`, `bid`, `ts`, `ts_fmt` |

时间戳使用 `github.com/dromara/carbon/v2` 格式化，`ts_fmt` 格式为 `"2006-01-02 15:04:05.000"`（`ToDateTimeMilliString`）。

### 3. Contracts

- 协议字段只注入 `ctx`，**不**写入消息文本——消息文本只保留 `"[dji-sdk] events"` 形式的纯动作标识
- `logx.WithContext(enhancedCtx)` 的 logx 输出会自动附加上下文字段，结构化格式（JSON）和文本格式均可查看
- 错误信息按需显式包含 `sn=`/`method=` 便于 grep，即使这些字段已在 ctx 中
- `tsFields` 辅助函数在 `timestamp <= 0` 时返回 nil，不会注入 1970 年时间

### 4. Validation

- 搜索 `logx.ContextWithFields` 确认每个 handler 都注入了协议字段
- 搜索 `payload=%s` 或 `string(payload)` 确认无明文 payload 泄露
- 新增 handler 必须按照上述模式注入上下文，**禁止**在消息文本中重复 ctx 已携带的字段

### 5. Tests

- `TestLogFieldsDoesNotIncludePayloadOrSensitiveData` 验证 `logFields` 不泄露敏感字段
- 各 handler 的 test 验证 handler 被调用（隐含 ctx 传播），不校验 ctx 内字段

## Scenario: 高频路径按 context 关闭 GORM SQL trace

### 1. Scope / Trigger

- Trigger: 高频采集、MQTT、Socket、SSE、Kafka 等路径会产生大量 GORM `Trace` SQL 日志，但仍需要保留业务错误日志和其他低频 SQL 诊断能力。

### 2. Signatures

- Context helper: `gormx.WithoutSQLTrace(ctx context.Context) context.Context`。
- Logger boundary: `func (c *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error)` 在检测到该 context 标记时直接返回，不调用 `fc()`。
- Service config example: `Telemetry.DisableOsdSQLTrace bool` 用于服务私有配置，不改全局 `gormx.Config.LogLevel`。

### 3. Contracts

- `gormx.WithoutSQLTrace(ctx)` 只影响 GORM logger 的 `Trace` 输出，包括正常 SQL、慢 SQL、record not found 和 SQL error trace。
- 业务层 `logx.WithContext(ctx).Errorf(...)` 不受影响，调用方仍需在边界层记录业务失败。
- 高频路径必须在进入 DB helper 前包裹 context，例如 OSD 写库调用 `FirstOrCreate` 前处理。
- 服务级配置默认保持旧行为；只有配置显式开启时才对目标高频路径使用该 context。

### 4. Validation & Error Matrix

- Context 未设置 + `LogLevel=info` -> `Trace` 可输出正常 SQL。
- Context 未设置 + 慢 SQL + `LogLevel=warn` -> `Trace` 可输出慢 SQL。
- Context 设置 `WithoutSQLTrace` -> `Trace` 不调用 `fc()`，不输出 SQL trace。
- DB helper 返回 error -> 调用方仍按业务边界记录错误；不要依赖 GORM trace 作为唯一错误日志。

### 5. Good/Base/Bad Cases

- Good: OSD 这类高频写库路径通过服务配置开启 `DisableOsdSQLTrace`，只在该 handler 内 `ctx = gormx.WithoutSQLTrace(ctx)`。
- Base: 临时排查某个高频路径时关闭配置，让 `gormx.Config.LogLevel` 恢复输出 SQL trace。
- Bad: 为了压低单一路径日志把 `DB.LogLevel` 全局改成 `silent`，导致其他数据库错误和慢 SQL诊断一起丢失。

### 6. Tests Required

- Unit: `TestGormLoggerTraceSkipsSQLWhenContextDisablesTrace` 断言设置 context 后 `Trace` 不调用 `fc()`。
- Service compile/test: 跑目标服务 handler 测试，确认配置字段能传到高频 handler 且不破坏既有写库行为。
- Regression: 高频 handler 写库仍成功，业务错误日志仍由 handler 显式输出。

### 7. Wrong vs Correct

#### Wrong

```go
// 为了关闭 OSD SQL 日志，误伤整个服务的 DB 诊断能力。
DB:
  LogLevel: silent
```

#### Correct

```go
if disableSQLTrace {
    ctx = gormx.WithoutSQLTrace(ctx)
}
if err := db.WithContext(ctx).Where(where).Assign(updateData).FirstOrCreate(createData).Error; err != nil {
	logx.WithContext(ctx).Errorf("upsert failed: %v", err)
}
```
