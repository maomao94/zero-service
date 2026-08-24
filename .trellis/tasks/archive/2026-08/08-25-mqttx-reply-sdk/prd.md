# 重构 mqttx reply 逻辑，抽取通用广播 ack SDK

## Goal

把 oryxserver 与 ieccaller 两处几乎相同的 MQTT 广播集群 ack 逻辑（用 MQTT 做 gRPC 广播集群 ack）抽取为 `mqttx` 的通用 SDK 能力（新子包 `common/mqttx/broadcast`），并参考 djisdk 的主题定义方式重构主题定义（函数 + 成对 Pattern + 三要素注释 + 分组 + 一致性测试）；因**该功能未上线**，同时把 ack 通道线格式优化为 djisdk 同风格的 `{prefix}/broadcast_reply/{id}`。

## 背景与已确认事实（来自 research/research-sizing.md）

- oryxserver：`app/oryxserver/internal/relay/broadcast.go`（裸字符串主题常量 `BroadcastTopic="oryx/relay/broadcast"`、`BroadcastAckTopicPrefix`、`RelayBody`/`RelayAckBody`、`DecodeRelayAck`）、`internal/svc/servicecontext.go`（初始化模板 + fire-and-forget 发送）、`mqtt/broadcast.go`（单 method 消费，found==false 不回 ack，无 errorKind）。
- ieccaller：`app/ieccaller/internal/svc/servicecontext.go`（内联主题串 `"iec/broadcast"`/拼接 ack、初始化模板、`PushPbBroadcast(WithAck)`、`decodeBroadcastAck`、`broadcastAckError`）、`mqtt/broadcast.go`（14 method 大 switch、`publishAckReply` 带 errorKind 归一）、body 在 `common/iec104/types/types.go`。
- 二者同仓库单模块 `zero-service`（go 1.26.0）；`common/mqttx` 已是 4 app + djisdk 复用的协议中立底座（client/ReplyRouter/RequestReply）；重复集中在协议层（主题常量、body、ack 解码器、初始化模板、发布/回包，约 110 行）。
- **该广播集群功能未在任何生产环境部署**：ieccaller 生产环境未启用集群模式（未使用集群广播），oryxserver 从未上线 → **无任何 wire 兼容约束**，旧主题值（含 `broadcast-ack`）、旧 body 均无需保留。
- **"djsdk" 实为 `djisdk`**（`common/djisdk/topic.go`）：每主题一个构造函数 + 成对 `XxxReplyTopicPattern`/`+` 通配 + 路径格式/方向/用途三要素注释 + `====` 分组 + doc.go + `topic_test.go` 一致性测试（`TestTopicPatternsMatchConcreteTopics`）；回复通道用 `_reply` 后缀（`services_reply` 等）。
- `mock-service` 仓库有 mqttx/djisdk 旧拷贝（独立仓库，非本任务范围）。

## Requirements

- R1: 新增 `common/mqttx/broadcast` 子包，承载广播协议泛化能力：djisdk 风格主题定义、协议中立 `BroadcastBody`/`BroadcastAckBody`（只含关联/路由字段，业务数据一律走 opaque `Body`/`ResponseBody`）、通用 ack 解码器、errorKind 归一（内置 timeout/duplicate/unknown，业务类别注册）、消费分发骨架（method→executor 注册 + 防回环 + ack 回发）、发送封装（fire-and-forget 与等待 ack 两种模式，只暴露 method + payload）。
- R2: 主题定义采用 djisdk 风格；导出的主题函数与值：
  - `BroadcastTopic(prefix)` = `{prefix}/broadcast`（同值 `BroadcastTopicPattern(prefix)`）
  - `BroadcastAckTopic(prefix, id)` = `{prefix}/broadcast_reply/{id}`；`BroadcastAckTopicPattern(prefix)` = `{prefix}/broadcast_reply/+`
  - 前缀由 app 配置：oryxserver 用 `oryx/server`（容器级命名，不带 relay 业务属性），ieccaller 用 `iec`。
- R3: 业务 method 分发留在各 app 的注册式 executor（oryx RelayManager.Stop / iec104 14 个命令 + ClearPointMappingCache）；SDK 不感知业务 payload 结构（payload 格式由各 executor 约定：protojson / taskId JSON）。
- R4: errorKind 归一化进骨架：SDK 仅内置 `timeout`/`duplicate`/`unknown`；`iec_rejected` 由 ieccaller 在 app 内声明常量并注册——ieccaller 现有错误类还原行为（`ErrReplyExpired`/`ErrDuplicateID`/`CommandRejectedError`）保持等价，oryxserver 顺带获得 errorKind 能力。
- R5: 未上线，允许线格式优化（`broadcast-ack/` → `broadcast_reply/`）；通用层不定义任何业务字段（taskId 等）与业务常量（iec_rejected）。

## Acceptance Criteria

- [ ] AC1: `common/mqttx/broadcast` 包存在并通过 `go build ./...`、`go vet ./...`；`topic_test.go` 一致性测试断言 4 个主题函数的输出值与相互匹配（含 `oryx/server/broadcast`、`iec/broadcast_reply/xyz` 具体值）。
- [ ] AC2: ieccaller 迁移完成：`servicecontext.go` 中内联主题串、`decodeBroadcastAck`、ReplyRouter 初始化模板、`pushBroadcast`/`broadcastAckError` 私有实现均删除；14 个 method 以 executor 注册；`ClearPointMappingCache` 走无 ack（`ErrSkipAck`）语义不变；`common/iec104/types` 的 `BroadcastBody`/`BroadcastAckBody` 删除前确认无其他引用（有则重导出别名）；`iec_rejected` 为 app 内常量和注册，不在 SDK。
- [ ] AC3: oryxserver 迁移完成：`BroadcastTopic`/`BroadcastAckTopicPrefix` 裸常量、`RelayBody`/`RelayAckBody`、`DecodeRelayAck` 删除；`PublishRelayStop` 等价改走 fire-and-forget（method + payload）；found==false 不回 ack 语义保留；前缀用 `oryx/server`。
- [ ] AC4: 发送/等待路径行为等价且协议隔离：`BroadcastReply` 等待 ack（失败按 errorKind 还原领域错误，成功只返回业务结果字节）；`Broadcast` 不等待即可返回；调用方只见 method + payload，不构造/拼接任何主题或协议字段。
- [ ] AC5: 消费分发骨架含：自身 ack 回环忽略、未注册 method 回 `Success=false, ErrorKind=KindUnknown`、executor 错误回 ack 带归一化 kind。
- [ ] AC6: 单测覆盖：主题一致性、分发骨架（回环/未注册/成功/失败/ErrSkipAck）、errorKind 双向回环；全仓 `go test ./...` 通过。
- [ ] AC7: grep 确认历史符号无残留（`BroadcastAckTopicPrefix`、`broadcast-ack/%s`、`DecodeRelayAck`、`decodeBroadcastAck`、`publishAckReply`）。

## Out of Scope

- `mock-service` 仓库旧拷贝同步（独立仓库，不处理）。
- 各 app 业务 method 分发逻辑的语义改动（iec104 命令执行、relay stop 判定）。
- `common/mqttx` 根包现有 API 改动。
- 主题值里加入 method 段（method 继续放 body，避免 ack 关联复杂化）。

## Open Questions

无。规划相关的用户决策均已确定（吸收范围、主题定义方式、ack 通道改名、前缀参数化）。
