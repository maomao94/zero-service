# 实施 Authorization 传播策略（观测与日志脱敏先行）

## Goal

基于已确认的 Authorization 审计结论，在**保留现有 raw token 传播**的前提下，建立接收端认证 metadata 的 report-only 观测（重复/冲突信号），并修复三处完整 token 日志泄漏，为后续策略实施提供数据基础。

## Requirements

- 本任务仅在前三个子任务经用户确认后规划和实施。
- 开发前必须由用户确认每条服务链路的传播模式、默认策略、灰度与回滚方案。
- 接收端兼容能力先于发送端策略切换，不得一次性切断现有认证链路。
- 重复 Authorization、空值/非空值并存和身份 metadata 冲突必须有明确拒绝或归一规则。
- 开发完成后暂停，由用户确认端到端结果后才提交和归档。
- `b64:` wire 协议升级仍不在本任务范围。

## Key Decision（用户已确认 2026-08-14）

- **传播策略**：raw token 按需传播（gRPC 侧可能需要用 token 获取用户信息或透传 HTTP/feign），**本任务不停止任何发送端的 raw token 传播**。
- **任务范围（修订）**：仅做 **日志脱敏 L1-L3**——StreamEvent 完整 token Info 日志、MCP echo Debug token 日志、MCP auth Debug extra map 日志。
- 接收端 report-only 观测为**过度设计，已移除**；不做观测。
- 发送端传播、MCP `_meta` 内容、重复/冲突处理均不改，留待后续策略任务。

## Acceptance Criteria

- [ ] 三处 token 日志泄漏（L1/L2/L3）已脱敏，不记录完整 token 值。
- [ ] 现有传播行为、wire key、`b64:`、claim 转换、MCP `_meta` 内容全不变。
- [ ] `auth_test.go` Extra 契约（含 raw token）保持通过。
- [ ] 用户确认开发结果后才提交并完成父任务验收。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- 规划中：`research/` 7 份证据文件已就绪（senders/receivers 清单、重复冲突 surface、日志修复点、配置灰度、MCP meta policy、index）。
- 开发前等待用户确认本 PRD / design / implement 后方可 `task.py start`。
