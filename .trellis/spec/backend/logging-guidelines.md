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

## Scenario: Errorf vs Errorw 选择标准

### 1. 决策树

```
有链路追踪价值的结构化字段（gateway_sn/method/tid/room/host/port/name）？
├── 是 → Errorw + 仅传必要的 Field
└── 否 → Errorf
```

| 条件 | 使用 | 示例 |
|------|------|------|
| 有 `gateway_sn`/`method`/`tid`/`room` 等可索引字段 | `Errorw` | `logDjiSDKError(ctx, "command rejected", gatewaySn, method, tid, err)` |
| 仅错误文本，无额外字段 | `Errorf` | `logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal osd failed: %v", err)` |
| ctx 已注入相同字段 | `Errorf`（不再传 Field） | ctx 已有 `socketId`，则 `Errorf("...failed: %v", err)` |
| 中间件/拦截器级（asynq/Interceptor） | `Errorf` 保留 `%+v` | `logx.WithContext(ctx).Errorf("rpc error: %+v", err)` |

### 2. `%+v` vs `%v` vs `err.Error()` 选择

| 场景 | 格式 | 原因 |
|------|------|------|
| 中间件/拦截器（error 来源不确定，可能有 `pkg/errors` 堆栈） | `%+v` | 保留完整堆栈链 |
| SDK 层错误（`fmt.Errorf` 刚创建，无堆栈） | `err.Error()` | `%+v` 无额外信息 |
| handler 内错误（marshaling/callback 返回） | `%v` | gRPC/DB/SDK 错误，无 `pkg/errors` 堆栈 |
| `errors.New` / 纯字符串错误 | `%v` | 无堆栈 |

### 3. Field 权限

**Field 只放有链路追踪价值的值**：
- ✅ `gateway_sn`、`method`、`tid`、`device_sn`、`room`、`host`、`port`、`name`、`version`
- ❌ 纯计数（如 `len(dst)`、`rowsAffected`）→ 放消息文本
- ❌ ctx 已注入的字段 → 不重复传

### 4. 日志前缀

所有 `common/` 包统一使用 **`[小写包名]`** 前缀：

| 包 | 前缀 |
|------|------|
| `djisdk` | `[dji-sdk]` |
| `mqttx` | `[mqtt]` |
| `socketiox` | `[socketio]` |
| `nacosx` | `[nacos]` |
| `mcpx` | `[mcpx]` |
| `app/djicloud` | `[dji-cloud]` |

前缀 **只出现在日志消息中**，不出现在 `fmt.Errorf` / `DJIError.Error()` / `PlatformError.Error()` 返回的错误值里。

### 5. 反模式（本 session 已修复）

```go
// Bad — Errorw 没有字段，应降级为 Errorf
logx.WithContext(ctx).Errorw("[dji-sdk] unmarshal failed: "+err.Error())

// Bad — fmt.Errorf 带 [xxx] 前缀
fmt.Errorf("[dji-sdk] command failed: %w", err)

// Bad — Errorw 传 ctx 已有的字段
logx.WithContext(connectCtx).Errorw("...", logx.Field("conn", socket.Id))  // connectCtx 已有 socketId

// Bad — 纯计数值放 Field
logx.Infow("auto migrate success", logx.Field("tables", len(dst)))

// Good — Errorf 用于无字段错误
logx.WithContext(ctx).Errorf("[dji-sdk] unmarshal osd failed: %v", err)

// Good — 干净的错误值
fmt.Errorf("command failed: %w", err)

// Good — 纯计数放消息文本
logx.Infof("auto migrate %d tables success", len(dst))
```

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
- 错误只在消息文本描述"发生了什么"；`gateway_sn`/`device_sn`/`method` 等已在 ctx 中，不重复写入消息也不重复传 Field
- `tsFields` 辅助函数在 `timestamp <= 0` 时返回 nil，不会注入 1970 年时间

### 4. Validation

- 搜索 `logx.ContextWithFields` 确认每个 handler 都注入了协议字段
- 搜索 `payload=%s` 或 `string(payload)` 确认无明文 payload 泄露
- 新增 handler 必须按照上述模式注入上下文，**禁止**在消息文本中重复 ctx 已携带的字段

### 5. Tests

- `TestLogFieldsDoesNotIncludePayloadOrSensitiveData` 验证 `logFields` 不泄露敏感字段
- 各 handler 的 test 验证 handler 被调用（隐含 ctx 传播），不校验 ctx 内字段

## Scenario: MQTT 客户端生命周期与订阅日志

### 1. Scope / Trigger

`common/mqttx/client.go` 和 `common/mqttx/dispatcher.go` 中 `[mqtt]` 前缀日志。涉及连接/断连/订阅恢复/消息分发。

### 2. Signatures

统一格式：**小写动作词**。

连接与断连：

```go
logx.Infof("[mqtt] connected client=%s", c.cfg.ClientID)
logx.Errorw("[mqtt] connection lost: "+err.Error())
logx.Info("[mqtt] connection closed")
```

订阅恢复：

```go
logx.Infof("[mqtt] subscribed topic=%s", topicTemplate)
logx.Errorw("[mqtt] subscribe failed: "+err.Error())
logx.Infof("[mqtt] restore subscriptions done subscribed=%d skipped=%d", result.subscribed, result.skipped)
```

消息分发（dispatcher.go，ctx 已带 client/topic/topic_template/payload_bytes）：

```go
logx.WithContext(ctx).Info("[mqtt] no handler registered")
logx.WithContext(ctx).Errorw("[mqtt] reply handler error: "+err.Error())
logx.WithContext(ctx).Errorw("[mqtt] handler error: "+err.Error())
```

