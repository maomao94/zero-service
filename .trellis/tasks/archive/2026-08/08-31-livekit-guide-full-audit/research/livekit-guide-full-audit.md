# LiveKit 指南全面审计记录

审计日期：2026-08-31。范围：`docs/livekit-integration-guide.md` 全文、`.trellis/spec/backend/livekit-guidelines.md`、文档导航、Go Server SDK/Protocol API、Server 配置与部署边界。未采用抽样。

## 版本证据

| 项目 | 稳定基线 | 本地证据 |
| --- | --- | --- |
| Server | `v1.13.6`，GitHub release/tag | 工作树 HEAD `460014023fe5df504bdfff51cbb47ddbfdd1e6fd`，`v1.13.6-12-g46001402` |
| Go SDK | `v2.18.1`，GitHub release/tag 与 Go module | 工作树 HEAD `a41bea984454ccc958c238bde111022917806aaa`，`v2.18.1-28-ga41bea9` |
| Protocol | 稳定 SDK tag `go.mod` 为 `v1.49.0` | 本地开发工作树解析为 `v1.50.5-0.20260829123501-a469dd43727b`；未混称为稳定 tag |
| Go | 稳定 SDK tag `go 1.26` | 本地开发工作树 `go 1.26.3`；未推导 zero-service 全项目工具链要求 |

## 发现与修复矩阵

| 范围 | 发现 | 修复 |
| --- | --- | --- |
| 版本 | 原指南写入了错误的 SDK 稳定版本 | 统一为稳定 `v2.18.1`，另列本地开发 commit |
| API 入口 | 混用 SDK 入口与 Protocol `New*JSONClient` | 以 `LiveKitAPI` 为首选；Protocol 直用要求锁版本、读生成代码并编译 |
| SIP | 旧示例字段/oneof 未证明 | 删除具体未核验字段，改为按选定 Protocol 核验的边界说明 |
| Agent | 将 worker、dispatch、Cloud Agents 权限混为一谈 | 分列 worker、房间 dispatch、Cloud 管理授权和部署依赖 |
| Webhook | 固定事件全集和可靠性语义 | 仅承诺验签、原始 body、去重、乱序/丢失处理，未知事件安全忽略；不把 Agent job 事件写成稳定 Webhook 契约 |
| 配置 | 旧 YAML 固定字段、默认值和端口无逐项证据 | 删除模板，引用当前 Server 配置样例并说明网络前置条件 |
| zero-service | 状态所有权、secret、错误和生命周期约束不完整 | backend spec 增加明确契约 |
| 示例 | 多个片段不可独立编译或缺少中文解释 | 保留完整功能覆盖，明确片段上下文，所有功能说明和 Go 步骤补中文注释 |
| 覆盖范围 | 重写时过度删减了房间管理、录制、输入、SIP、Agent、媒体和排障内容 | 恢复为单一不重复章节，并逐项补回源码支持的功能入口 |
| 文档结构 | 恢复功能时出现重复章节 | 合并重复的 Token、实时参与者、Webhook、Twirp 和功能边界说明 |

## 源码核验

已读取 SDK 的 `livekitapi.go`、`roomclient.go`、`egressclient.go`、`ingressclient.go`、`sipclient.go`、`agent_dispatch_client.go`、`room.go`、`localparticipant.go`、`auth.go`、`errors.go` 和稳定 tag/本地工作树 `go.mod`；确认 `NewLiveKitAPI`、`ConnectToRoomWithToken`、管理子客户端和 `ServerError` 的实际形状。已读取 Protocol module cache 中房间、SIP、Agent dispatch 生成类型；稳定依赖与本地开发解析值分开记录。

## 验证限制

本次任务完成文档全文静态审计、版本/tag/commit 证据整理和源码 API 核对。临时 module 使用稳定 SDK `v2.18.1` 及其稳定 Protocol 依赖 `v1.49.0`，执行了 `go mod tidy` 和 `go test ./...`，并额外覆盖房间、参与者、Egress、Ingress、SIP、Agent Dispatch、Token、实时参与者、错误和 Webhook 类型入口，结果通过且无测试文件。由于没有在本任务中启动完整 LiveKit Server、Egress/Ingress/SIP/Agent、浏览器媒体链路、TURN、Redis 集群和真实凭据，未宣称这些场景的端到端通过。
