# 规范化认证 Claim 值

## Goal

为 JWT 和 MCP 身份 claim 定义类型白名单、精确字符串转换及非法输入行为，避免任意对象、数组、nil 或不精确数字进入身份上下文。

## Requirements

- 本任务仅在前置审计与 typed context key 子任务经用户确认后规划和实施。
- 开发前必须由用户确认支持类型矩阵，以及非法必需/可选 claim 是拒绝还是忽略。
- 开发完成后必须暂停并提交兼容性与测试结果，用户确认后才归档。
- 不在本任务改变 Authorization 跨服务传播策略。

## Key Decision（用户已确认 2026-08-14）

- **类型白名单**：保持宽松兼容——`user-id` 接受 string、int/int64/uint/uint64、整数值 float64（≤2^53 精确）、`json.Number`（精度无损）。现有 zerorpc(int64)/socketpush(string) 签发链路继续可用。
- **非法值语义**：在无错误通道的边界（网关 `BridgeJWTClaims`、MCP `_meta` 恢复、claims 映射）对非法类型（bool/数组/对象/分数 float/超大 float）**忽略跳过**，视同缺失，不写入 typed key；不新增网关 401 通道。
- **数字精度**：float64 路径（MCP/socket `tool.ParseToken`）整数值 >2^53 已在解析时舍入，规范化时**拒绝**（无法恢复精确值）；go-zero 网关路径用 `json.Number` 天然精确，不做额外处理。

## Acceptance Criteria

- [ ] JWT/MCP claim 类型矩阵和错误语义经用户确认。
- [ ] 数字精度、nil、bool、数组、对象和缺失值均有契约测试。
- [ ] 现有合法身份 token 有明确兼容证据或迁移模式。
- [ ] 用户确认开发结果后才进入 Authorization 策略实施。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- 规划中：`research/` 7 份证据文件已就绪（转换矩阵、值类型清单、消费者影响、精度分析、白名单选项、边界约束、未知项）。
- 开发前等待用户确认本 PRD / design / implement 后方可 `task.py start`。
