# 开发 `common/livekitx` 快速开箱包

## Goal

为 zero-service 的业务服务提供一个基于 `github.com/livekit/server-sdk-go/v2` 的统一 LiveKit 能力包。业务方能够通过稳定、可发现、带中文文档的 API 完成管理请求、实时参与者连接、Token 签发、Webhook 接收和事件 Hook 注册，而不需要重复理解 LiveKit SDK 的认证、客户端装配、事件路由和资源关闭细节。

## Background and Confirmed Facts

- 当前仓库没有 `common/livekitx` 实现，也没有在根 `go.mod` 声明 LiveKit Go SDK。
- 稳定基线为 LiveKit Server `v1.13.6`、`github.com/livekit/server-sdk-go/v2 v2.18.1`；Protocol 版本必须从锁定 SDK 的 `go.mod` 得出。
- 当前本地已运行 `livekit-server --dev`，服务端地址为 `http://127.0.0.1:7880`，开发凭据为 `devkey` / `secret`，仅限本地测试使用。
- 稳定 SDK 的统一管理入口 `LiveKitAPI` 提供 Room、Egress、Ingress、SIP、AgentDispatch、Connector 子客户端；实时连接、Token、Webhook 和媒体能力使用独立 SDK 包或入口。
- `common` 包规范要求公共包只提供机制，业务状态和业务授权策略留在调用服务；client 应复用、资源应可关闭、外部调用应传递 context 并保留底层错误语义。

## Requirements

### R1. 统一配置与生命周期

- 提供可校验的配置和 function options，支持服务 URL、API key/secret、HTTP transport/client 和日志/观测注入；请求超时由业务传入的 context 控制。
- 构造并复用统一的管理 API client；不得在每次请求中重新创建 client。
- 明确零值、配置冲突、凭据缺失和关闭语义；长期资源提供幂等 `Close`。

### R2. 核心管理 API 分发

- 对稳定 SDK 暴露的管理 API 提供统一可发现的 client 访问能力，优先覆盖快速会议业务需要的 Room/Participant、Egress、Ingress、SIP 和 AgentDispatch。
- Connector、AgentSimulation 和 Cloud Agents 不属于本次 MVP；未来可以基于相同认证/HTTP 基础设施增加独立扩展包。
- 不复制或重新定义 LiveKit Protocol 请求/响应类型；调用方可以使用锁定 SDK 的原生类型，减少版本漂移和字段丢失。
- 每个请求支持调用方 context，保留 SDK 原始错误和 Twirp code/message，不把所有失败压扁为单一错误。

### R3. Token 与实时参与者能力

- 提供 Video/SIP 等授权 Token 的便捷构造入口，强制或引导最小权限和指定房间/身份。
- 提供实时参与者连接、发布/订阅、数据、RPC 和媒体相关 SDK 能力的统一接入；业务策略不得固化在公共包中。
- “聊天”在本包中提供可靠的 Data/RPC 发送和接收机制，并通过聊天消息 Hook 把事件交给业务服务；消息持久化、历史查询、未读数、敏感词和业务会话归属由业务 handler 负责。
- 清楚区分管理 API 凭据、参与者 Token、Webhook signing key、SIP 凭据和 worker/agent 凭据。

### R4. Hook 与事件分发

- 提供业务服务注册 handler 的机制，覆盖 Webhook 事件、实时 Room/Participant/Track/Connection/Data/RPC 回调，以及 SDK 可注册的其他公开回调能力。
- Hook API 优先服务于会议状态同步、参与者变更、轨道状态和聊天消息分发。业务服务可以在 handler 中落库、审核、更新会议状态或触发业务事件；公共包不直接绑定业务数据库、业务模型或业务 gRPC。
- handler 注册、取消、并发执行顺序、panic/error 传播、context 取消、关闭期间行为必须有明确契约。
- Webhook 保留原始 body 和 `Authorization` 进行签名校验；事件 ID 去重、重试、迟到、乱序、未知事件和对账责任由公共机制支持、业务服务决定持久化策略。

### R5. 测试与文档

- 提供完整单元测试，覆盖配置、认证、请求路由、错误、超时/取消、Hook 注册与分发、并发、关闭、重复 Webhook 和边界输入。
- 使用本地 `livekit-server --dev` 提供可重复的集成测试，覆盖真实 Room/Participant、Token、Webhook 和可由开发 Server 支持的管理 API；需要外部 Egress/Ingress/SIP/Agent/Redis/媒体基础设施的能力必须标注环境前置条件，不得伪造通过。
- 在 `common/livekitx/README.md` 提供完整功能目录、初始化示例、API 使用方式、Hook 注册示例、错误/生命周期约定、测试启动命令和能力边界，方便后续业务服务接入。

## Out of Scope

- 不实现 `app/meeting` 或任何具体业务会议单据、用户角色、数据库模型和业务 gRPC/HTTP 契约。
- 不复制 LiveKit Protocol 生成代码，不维护与 SDK 平行的业务 DTO 映射层。
- 不纳入 `pkg/cloudagents`；本项目当前使用自托管 LiveKit Server 和独立 Agent worker，不负责 LiveKit Cloud Agent 的源代码上传、构建、部署、发布或日志管理。
- 不在本任务内部署或改造生产 LiveKit、Egress、Ingress、SIP、Agent、Redis、TURN、对象存储或运营商环境。
- 不承诺 Webhook Exactly Once，不把 Webhook 作为业务唯一状态源。

## Acceptance Criteria

- [ ] `common/livekitx` 包完成构造、配置校验、统一 client 复用和幂等关闭，并通过单元测试。
- [ ] 稳定 SDK `v2.18.1` 中与快速视频会议直接相关的管理 client、Token、实时连接、Webhook、数据/RPC 和媒体相关入口均有可发现的封装或明确的原生转发入口；Connector、AgentSimulation、Cloud Agents 等非 MVP 能力有明确排除清单和理由。
- [ ] 业务服务可以注册并注销 Webhook 与实时事件 handler；分发行为、并发安全、错误和 panic 语义有测试覆盖。
- [ ] 本地 `livekit-server --dev` 集成测试可在明确的环境变量/启动条件下运行，并验证真实服务端交互；不可用的外部能力被单独跳过并说明原因。
- [ ] `common/livekitx/README.md` 覆盖全部已提供功能、参数、示例、生命周期、错误、Hook 和测试说明，示例与代码可编译。
- [ ] 目标包测试、必要的 `go test -race`、全仓 `go test ./...` 和 `git diff --check` 通过，且不引入真实凭据。
