# LiveKit 中文学习与对接指南执行计划

## 1. 审计事实

- [x] 核对 `server-sdk-go` 版本、Go 版本、顶层客户端、认证和 failover 行为。
- [x] 核对 Room/Participant、Webhook、Egress、Ingress、SIP、Agent 的真实方法和权限。
- [x] 核对 LiveKit Server 自托管端口、Redis、TURN/TLS 及独立组件要求。
- [x] 记录当前文档中的错误、过度承诺、版本漂移和缺失主题。

## 2. 重构学习指南

- [x] 增加系统地图、术语、读者路径和先修知识。
- [x] 提供最小本地实验和故障定位步骤。
- [x] 分开说明管理 API 与实时参与者 SDK。
- [x] 完善 Webhook、Egress、Ingress、SIP、Agent 和生产部署章节。
- [x] 增加 zero-service 后续对接边界、学习检查表和源码索引。

## 3. 收敛工程 Spec

- [x] 删除未实现 helper 的固定签名和不可靠默认值。
- [x] 固化依赖方向、认证、状态所有权、幂等、错误、超时和部署前置条件。
- [x] 将教程细节链接到 `docs/livekit-integration-guide.md`，避免重复。

## 4. 验证

- [x] 使用本地 SDK 对核心示例执行 `go test` 或 `go build`。
- [x] 扫描占位标记、死链接式本地引用、术语和版本不一致。
- [x] 运行 `git diff --check` 并审阅最终 diff。
- [x] 使用 Trellis quality check 复核 spec 与任务验收条件。

## 回滚点

- 只修改当前两份 LiveKit 文档、backend spec 索引和任务规划工件。
- 若大纲重构导致信息丢失，保留原指南中的有效 API 示例并迁移到专题附录，不删除未经核对的独有信息。
