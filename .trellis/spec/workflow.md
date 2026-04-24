# Trellis 工作流规范

> Trellis 工具的命令、自动触发、会话记忆规范。Sprint 敏捷流程详见 `agile-dev-manager` 技能。

## 编码三段式

1. **编码前**：读 `.trellis/spec/` 规范 → 检索项目代码 → 确认技术栈
2. **编码中**：遵循规范 → `zero-skills` 获取框架知识 → 复用现有组件
3. **编码后**：`go build` → 工具类单测 → proto/api 注释检查

## Trellis 命令

| 命令 | 用途 | 使用时机 | 对应 Sprint 阶段 |
| --- | --- | --- | --- |
| `/start` | 加载项目上下文（spec + workspace + current-task） | 会话开始 | Phase 0 |
| `/brainstorm` | 探索需求、需求澄清 | Planning | Phase 1 |
| `/before-dev` | 编码前上下文加载、规范注入 | Planning → Execute | Phase 1→2 |
| `/check` | 代码规范检查 + 自修复循环 | Execute / Review | Phase 2/3 |
| `/finish-work` | 提交前完成检查 | Retro | Phase 4 |
| `/record-session` | 记录会话到 workspace journal | 会话结束 | Phase 4 |
| `/update-spec` | 规范沉淀到 `.trellis/spec/` | 发现新模式时 | Spec 模式 |
| `/continue` | 推进下一步工作流，防止跳步骤 | 卡住时 | 任意 |

> `/continue` 读取 `.current-task` 和任务 status，对照 workflow.md 判断当前阶段，决定下一步该做什么。

## Trellis Skill 自动触发

| Skill | 自动触发条件 | 行为 |
| --- | --- | --- |
| trellis-brainstorm | 用户描述需求/bug/模糊诉求 | 转成 task + prd.md |
| trellis-before-dev | 编码前 | 先读 spec 再写代码 |
| trellis-check | 实现完成后 / sub-agent 调用 | 验证 + 自修复循环 |
| trellis-update-spec | 有值得沉淀的知识 | 固化进 `.trellis/spec/` |
| trellis-break-loop | 修完棘手 bug 后 | 根因分析 + 预防机制 |

## Trellis Sub-agent

| Sub-agent | 触发条件 | 行为 |
| --- | --- | --- |
| trellis-research | 主会话需要调研时 spawn | 只读代码搜索 |
| trellis-implement | 主会话在编码阶段 spawn | 写代码，不 commit |
| trellis-check | 主会话在验证阶段 spawn | 自带验证 + 自修复循环 |

> Sub-agent 可并行执行。与 AI-Team 角色的关系：AI-Team 是高层角色编排（谁何时做什么），Sub-agent 是底层执行委托（独立进程跑具体任务），两者互补。

## 会话记忆

- 会话记录存储在 `.trellis/workspace/boss/`
- 新会话自动加载最近的工作记录
- 使用 `/record-session` 在会话结束时记录工作内容
- Journal 单文件不超过 2000 行

## 最佳实践

1. **Read before write**：`/start` 加载上下文，读 spec 再编码
2. **规范即代码**：踩坑后用 `/update-spec` 沉淀，后续会话自动参考
3. **自修复循环**：`trellis-check` sub-agent 内置重试，不再需要外部 Ralph Loop
4. **根因 > 症状**：修完 bug 后触发 `trellis-break-loop`，确保同类 bug 不再发生
5. **不跳步骤**：用 `/continue` 强制 AI 按工作流推进
6. **AI 不 commit**：代码由人工测试后手动提交