### 3. Contracts

- 所有 `[mqtt]` 前缀日志统一用小写动作词
- 错误使用 `Errorw("action: "+err.Error())` 格式，有结构化字段时追加 Field
- `err=` 作为结构化字段的用法已废弃，改为拼入消息文本
- 单条 `subscribed topic=%s` 打印在 `subscribe()` 内部（Info 级），调用方不得重复打印
- `restore subscriptions done` 为批量摘要，仅 `onConnect` 中打印一次

### 4. Good/Base/Bad Cases

- Good：`[mqtt] connected client=dji-cloud-001`
- Good：`[mqtt] subscribed topic=thing/product/+/events`
- Good：`[mqtt] restore subscriptions done subscribed=2 skipped=0`
- Good：`[mqtt] connection lost: dial tcp 1.2.3.4:1883: connection refused`
- Bad：`[mqtt] Connection successful, client=dji-cloud-001`（大写动作词）
- Bad：`[mqtt] connection lost err=dial tcp...`（err= 应拼入消息文本）
- Bad：`fmt.Errorf("[mqtt] connect failed: %w", err)`（fmt.Errorf 不带 `[xxx]` 前缀）

## Scenario: 高频路径按 context 关闭 GORM SQL trace

### 1. Scope / Trigger

- Trigger: 高频采集、MQTT、Socket、SSE、Kafka 等路径会产生大量 GORM `Trace` SQL 日志，但仍需要保留业务错误日志和其他低频 SQL 诊断能力。

### 2. Signatures

- Context helper: `gormx.WithoutSQLTrace(ctx context.Context) context.Context`。
- Logger boundary: `func (c *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error)` 在检测到该 context 标记且 SQL 无错误、未超慢阈值时直接返回，不调用 `fc()`。
- Service config example: `Telemetry.DisableOsdSQLTrace bool` 用于服务私有配置，不改全局 `gormx.Config.LogLevel`。

### 3. Contracts

- `gormx.WithoutSQLTrace(ctx)` 只压制无错误且未超慢阈值的普通 SQL trace，不能压制真实 SQL error trace 或 slow SQL trace；错误和慢日志必须一直打印，避免高频路径静默丢失数据库错误或性能异常。
- GORM `Trace` 日志不是互斥分支：同一条 SQL 如果同时满足 info 和 slow，应各打一条 info trace 和 slow trace；如果同时满足 error 和 slow，应各打一条 error trace 和 slow trace。
- 业务层 `logx.WithContext(ctx).Errorf(...)` 不受影响，调用方仍需在边界层记录业务失败。
- 高频路径必须在进入 DB helper 前包裹 context，例如 OSD 写库调用 `FirstOrCreate` 前处理。
- 服务级配置默认保持旧行为；只有配置显式开启时才对目标高频路径使用该 context。

### 4. Validation & Error Matrix

- Context 未设置 + `LogLevel=info` -> `Trace` 可输出正常 SQL。
- Context 未设置 + 慢 SQL + `LogLevel=warn` -> `Trace` 可输出慢 SQL。
- Context 设置 `WithoutSQLTrace` + SQL 成功且不慢 -> `Trace` 不调用 `fc()`，不输出 SQL trace。
- Context 设置 `WithoutSQLTrace` + 慢 SQL -> `Trace` 仍调用 `fc()` 并输出 slow trace。
- Context 设置 `WithoutSQLTrace` + SQL error -> `Trace` 仍调用 `fc()` 并输出 error trace。
- `LogLevel=info` + 慢 SQL -> 同时输出普通 info trace 和 slow trace。
- `LogLevel=error|warn|info` + 慢 SQL -> 输出 slow trace；slow 不依赖 warn 级别才打印。
- DB helper 返回 error -> GORM error trace 会保留；调用方仍按业务边界记录必要错误上下文，避免只依赖 SQL 文本理解业务失败。

### 5. Good/Base/Bad Cases

- Good: OSD 这类高频写库路径通过服务配置开启 `DisableOsdSQLTrace`，只在该 handler 内 `ctx = gormx.WithoutSQLTrace(ctx)`。
- Good: `WithoutSQLTrace(ctx)` 后写库失败仍输出 `[gorm] ... error: ...`。
- Good: `WithoutSQLTrace(ctx)` 后慢 SQL 仍输出 `[gorm] ... [SLOW] ...`。
- Good: `LogLevel=info` 且 SQL 超慢时，日志中同时出现普通 SQL trace 和 `[SLOW]` trace。
- Base: 临时排查某个高频路径时关闭配置，让 `gormx.Config.LogLevel` 恢复输出 SQL trace。
- Bad: 为了压低单一路径日志把 `DB.LogLevel` 全局改成 `silent`，导致其他数据库错误和慢 SQL 诊断一起丢失。
- Bad: `Trace` 在检测到 `WithoutSQLTrace` 后无条件 return，导致 SQL error trace 被吞掉。

### 6. Tests Required

- Unit: `TestGormLoggerTraceSkipsSuccessfulSQLWhenContextDisablesTrace` 断言设置 context 后成功 SQL 不调用 `fc()`。
- Unit: `TestGormLoggerTraceLogsErrorWhenContextDisablesTrace` 断言设置 context 后 SQL error 仍调用 `fc()`。
- Unit: `TestGormLoggerTraceLogsSlowSQLWhenContextDisablesTrace` 断言设置 context 后 slow SQL 仍调用 `fc()`。
- Unit: `TestGormLoggerTraceLogsInfoAndSlowWhenInfoLevelSQLIsSlow` 断言 info 级慢 SQL 输出两条日志。
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
