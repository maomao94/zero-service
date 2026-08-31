# 全面复核 LiveKit 学习指导并升级到最新版本

## Goal

全面检查 LiveKit 学习与对接指导及其后续开发规范，基于检查时可获得的最新稳定 LiveKit Server、Go Server SDK、Protocol 版本和官方文档，修正所有可验证的错误，使后续会议服务开发可以将文档作为可靠依据。

## Background

- 目标文档为 `docs/livekit-integration-guide.md`，关联规范为 `.trellis/spec/backend/livekit-guidelines.md`，导航和本任务审计资料也必须保持一致。
- 上一轮文档记录的版本是 LiveKit Server `1.13.6`、Go Server SDK `2.18.2`，但当前本地源码工作树显示 Server `1.13.6-12-g46001402`、SDK `v2.18.1-28-ga41bea9`，必须重新确认版本来源和可复现方式。
- 检查范围覆盖全文，不采用抽样；包括文字陈述、版本、模块路径、Go API/Protobuf 字段和枚举、Token grant、Webhook、管理 API、实时参与者、Egress、Ingress、SIP、Agent、自托管配置、架构边界、示例代码、验证命令和源码链接。

## Requirements

1. 先确认检查时最新稳定版本的定义和证据来源。对 LiveKit Server、`github.com/livekit/server-sdk-go/v2`、`github.com/livekit/protocol`、Go 版本要求分别记录版本、来源、查询日期和本地验证状态；不能把本地未提交源码或 pseudo-version 无依据地称为最新稳定版本。
2. 逐段逐项对照本地 LiveKit Server、Go Server SDK、Protocol 源码及官方最新文档，建立审计记录；每个发现的问题必须修正文档或明确标记为版本条件、部署前置条件或未验证项。
3. 所有 Go 示例中的 import、构造函数、方法名、参数类型、字段名、oneof、枚举、grant 和返回值必须与选定版本实际编译接口一致；不能保留已知错误示例或过时 API。
4. 对 Server SDK、Protocol、客户端 SDK、Go 实时参与者、Webhook、Egress、Ingress、SIP、Agent worker/Cloud Agents 的职责和授权边界分别核验，避免将管理 API 可调用误写成对应运行服务已部署。
5. 核验自托管部署、网络、TLS/WSS、WebRTC、Redis、TURN 及独立服务要求，并明确 Cloud 专属能力不能推导为自托管能力。
6. 核验 zero-service 集成规范中的状态所有权、密钥管理、最小权限、Webhook 幂等/乱序/丢弃处理、错误传播和客户端安全约束，确保规范只承诺已实现或已验证的接口。
7. 文档应明确版本升级后的复核步骤和验证限制；若无法完成真实端到端测试，必须准确记录缺失的服务、凭据或系统依赖，不得声称全部通过。

## Acceptance Criteria

- [ ] 指南、backend spec、docs 导航及任务审计资料中的版本和链接口径一致，并能追溯到官方发布页、模块元数据或源码 commit。
- [ ] 指南全文完成逐项审计，审计报告列出检查范围、证据来源、发现项、修复项和未能验证的环境限制；不存在 `TBD`、`TODO`、旧 API 名称、无依据的最新版本表述或已知错误示例。
- [ ] 选定版本下所有可执行 Go 示例通过独立临时 module 的 `go mod tidy`、`go test ./...` 或等价编译检查；无法执行的示例有明确原因和替代静态核验依据。
- [ ] 运行文档链接、路径、代码字段、版本和敏感信息扫描，以及 `git diff --check`；扫描结果无个人绝对路径、真实密钥和陈旧 API。
- [ ] 对完整 SDK 测试、LiveKit Server 端到端测试和本机系统依赖的通过/未通过状态作出事实准确的最终报告。
- [ ] 任务文档包含最终 `prd.md`、`design.md`、`implement.md`，并在完成后归档任务和记录会话。

## Out Of Scope

- 不实现 `common/livekitx`、`app/meeting` 或新的会议 gRPC API。
- 不修改 LiveKit Server 或 `server-sdk-go` 外部源码；只使用它们作为核验依据。
- 不将无法由当前环境验证的 Cloud、SIP 运营商、TURN、浏览器媒体和多节点故障切换行为伪装成已通过的端到端结果。

## Open Questions

- None. “最新”按复核执行当日可核验的官方最新稳定发布版本定义；若官方没有稳定发布或本地源码与稳定发布不一致，文档必须分别记录稳定版本和本地验证 commit，不得混写。
