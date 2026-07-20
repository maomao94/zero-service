# IEC 104 链路追踪规范

> ieccaller ASDU 消息 OTel trace propagation 的 canonical source。

## When to read

- 修改 `common/iec104/trace.go` 或 `app/ieccaller/internal/svc/servicecontext.go` 的 PushASDU。
- 新增 ASDU 下行消费者（Kafka/MQTT/gRPC）需要还原 trace context。
- 新增 chunk 批量推送路径或修改 `PushChunkAsdu` 调用链。

## 全链路文件

| 文件 | 职责 |
|------|------|
| `common/iec104/trace.go` | `StartRecvSpan`, `StartForwardSpan`, `TraceHeaders`, `TraceIdFromContext`, `ExtractTraceHeaders` |
| `common/iec104/types/types.go` | `MsgBody.Headers` + `MsgBody.TraceId` 字段 |
| `app/ieccaller/internal/svc/servicecontext.go` | `PushASDU`: marshal 前设 `Headers`/`TraceId`，三路推送 |
| `common/iec104/client/handle.go` | 每个 IEC104 handler 入口启动 `StartRecvSpan`；`IecLogContext` 注入统一日志字段和普通 `stationId` ctx value |
| `app/ieccaller/internal/iec/clienthandler.go` | `OnASDU` 使用 handler 传入的 ctx 异步处理 ASDU 并调用 `PushASDU` |
| `facade/streamevent/streamevent.proto` | `MsgBody.traceId` + `MsgBody.headers` |

## 核心模式

### 1. TraceHeaders — 提取 trace 上下文

```go
// common/iec104/trace.go
func TraceHeaders(ctx context.Context) (map[string]string, string) {
    headers := make(map[string]string)
    tracex.Inject(ctx, tracex.NewCarrier(headers))
    return headers, ztrace.TraceIDFromContext(ctx)
}
```

- `Inject` 从 ctx 中的 span context 提取 W3C `traceparent`/`tracestate`。
- `TraceIDFromContext` 返回 span 的 trace ID（与 headers.traceparent 中的一致）。
- 无 span context 时 headers 为空，调用方不做额外处理。

### 2. PushASDU — marshal 前设字段

```go
// app/ieccaller/internal/svc/servicecontext.go
data.Headers, data.TraceId = iec104.TraceHeaders(ctx)
byteData, err := jsonx.Marshal(data)
```

- 在 `jsonx.Marshal(data)` 前设置 `MsgBody.Headers` 和 `MsgBody.TraceId`。
- struct 序列化保序，`Headers`/`TraceId` 出现在 JSON 末尾。
- `omitempty` 使得无 trace 时字段不出现在 JSON 中。
- **不要在 marshal 后做字节拼接或二次 JSON 修改。**

### 3. 三路推送 — 同一 payload

```go
mr.FinishVoid(
    func() { svc.KafkaASDUPusher.PushWithKey(spanCtx, key, string(byteData)) }, // native Kafka headers + JSON headers
    func() { svc.MqttClient.Publish(spanCtx, topic, byteData) },                 // JSON headers only
    func() { svc.ChunkAsduPusher.Write(string(byteData)) },                      // JSON headers into async queue
)
```

- Kafka 路径用 `PushWithKey`，go-queue 自动在 Kafka message headers 写入 traceparent。JSON payload 中也有 headers/traceId 作为 fallback。
- MQTT 路径用 `Publish`（不是 `PublishWithTrace`），保持原始 JSON，避免 envelope 包装破坏现有消费者。
- Chunk 路径写 byteData 到异步缓冲队列，headers/traceId 随 JSON 传入。

### 4. Handler 入口上下文 — span、日志字段、stationId

```go
// common/iec104/client/handle.go
ctx, span := iec104.StartRecvSpan(context.Background(), rxAsdu, h.traceOpts)
defer span.End()
ctx = IecLogContext(ctx, rxAsdu, h.traceOpts)
return h.call.OnASDU(ctx, rxAsdu)
```

