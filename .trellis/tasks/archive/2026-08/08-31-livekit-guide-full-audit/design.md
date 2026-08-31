# LiveKit 指南全面复核设计

## 目标

将 LiveKit 学习指南从一次性源码审阅稿提升为可持续核验的开发依据。文档以官方最新稳定发布为版本基线，以本地源码 commit 作为编译证据，并对正文中的每一个 API、字段、配置和行为陈述建立可追溯证据。

## 版本策略

- 稳定版本以官方 GitHub release/tag 和 Go module tag 为准。
- 本次初始证据：LiveKit Server 最新稳定 tag 为 `v1.13.6`；`server-sdk-go` 最新稳定 tag 为 `v2.18.1`；两者当前本地工作树分别是 `v1.13.6-12-g46001402` 和 `v2.18.1-28-ga41bea9`。
- Protocol 是 SDK 和 Server 的依赖，不单独猜测或截断 pseudo-version；记录选定 SDK 的 `go.mod` 解析结果。
- 每次版本升级都要重新运行 API 示例编译、配置字段检查、官方链接检查和全文扫描。

## 核验层次

1. 版本与来源：release、tag、module metadata、源码 commit、官方文档更新时间。
2. API 契约：Server SDK 导出 API、Protocol 生成类型、oneof、枚举、返回值和错误类型。
3. 行为契约：授权、房间状态、Webhook 发送/重试、Egress/Ingress/SIP/Agent 服务依赖和 Cloud/自托管差异。
4. 配置契约：Server `config-sample.yaml` 与当前官方部署文档中的字段、默认值、端口和证书要求。
5. 工程契约：zero-service 的状态所有权、secret、context、超时、幂等、错误传播、资源生命周期和文档同步。
6. 可执行性：提取全部 Go fenced code blocks，逐个分类；核心示例在独立 module 编译，概念性片段标明缺失上下文或补齐为可运行示例。

## 产物边界

- 修改 `docs/livekit-integration-guide.md`、`.trellis/spec/backend/livekit-guidelines.md`、相关导航和本任务审计报告。
- 不修改 LiveKit 外部源码，不实现 zero-service 业务服务。
- 不能用“已检查”替代证据；每项无法运行的验证必须写出具体环境限制。

## 风险与处置

- 最新 tag 可能落后于开发分支：稳定发布和本地开发 commit 分栏记录，避免链接指向不存在的 tag。
- Protocol 生成字段可能随 pseudo-version 变化：示例以选定 SDK 的实际依赖和本地源码编译为准。
- 官方文档 URL 可能重定向或迁移：正文链接统一使用当前 canonical URL，并保留主题名称。
- 外部服务和原生媒体库无法在本机完整启动：完成静态/API/编译验证，明确不宣称端到端通过。
