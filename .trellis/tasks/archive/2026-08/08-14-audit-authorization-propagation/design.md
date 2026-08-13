# Authorization 传播审计设计

## 边界

本子任务是只读安全审计。它拥有传播矩阵、信任边界、风险分类、后续策略输入和迁移建议，不拥有任何生产代码、配置或运行策略变更。

审计覆盖 HTTP、Socket.IO、JWT、`common/authctx`、gRPC `common/grpcx`、MCP `common/mcpx`、下游身份消费者、日志/trace/error/raw `_meta` 和可发现的持久化路径。

## 分类模型

每条 transport boundary 记录：

```text
source -> verification -> context write -> transport -> receiver -> consumer
```

并标注：凭证类型、是否重验、身份是否影响数据隔离、当前信任假设、泄漏面、owner、推荐目标模式和证据路径。

目标模式使用以下固定词汇：

- `user-token`：原始用户凭证，仅允许明确批准的用户委托链路，接收端必须具备用户 token 验证/授权职责。
- `claims-only`：不传播原始 token，只传播规范化身份；依赖受认证的服务边界。
- `service-token`：使用独立服务凭证认证 transport，用户身份若需要则单独携带。
- `none`：既不传播原始 token，也不传播用户身份。
- `unresolved`：缺少业务 owner 或部署信任信息，不能在代码证据上推断。

用户已批准默认拒绝基线：除显式委托 allowlist 外，不将链路分类为 `user-token`。

## 当前关键数据流

- AI gateway 的受保护 HTTP 路由验证 JWT 后，将 raw Authorization 和映射 claims 写入 context；gRPC client interceptor 会通用转发。
- Socket.IO 握手验证 token 后，每个事件 context 保留 raw token，并经 StreamEvent gRPC 转发；当前下游直接用途仅发现完整 token 日志。
- MCP HTTP transport 使用 service token 认证连接，但每次 tool call 又会把调用方 raw Authorization 放入 `_meta`；server wrapper 可恢复后继续经 gRPC 转发。
- gRPC server interceptor 不验证 token 或 reconcile `x-user-*` 与 Authorization，默认信任调用方 metadata。

## 风险与策略输入

- 已确认三处完整 token 日志：StreamEvent、MCP echo、MCP auth extra map。
- gRPC 重复值取首项，空首项会抑制后续值；Authorization 与身份 metadata 不做一致性校验。
- MCP raw `_meta` 在请求 context 生命周期内原样保留，当前未发现业务读取或持久化消费者。
- `user-id` 已用于 AI session/knowledge 数据隔离，因此身份来源和冲突处理是安全属性。
- 仓库无法证明生产 mTLS、service identity、proxy header 规则和外部 MCP server 行为，这些保持为 deployment owner 输入。

## 交付物

- 传播矩阵和身份 claims 矩阵。
- gRPC duplicate/conflict 契约审计。
- MCP `_meta` 与 token 泄漏审计。
- 按默认拒绝基线分类的候选策略表，未获业务 owner 证明的链路标为 unresolved，不擅自加入 allowlist。
- receiver-first 迁移顺序、无 token 内容的观测字段、回滚策略。
- 后续三个子任务的输入与禁止混合项。

## 兼容与回滚

本审计不改变运行行为，因此无代码回滚。审计建议的后续实施必须遵循 receiver-first：先观测和兼容，再按小范围 sender cohort 切换；保留 legacy 开关直到批准的观测窗口结束。`b64:`、wire key 和字段顺序保持不变。
