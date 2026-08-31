# 审阅并完善 LiveKit 中文学习与对接指南

## Goal

基于本地 `livekit` 与 `server-sdk-go` 源码，形成一套可信、可追溯、可用于后续 zero-service 开发的中文 LiveKit 学习与对接资料。读者应能先理解 LiveKit 的系统边界，再完成最小实验，并能据此设计后续 gRPC 业务服务，而不依赖猜测英文 API 的语义。

## Background

- 本地源码位于 `livekit` 与 `server-sdk-go`。
- 当前已有 `docs/livekit-integration-guide.md` 与 `.trellis/spec/backend/livekit-guidelines.md`，但尚未系统核对示例可编译性、版本事实、部署依赖、学习顺序和工程契约边界。
- 当前阶段只学习、调研和完善文档，不开发 `common/livekitx` 或 `app/meeting`。

## Requirements

- 以本地源码、测试和官方文档为证据，核对 Server SDK、Protocol、LiveKit Server 的职责和真实 API。
- 明确三条不同接入路径：服务端管理 API、前端/客户端加入房间、Go 后端作为实时参与者加入房间。
- 中文学习指南按概念、部署、最小实验、Server SDK、实时参与者、Webhook、Egress、Ingress、SIP、Agent、生产注意事项和 zero-service 映射组织。
- 示例必须注明必要 import、前置服务和权限；核心示例应能通过独立编译检查或明确标注为片段。
- 记录自托管与 LiveKit Cloud 的差异，不能把 Cloud 专属容错或服务能力描述为 OSS 默认行为。
- 说明 Webhook、管理 API 和本地业务数据库之间的状态所有权、重复事件、顺序、幂等和最终一致性边界。
- `.trellis/spec/backend/livekit-guidelines.md` 只保留后续开发必须遵守的契约，不提前承诺尚未实现的 `common/livekitx` 公共 API。
- 所有版本号都标注来源或改为“以当前 `go.mod`/选定版本为准”，避免复制伪版本。

## Acceptance Criteria

- [ ] 指南提供从零开始的建议学习顺序和至少一个可执行的最小验证闭环。
- [ ] 指南清楚区分 LiveKit Server、Server SDK、Protocol、客户端 SDK、Egress、Ingress、SIP 与 Agents。
- [ ] Room/Participant、Token、Webhook、Egress、SIP、Agent 的关键用法均有源码或官方链接依据。
- [ ] 示例中的关键类型、字段和方法与本地 `server-sdk-go`/`protocol` 版本一致；抽检代码通过编译验证。
- [ ] 文档明确 API secret 只存在于服务端，并给出最小权限 token 原则。
- [ ] 文档明确自托管生产部署所需端口、Redis/TURN/TLS 和独立服务依赖，不将单节点示例当作生产方案。
- [ ] Spec 不包含未经实现验证的公共 helper 签名，后续 `app/meeting` 设计可直接引用其边界和检查清单。
- [ ] 无 `TBD`/`TODO`，内部术语、链接和交叉引用一致，`git diff --check` 通过。

## Out of Scope

- 不创建或修改 `common/livekitx`、`app/meeting`、protobuf、数据库模型或部署环境。
- 不实现 Web、移动端或桌面端会议 UI。
- 不部署 LiveKit、Egress、Ingress、SIP 或 Agent worker。
- 不承诺业务会议模型、用户权限模型或 API 契约；这些在后续 gRPC 服务任务中设计。
