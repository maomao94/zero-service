# Log Dump 日志汇聚规范

## 适用范围

修改 `app/logdump` 或日志汇聚/转发相关功能时读取。

## 服务定位

logdump 是一个极简 zRPC 服务，作为 gRPC-to-logx 桥接器：接收上游服务通过 gRPC 推送的日志条目，经过字段白名单过滤后写入 go-zero 结构化日志系统。

依据：`app/logdump/internal/logic/pushloglogic.go`、`app/logdump/internal/config/config.go`。

## 入口与配置

- `logdump.go` 标准 go-zero zRPC 入口，含 Nacos 服务注册和 `interceptor.LoggerInterceptor`。
- `config.Config` 嵌入 `zrpc.RpcServerConf`，额外包含：
  - `NacosConfig` — 可选 Nacos 配置
  - `ExtraFields []string` — 可选，允许作为结构化字段的额外字段名白名单
- `ServiceContext` 极简：仅持有 `Config`，无 DB、无外部客户端。

依据：`app/logdump/logdump.go`、`app/logdump/internal/svc/servicecontext.go`。

## RPC 方法

2 个 RPC 方法 (`internal/server/logdumpserver.go`)：

| RPC | 功能 |
|-----|------|
| Ping | 健康检查 |
| PushLog | 接收 `PushLogReq`（含 `[]*LogEntry`），逐条写入 go-zero logx |

## PushLog 逻辑

### 字段处理

每条 `LogEntry` 的处理流程：

1. **必选字段**: `seq` 和 `service` 始终作为结构化字段（`logx.Field`）添加。
2. **白名单字段**: 仅当 `ExtraFields` 配置中包含某字段名时，该字段才作为独立结构化字段添加。
3. **人类可读格式**: **所有**额外字段（无论是否在白名单中）都拼接为 `key1=val1, key2=val2` 字符串。
4. **消息格式**: `[{service}] {message} | key1=val1, key2=val2`
5. **日志级别路由**: `LogLevel_ERROR` → `Logger.Error()`，其他 → `Logger.Info()`
6. **返回**: 空的 `PushLogRes`

依据：`app/logdump/internal/logic/pushloglogic.go`。

### 关键约定

- **单个 gRPC 调用中的多条日志独立写入**，不聚合。
- **配置白名单决定哪些字段有结构化表示**，但所有字段都出现在消息正文中。
- **`ExtraFields` 为空时，所有额外字段仅出现在消息正文中**，不作为结构化字段。
- 日志写入失败不返回 gRPC error（已记录到 logx）。

依据：`app/logdump/internal/logic/pushloglogic.go`。

## 反模式

- 在 PushLog 中聚合多条日志为一条（丢失独立性和时间精度）。
- 绕过 `ExtraFields` 白名单将所有字段都设为结构化字段（导致字段爆炸）。
- 在 ServiceContext 中添加外部依赖（logdump 应该保持极简）。
- 将 PushLog 的日志写入失败映射为 gRPC error。
- 修改消息格式不保持向后兼容。

## 验证

- 验证 `ExtraFields` 白名单正确过滤结构化字段。
- 验证所有字段都出现在消息正文中。
- 验证日志级别路由（Error vs Info）。
- 验证空 `ExtraFields` 配置时的行为。
- 验证多条日志条目独立写入不互相污染。
