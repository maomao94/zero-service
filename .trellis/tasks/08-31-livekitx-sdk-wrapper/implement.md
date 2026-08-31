# `common/livekitx` 实施计划

## 阶段 1：依赖与骨架

- [ ] 锁定 `github.com/livekit/server-sdk-go/v2 v2.18.1`，执行 `go mod tidy` 并确认 Protocol 依赖。
- [ ] 创建 `common/livekitx` 包、文档注释、Config/Option、Client 和幂等 Close。
- [ ] 接入共享 HTTP client、go-zero httpc.Service 适配和凭据脱敏；请求超时完全由调用方 context 控制。
- [ ] 先写配置和生命周期测试，再实现错误路径和资源所有权。

## 阶段 2：管理 API 与 Token

- [ ] 暴露 Room/Participant、Egress、Ingress、SIP、AgentDispatch 的统一访问入口，直接使用 SDK 原生请求/响应类型。
- [ ] 编写管理 API mock 测试，覆盖认证、context 取消、Twirp 错误和请求边界。
- [ ] 实现 Join Token 与 SIP Token 便捷构造，覆盖最小权限、身份/房间校验和有效期。
- [ ] 添加 Token 单元测试，禁止测试输出真实 secret/token。

## 阶段 3：Hook、实时连接、Data/RPC

- [ ] 设计并实现实例级 typed Hook dispatcher、Subscription 注销和关闭语义。
- [ ] 先添加 handler 顺序、并发注册/注销、错误聚合、panic recovery、context 取消测试。
- [ ] 桥接实时 Room、Participant、Track、Connection、Data 和 RPC 回调。
- [ ] 实现聊天消息标准事件与可靠/不可靠 Data 发送入口；README 明确业务 handler 的持久化责任。
- [ ] 添加真实 SDK 实时连接测试所需的最小参与者 fixture，并确保连接关闭不泄漏 goroutine。

## 阶段 4：Webhook 与本地集成

- [ ] 实现原始 body/Authorization 的 Webhook 校验和事件分发适配器。
- [ ] 测试签名失败不调用 handler、重复 event ID 可交给业务去重、未知事件安全处理。
- [ ] 添加 `livekit-server --dev` 集成测试文件和环境变量开关；默认不因本地服务未启动导致普通单元测试失败。
- [ ] 覆盖房间生命周期、Token 入会、实时 Data/RPC 和可用 Webhook 事件。

## 阶段 5：文档与质量门禁

- [ ] 编写完整 `common/livekitx/README.md`：初始化、房间、参与者、Token、实时连接、聊天、RPC、Webhook、录制/输入/SIP、Hook、错误、关闭、测试和能力边界。
- [ ] 为 README 每个 Go 示例提供可识别的 import/变量上下文，示例代码通过编译检查。
- [ ] 运行 `gofmt`、`go vet ./common/livekitx/...`、`go test ./common/livekitx/...`、`go test -race ./common/livekitx/...` 和 `go test ./...`。
- [ ] 运行文档/secret/旧 API 扫描、`git diff --check`，完成 `trellis-check` 质量复核。

## 风险与回滚点

- SDK 公开 API 与本地 checkout 可能继续变化：所有实现以 go.mod 锁定版本编译，升级单独提交。
- 本地 dev server 不包含 Egress/Ingress/SIP/Agent 外部服务：这些只做 mock/请求层测试，集成报告必须单独标注跳过原因。
- 实时媒体测试可能受本机 native codec 依赖影响：将核心 Hook/Data/RPC 测试与媒体编解码测试分离。
- Hook API 一旦被业务服务使用即形成公共契约；先完成测试和 README，再扩展事件类型，不在首版暴露内部 mutex/map。
