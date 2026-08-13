# 迁移认证 Typed Context Key

## Goal

将进程内认证身份存储迁移为包私有 typed context key 和统一 setter/getter，同时保持 JWT、gRPC metadata、MCP `_meta` 等 wire key 与传播行为兼容。

## Requirements

- 本任务仅在 Authorization 审计结果经用户确认后规划和实施。
- 开发前与开发后均需单独人工确认，确认前不得自动进入下一子任务。
- typed key 与 wire key 必须分离；不得仅把公开 string constant 改成自定义 string 类型。
- 迁移期兼容策略、旧 string key fallback 的移除条件和全仓调用清单必须在设计阶段明确。
- 不在本任务收缩 Authorization 传播或改变 claim 类型规则。

## Acceptance Criteria

- [ ] 用户确认最终设计后才进入开发。
- [ ] HTTP、Socket.IO、JWT、gRPC、MCP 与业务调用方使用统一 authctx setter/getter。
- [ ] wire key、metadata key、字段顺序和请求行为保持不变。
- [ ] 开发和检查完成后暂停，用户确认后才提交、归档和进入下一任务。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- 规划已完成：`design.md`（阶段 0-3 迁移、双读回退、移除条件、风险表）、`implement.md`（分阶段执行清单与验证命令）、`research/`（6 份证据清单）均已就绪。
- 开发前等待用户确认本 PRD / design / implement 后方可 `task.py start`。