- 所有 `ClientHandler` 方法都必须从 handler 入口创建 recv span，再把 ctx 传给对应 `ASDUCall` 方法。
- `IecLogContext` 负责写入 go-zero log fields：`host`、`port`、`stationId`、`iecType`、`typeId`、`coa`、`cot`、`cotCause`、`isNegative`。
- 不要手动添加 `logx.Field("traceId", ...)`；go-zero/logx 已支持基于 ctx 输出 trace。
- `IecLogContext` 还必须写入普通 context value：`context.WithValue(ctx, "stationId", traceOpts.StationId)`。这是业务读取契约，不是日志字段。
- `app/ieccaller/internal/svc/servicecontext.go:PushASDU` 会通过 `ctx.Value("stationId")` 做点位映射查询；缺失时才 fallback 到 `util.GenerateStationId(data.Host, data.Port)`。
- `logx.ContextWithFields` 不等价于 `context.WithValue`，不能依赖 log field 供业务代码读取。

> **Span lifecycle note**: `ASDUHandler` 调用的 `OnASDU` 会通过 `TaskRunner.Schedule` 异步处理。handler 入口 `defer span.End()` 表示 recv span 覆盖同步接收/投递阶段；goroutine 内仍可用 ctx 提取 trace headers，但后续 Kafka/MQTT/chunk push 耗时不会计入该 span duration。

### 5. Chunk 回调 — 独立 root span

```go
// ieccaller
ctx, span := iec104.StartForwardSpan(context.Background())
// iecstash（无 span，不 import iec104）
streamEventCli.PushChunkAsdu(context.Background(), ...)
```

- Chunk 是压缩合并操作，一条 chunk 包含多个不同 trace 的消息。
- **不要**从任意一条 msg 提取 headers 作为批次 parent——不同 trace 无法代表。
- `StartForwardSpan(context.Background())` 创建独立 Producer root span。

### 6. 消费者还原 trace

```go
// common/iec104/trace.go
func ExtractTraceHeaders(ctx context.Context, payload string) context.Context {
    var obj struct {
        Headers map[string]string `json:"headers"`
    }
    if err := json.Unmarshal([]byte(payload), &obj); err != nil || len(obj.Headers) == 0 {
        return ctx
    }
    return tracex.Extract(ctx, tracex.NewCarrier(obj.Headers))
}
```

- 消费端从 JSON `headers` 字段还原 OTel propagation context。
- Kafka 消费端 go-queue 已从 Kafka headers 还原 trace，无需用此函数。
- MQTT 和 gRPC `PushChunkAsdu` 服务端可用此函数为每条 msg 还原链路。

## 边界规则

| 规则 | 说明 |
|------|------|
| `metaData` 不含 trace 字段 | `traceparent`、`tracestate`、`traceId` 不写入 `MsgBody.MetaData` |
| `headers` + `traceId` 是传输字段 | 位于 JSON 顶层，由 `Inject`/`Extract` 成对处理 |
| 不使用 `PublishWithTrace` | ASDU MQTT 路径保留原始 JSON，不包 envelope |
| Chunk span 独立 | chunk 批次的 span 不从单条 msg 继承 parent |
| `copyMetaData` 保护配置 | `newMsgBody` 每次拷贝 `ClientConfig.MetaData`，避免并发写 |
| `stationId` 普通 ctx value | handler 入口写入，`PushASDU` 读取；log field 不能替代业务 ctx value |
| `traceId` 日志 | 不手写 `logx.Field("traceId", ...)`，由 go-zero/logx trace 支持输出 |

## Good / Base / Bad

- **Good**: `PushASDU` 中 `data.Headers, data.TraceId = iec104.TraceHeaders(ctx)` 一次赋值，struct marshal 保序。
- **Base**: marshal 后做 JSON 字节拼接（额外一次 `json.Marshal(headers)`，但无 `Unmarshal` 开销）。
- **Bad**: `map[string]interface{}` 先 `Unmarshal` 再 `Marshal`（乱序 + 双倍开销）。chunk 回调从第一条 msg 提取 headers 作为 batch parent（不同 trace 错配）。

## 测试

- Unit: `TraceHeaders(ctx)` 返回 `traceparent` 和 `traceId` 与 span context 一致。
- Unit: `newMsgBody` 不把 traceId 写入 `MetaData`，`copyMetaData` 隔离。
- Unit: `IecLogContext` 输出统一 IEC104 fields，并保留普通 `stationId` ctx value 供 `PushASDU` 读取。
