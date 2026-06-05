# ieccaller 集群指令 ACK 回传实施计划

## 检查清单

1. 编辑前加载 `app/ieccaller`、`common/iec104/client`、Kafka 使用相关的 Trellis backend/guides specs。
2. 在 `internal/config/config.go` 和示例 YAML 中新增 broadcast ACK topic 配置和默认值。
3. 在 service context 中新增 broadcast ACK pusher 和基于 traceId 的 Kafka replypool，并在 `ServiceContext.Close` 中关闭资源。
4. 为 broadcast request reply metadata 和 ACK reply 增加消息契约。若 `BroadcastBody` 已在 `common/iec104/types` 中定义，优先把兼容结构放在同一类型包中。
5. 扩展 `PushPbBroadcast` 或新增配套 helper：生成 traceId、注册 replypool wait、发布 broadcast request、等待 ACK、把 reply body unmarshal 到 typed response。
6. 在 `app/ieccaller/ieccaller.go` 中按现有 broadcast consumer 模式接入 broadcast ACK topic consumer。
7. 新增 broadcast ACK consumer，按 traceId resolve service context 中的 Kafka replypool wait。
8. 更新 ACK 型 gRPC command logic：当本地没有目标 client 且处于 cluster 模式时，使用支持 reply 的 broadcast helper。
9. 更新 `kafka.Broadcast.Consume` 中 ACK 型 method case：执行本地指令时传入 `client.WithAck()`，通过 onASDU 拿到 IEC104 ACK 响应，构造与本地路径一致的 response JSON，并向 broadcast ACK topic 发布带同一 traceId 的 success/error 响应广播。
10. 保持非 ACK broadcast case 的 fire-and-forget 行为，包括通用 `SendCommand`。
11. 新增或更新测试，覆盖 service context reply 行为、ACK consumer resolve 行为，以及一个或多个 broadcast command case。
12. 先运行目标 Go 测试，再在可行时扩大测试范围。

## 验证命令

```bash
go test ./app/ieccaller/internal/svc ./app/ieccaller/kafka ./app/ieccaller/internal/logic ./common/iec104/client
```

如果不修改 protobuf，则不需要重新生成 proto 代码。

## 高风险文件

- `app/ieccaller/internal/svc/servicecontext.go`
- `app/ieccaller/kafka/broadcast.go`
- `app/ieccaller/ieccaller.go`
- `app/ieccaller/internal/logic/send*command*logic.go`
- `common/iec104/types` 中定义 broadcast payload 的文件

## Review Gates

- `task.py start` 前 review 最终 PRD/design/implement。
- 实现后运行 `trellis-check`。
- 如果本次引入了可复用 Kafka request/reply 约定，完成后通过 `trellis-update-spec` 更新项目规范。
