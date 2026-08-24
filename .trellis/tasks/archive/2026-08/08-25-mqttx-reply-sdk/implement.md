# 实施计划：mqttx 广播层 SDK

按 checkpoint 顺序执行，每步可独立构建、可回滚。commit 建议按 checkpoint 拆分。

## 前置检查

```bash
cd /Users/hehanpeng/GolandProjects/zero-service
go build ./... && go vet ./...
```

## Checkpoint 1: 新增 `common/mqttx/broadcast` 包（纯新增，无引用方变化）

- `topic.go`：
  - `BroadcastTopic(prefix string) string` → `{prefix}/broadcast`（构造 + 三要素注释：路径格式/方向/用途）
  - `BroadcastTopicPattern(prefix string) string` → 同值（订阅具体主题的锚点，保持一致性测试）
  - `BroadcastAckTopic(prefix, instanceID string) string` → `{prefix}/broadcast_reply/{instanceID}`
  - `BroadcastAckTopicPattern(prefix string) string` → `{prefix}/broadcast_reply/+`
  - `joinPrefix(parts ...string) string` 小工具；`====` 分组注释 + doc.go
- `topic_test.go`：仿 djisdk `TestTopicPatternsMatchConcreteTopics`——Pattern 与具体主题对齐（`BroadcastTopicPattern(p)` 匹配 `BroadcastTopic(p)`；`BroadcastAckTopicPattern(p)` 匹配任意 instanceID 的 `BroadcastAckTopic(p, id)`；具体值断言：oryx/relay/broadcast、oryx/relay/broadcast_reply/xyz、iec/broadcast、iec/broadcast_reply/xyz）
- `body.go`：`BroadcastBody`（Tid/AckTopic/Method/Body，json tag 与现状一致；**不含任何业务字段**，TaskId 移除，业务数据走 Body opaque payload）+ `BroadcastAckBody`（Tid/Method/Success/ResponseBody/Error/ErrorKind）
- `errors.go`：
  - kind 常量：`KindTimeout="timeout"`、`KindDuplicate="duplicate"`、`KindUnknown="unknown"`
  - `ErrSkipAck` 哨兵（executor 语义：成功但不回 ack，对齐 oryxserver found==false 与 ieccaller ClearPointMappingCache）
  - `RegisterErrorKind(src error, kind string, dst func(msg string) error)` + `NormalizeErrorKind(err)` + `ErrorFromKind(kind, msg string) error`（内置 timeout/duplicate/unknown 两条默认映射）
- `ack.go`：`decodeAck(ctx, payload, topic, template)` 通用解码器（= DecodeRelayAck/decodeBroadcastAck 并集，Tid 空 → `mqttx.ErrEmptyReplyTid`）；`NewAckReplyRouter(ttl, name)`（`mqttx.NewReplyRouter[*BroadcastAckBody]` + 上述解码器）
- `client.go`：`Broadcaster` 接口 + `NewBroadcaster(c mqttx.Client, instanceID string, opts...)`：
  - 字段：prefix、instanceID、replyTTL（默认 10s，`WithReplyTTL` 可配）、executor map
  - `Broadcast(ctx, method string, payload []byte)`：fire-and-forget；SDK 内部构造 `BroadcastBody{Tid 生成, AckTopic 自动填, Method}` → `PublishWithTrace`（对齐 oryxserver PublishRelayStop）
  - `BroadcastReply(ctx, method string, payload []byte, ttl)`：内部生成 Tid + 构造 body → `mqttx.RequestReply[*BroadcastAckBody]`；Success=false 时 `ErrorFromKind` 还原错误；成功返回 `[]byte(ack.ResponseBody)`（对齐 ieccaller PushPbBroadcastWithAck）
  - `AddExecutor(method, Executor)`；`Executor = func(ctx, method string, payload []byte) ([]byte, error)`（调协议字段与业务隔离；骨架构造 ack 时填 Success/Error/ErrorKind/ResponseBody）
  - `ConsumeBroadcast`：方法签名为 `mqttx.ConsumeHandler` 可传给 `mqttx.Client.AddHandlerFunc(BroadcastTopicPattern(prefix), ...)`；流程：unmarshal → 自身 ack 回环忽略（`body.AckTopic == BroadcastAckTopic(prefix, instanceID)`）→ executor 分发（传 method+Body 字节）→ 未注册 method 回 ack `Success=false, ErrorKind=KindUnknown` → 成功：executor 返回 `ErrSkipAck` 则不回，否则回 ack（ResponseBody=结果）→ 失败：`NormalizeErrorKind` 回 ack
- `dispatcher_test.go` / `client_test.go`：mock mqttx.Client（如已有测试基建则复用，否则用 paho 内存 broker 或接口 mock）验证：回环忽略、未注册 method、成功/失败 ack 回发（含 errorKind）、ErrSkipAck、BroadcastReply 还原错误

