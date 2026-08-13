# 认证上下文与凭证传播加固

## Goal

分阶段加固认证身份 context、JWT/MCP claim 和 Authorization 跨服务传播，在不隐式破坏现有调用链的前提下，明确进程内身份所有权、外部输入规范和凭证信任边界。

本父任务只负责路线图、子任务顺序和跨子任务验收，不直接作为代码实施目标。

## Requirements

- 按以下顺序执行四个可独立验收和归档的子任务：
  1. `audit-authorization-propagation`：只读审计凭证传播链、信任边界、重复 Authorization 和泄漏面。
  2. `typed-auth-context-keys`：分离进程内 typed key 与 JWT/gRPC/MCP wire key。
  3. `normalize-auth-claims`：定义并实施身份 claim 类型与错误规则。
  4. `enforce-authorization-policy`：根据审计结论实施凭证传播模式和冲突规则。
- 每个子任务在开发前必须提交完整 PRD、设计、实施计划和验证范围，并等待用户明确回复“开始开发”或等价确认。
- 每个子任务开发和质量检查完成后必须暂停，向用户报告 diff、行为变化、兼容性、测试和残余风险；用户确认后才能提交、归档并规划下一子任务。
- 后续子任务不得因前一子任务获批而自动获得开发许可。
- `b64:` gRPC metadata 编码升级不纳入本父任务；保持现有 wire contract，另行按实际风险决定是否立项。
- 不在审计子任务中修改应用代码、配置或运行策略。

## Acceptance Criteria

- [ ] 四个子任务均分别通过开发前和开发后人工确认门禁。
- [ ] Authorization 的服务信任边界、允许传播目标和凭证类型有明确可执行策略。
- [ ] 进程内身份 context 不再由公开 string key 作为主要存储机制，wire key 保持兼容。
- [ ] JWT/MCP 身份 claim 的支持类型、非法值和错误行为有明确测试。
- [ ] gRPC/MCP 原始 Authorization 仅按批准策略传播，重复或冲突值有确定行为。
- [ ] HTTP、Socket.IO、gRPC、MCP 和关键业务身份读取链路具有回归测试与回滚方案。
- [ ] `b64:` 编码、现有 wire key 和字段顺序除非另立任务，不因本父任务发生变化。

## Subtasks

- `08-14-audit-authorization-propagation`
- `08-14-typed-auth-context-keys`
- `08-14-normalize-auth-claims`
- `08-14-enforce-authorization-policy`
