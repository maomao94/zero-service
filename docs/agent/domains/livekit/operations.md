# LiveKit 平台与运维

## 版本与依赖

- 审计基线为 LiveKit Server `v1.13.6`、`github.com/livekit/server-sdk-go/v2 v2.18.1`。
- Protocol 版本由选定 SDK 的 `go.mod` 解析；不得手工猜测或截断 pseudo-version。
- 升级后重新执行核心示例的 `go mod tidy`、`go test ./...`，并检查生成字段、枚举、oneof、配置和官方链接。

## Token 与授权

- 业务服务使用 `auth.NewAccessToken` 签发短期、指定房间、最小权限的 join token；管理 token 不返回客户端。`livekitx` 提供 `NewJoinToken(opts JoinTokenOptions)`（自定义 grant 场景）。
- `VideoGrant` 的房间、创建、管理、录制、Ingress、Agent、发布/订阅和数据字段必须以锁定 Protocol 生成类型核对；`CanPublish`/`CanSubscribe` 是 `*bool`，用 `grant.SetCanPublish(bool)` 设置，不能直接结构体字面量赋值。
- 聊天/Data 链路 Token 必须设置 `CanPublishData`（`grant.SetCanPublishData(true)`），否则 Data 发布被服务端拒绝。
- SIP 使用 `SIPGrant`，不得写成 `VideoGrant` 字段。
- API key/secret、Webhook signing key、SIP 密码只能来自安全配置。
- `auth.NewAccessToken(key, secret).ToJWT()` 在 key 为空时报错；Token 测试断言用 `auth.ParseAPIToken(token)` + `verifier.Verify(secret)`，不要在测试中输出 token/secret。

## 状态与 Webhook

LiveKit 拥有房间、参与者、轨道和录制运行态；zero-service 拥有用户、会议单据、角色授权、审计和业务状态。Webhook 只触发同步或对账，不授予权限，也不替代管理 API。

## 实时连接与回调

- 业务直接使用 SDK 原生 `*lksdk.RoomCallback`，通过 `lksdk.NewRoom(callback)` 构造，`room.JoinWithContext` 加入。
- `JoinWithContext` 成功返回即"已连接"（SDK 原生无 connected 回调）；断开由业务调 `room.Disconnect()`。
- 聊天识别是业务职责：`*livekit.ChatMessage`（文本）与 `UserDataPacket`（业务自定义 topic）都在 `OnDataPacket` 到达。
- 服务端 RPC（`client.Room().PerformRpc()`）是服务端→单个客户端的定向请求-响应。
- SendData 与 PerformRpc 选型：只通知不关心结果→ `SendData`；要客户端执行并返回结果→ `PerformRpc`。
- 服务端 RPC 错误诊断矩阵（`rpc_self_test.go` 实测验证）：

| 错误 | 原因 | 排查方向 |
|------|------|---------|
| `RpcError 1400: Method not supported at destination` | 目标客户端**在线但未注册**该 method | 检查目标端 `registerRpcMethod` 是否执行（刷新页面后注册会丢失） |
| `no response from servers` | 目标参与者**不存在或已离线** | 用 ListParticipants 确认目标在线 |
| `ResponseTimeout` 类错误 | 目标在线、已注册，但 handler 未在时限内返回 | handler 阻塞（如等用户输入）或网络问题 |

- 目标参与者可以是调用方自身（自发自收），LiveKit 服务端按 identity 路由，不做 caller≠destination 校验；排查 RPC 失败时不要怀疑"自己发给自己"。
- 管理 API 方法名以锁定 protocol 源码核对：静音是 `MutePublishedTrack`；结束会议统一用 `DeleteRoom`。

## 错误、生命周期与反模式

- 每个外部调用设置 context 超时，保留 Twirp code/message。
- ServiceContext 复用 client，关闭服务时释放。
- 不在请求中重复创建 API client，不硬编码 secret，不在公共包固化业务 grant。

## 部署与验证

生产使用可信 TLS 的 HTTPS/WSS；WebRTC UDP/TCP、ICE、NAT、公网地址、TURN、WebSocket、Redis、多节点负载均衡和独立 Egress/Ingress/SIP/Agent 进程按官方当前配置逐项验收。开放 API 端口不等于媒体链路可用。

验证至少包括：临时 module 锁定稳定 SDK 后 `go mod tidy` 与 `go test ./...`；Token/Webhook/API 的成功、权限失败、超时、重复和边界测试；`git diff --check` 以及文档链接、版本、secret 和个人路径扫描。没有 Server、凭据、浏览器媒体、TURN、Redis 集群或外部运营商时，必须报告未完成，不得声称端到端通过。

`common/livekitx` 本地集成测试以 `LIVEKITX_INTEGRATION=1` 显式开启（默认跳过，不影响普通单测），连接 `http://127.0.0.1:7880`、`devkey`/`secret`；覆盖房间生命周期、SDK 原生入会、原生回调（入会/聊天双路径/RPC 往返/断开原因）、`SendData` 富媒体投递。Egress/Ingress/SIP/Agent 与 Webhook 服务端推送不做真实端到端断言，只能标注环境前置条件。