验证：

```bash
go build ./... && go vet ./common/mqttx/broadcast/...
go test ./common/mqttx/broadcast/... -cover
```

## Checkpoint 2: ieccaller 迁移

- `app/ieccaller/internal/svc/servicecontext.go`：
  - 删除内联主题串（`bytecode:70-71`）、`decodeBroadcastAck`（390-410）、ReplyRouter 初始化模板（69-80）
  - 改：`NewBroadcaster(svcCtx.MqttClient, svcCtx.broadcastInstanceId, broadcast.WithPrefix("iec"), ...)` + app 内 `const errKindIECRejected = "iec_rejected"` 后用 `RegisterErrorKind(client.CommandRejectedError{}, errKindIECRejected, ...)` 注册（按 CommandRejectedError 实际类型/构造方式确认）
  - `PushPbBroadcast` / `PushPbBroadcastWithAck`（276-310）改为调用 `Broadcast` / `BroadcastReply`（method + `protojson.Marshal(in)` payload），删除私有实现 `pushBroadcast`/`broadcastAckError`/`broadcastAckReply`；结果 `protojson.Unmarshal(respBody, res)` 留在调用处
- `app/ieccaller/mqtt/broadcast.go`：Consume 的大 switch 拆为 14 个 executor 注册（protojson 反序列化留在各 executor 内），`publishAckReply` 模板删除（errorKind 归一进骨架）；`ClearPointMappingCache` 用 `ErrSkipAck` 语义
- `app/ieccaller/ieccaller.go:95-102`：消费注册改 `AddHandlerFunc(broadcast.BroadcastTopicPattern("iec"), broadcster.ConsumeBroadcast)`（或 builder 暴露）
- `common/iec104/types`：grep `BroadcastBody|BroadcastAckBody` 确认无其他引用后删除这两个结构体（有引用则保留重导出别名）
- 迁移前后对比测试：以现有 `servicecontext_test.go` 为基，补充/保留 ack 语义用例

验证：

```bash
go build ./... && go vet ./...
go test ./app/ieccaller/... ./common/mqttx/... ./common/iec104/...
```

## Checkpoint 3: oryxserver 迁移

- `app/oryxserver/internal/relay/broadcast.go`：删除 `BroadcastTopic`/`BroadcastAckTopicPrefix`/`MethodStreamRelayStop`?（Method 是 gRPC full method 常量，保留在 app 内或迁到 SDK 的 method 约定层；建议保留 app 内，SDK 只做分发）、`RelayBody`/`RelayAckBody`、`DecodeRelayAck`
- `app/oryxserver/internal/svc/servicecontext.go:51-63`：`NewBroadcaster(..., broadcast.WithPrefix(broadcast.Prefix("oryx", "server")))`（容器级前缀，不带 relay 业务属性）；`PublishRelayStop`（96-117）→ `Broadcast(ctx, MethodStreamRelayStop, taskPayload)`（taskId 编码格式由 executor 约定，如 JSON）
- `app/oryxserver/mqtt/broadcast.go`：Consume 改为注册单一 executor（`MethodStreamRelayStop` → RelayManager.Stop；found==false → `ErrSkipAck`）

验证：

```bash
go build ./... && go vet ./...
go test ./app/oryxserver/... ./common/mqttx/...
```

## Checkpoint 4: 收尾

- grep 确认无历史常量残留（`BroadcastAckTopicPrefix`、`broadcast-ack/%s` 拼接、`DecodeRelayAck`、`decodeBroadcastAck`）
- `go test ./...`（全仓）；`go build ./...`
- 更新 spec（如 `backend` 层规范需补充 broadcast 包规范）→ Phase 3.3 处理

## 风险文件 / 回滚点

- `common/mqttx/broadcast/*`（新增）：回滚=删包
- `app/ieccaller/internal/svc/servicecontext.go`、`app/ieccaller/mqtt/broadcast.go`、`common/iec104/types/types.go`：回滚=git checkout 前一 commit（wire 不变，集群安全）
- `app/oryxserver/internal/relay/broadcast.go`、`app/oryxserver/internal/svc/servicecontext.go`、`app/oryxserver/mqtt/broadcast.go`：同上

## 检查点验证命令汇总

| 阶段 | 命令 |
|---|---|
| 每步 | `go build ./...` + `go vet ./...` |
| C1 | `go test ./common/mqttx/broadcast/... -cover` |
| C2 | `go test ./app/ieccaller/... ./common/mqttx/... ./common/iec104/...` |
| C3 | `go test ./app/oryxserver/... ./common/mqttx/...` |
| C4 | `go test ./...` |
