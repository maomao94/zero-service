<!-- TRELLIS:START -->
# Trellis Instructions

These instructions are for AI assistants working in this project.

This project is managed by Trellis. The working knowledge you need lives under `.trellis/`:

- `.trellis/workflow.md` — development phases, when to create tasks, skill routing
- `.trellis/spec/` — package- and layer-scoped coding guidelines (read before writing code in a given layer)
- `.trellis/workspace/` — per-developer journals and session traces
- `.trellis/tasks/` — active and archived tasks (PRDs, research, jsonl context)

If a Trellis command is available on your platform (e.g. `/trellis:finish-work`, `/trellis:continue`), prefer it over manual steps. Not every platform exposes every command.

If you're using Codex or another agent-capable tool, additional project-scoped helpers may live in:
- `.agents/skills/` — reusable Trellis skills
- `.codex/agents/` — optional custom subagents

Managed by Trellis. Edits outside this block are preserved; edits inside may be overwritten by a future `trellis update`.

<!-- TRELLIS:END -->

## Agent skills

- 工作项由 Trellis 管理（`.trellis/tasks/`），不用 GitHub Issues。Matt 的 `to-tickets` / `to-spec` 等技能的产物落到 Trellis 任务（`task.py create`），不要建 GitHub issue。
- 不维护独立术语表和 ADR；知识沉淀走 `.trellis/spec/`（跨任务）或任务 `design.md`（任务内）。

## Workflow 分工：Trellis × Matt Pocock Skills

本项目同时安装 Trellis（任务生命周期管理）和 Matt Pocock 技能集（工程方法层）。**Matt 技能始终可用，不依赖 Trellis**；Trellis 是可选的生命周期包装，不是使用 Matt 技能的前提。分工原则：**Trellis 管生命周期，Matt 管方法**。

不确定该用哪个 Matt 技能时，先调用 `ask-matt`（Matt 技能集自带的路由器，覆盖 idea → ship 全流程）确认。

### 两种模式

**模式 A · 独立模式**（无 Trellis 任务，含用户拒绝建任务的会话）——直接调用 Matt 技能，按技能自身约定工作：

| 意图 | 技能 | 产物落点 |
| --- | --- | --- |
| 打磨想法 / 压力测试计划 | `grilling` / `grill-me` | 聊天内收敛 |
| 主题调研 | `research` | `docs/research/<topic>.md`（或用户指定位置） |
| 测试先行实现 | `tdd` | 代码 + 测试 |
| bug / 性能诊断 | `diagnosing-bugs` | 修复 + 回归测试 |
| 变更 / 分支评审 | `code-review` | 结论进聊天，修复进代码 |
| 一次性原型 | `prototype` | 临时目录，用后即弃 |

独立模式中若发现工作量大、需跨会话续做、或需要提交纪律，主动建议用户转入 Trellis（征得同意后 `task.py create`）。

**模式 B · 集成模式**（Trellis 任务内）——Trellis 管生命周期门禁与工件落点，Matt 技能作为阶段内的方法层，见下方映射表。

模式 B 的深度集成由 **matt workflow 变体**（`.trellis/workflows/matt.md`）提供：Matt 方法已织入 phase 步骤与每轮 breadcrumb，当前设为个人默认（`.trellis/.developer` 的 `workflow=matt`）。用法：

- 新任务直接选用：`task.py create "<标题>" --workflow matt`
- 切换 active task 的 workflow：`task.py workflow matt`（`--clear` 恢复默认解析链）
- 退回标准流程：`.trellis/.developer` 移除 `workflow=matt`

未选 matt 变体时退回标准 native 流程（全局 `workflow.md` 保持默认、不含 Matt 路由）；Matt 技能仍可按本节规则直接调用。

### 路由优先级

1. **Inline 简单任务**（只读问答、单文件小修）→ 直接处理，不建任务。
2. **中等工作量、单会话可完成、或用户拒绝建任务** → 模式 A：Matt 技能独立使用，不建任务。
3. **需要跨会话状态、规划工件、提交纪律** → 模式 B：走 Trellis 流程（consent → planning → `task.py start` → commit），阶段内部调用 Matt 技能。

### 阶段 × 技能映射（模式 B · 集成模式）

| Trellis 阶段 | Trellis 负责（owner） | Matt 技能（方法层） | 产物落点 |
| --- | --- | --- | --- |
| 1.1 需求探索 | `trellis-brainstorm`、`prd.md` | `grill-me`（需求拷问）、`to-questionnaire`（向他人要信息） | `{TASK_DIR}/prd.md` |
| 1.1 大需求拆分 | parent/child 任务树 | `to-tickets` 垂直切片法；超大规划用 `wayfinder` | 子任务目录 |
| 1.2 研究 | `research/` 持久化 | `research`（一手来源调研） | `{TASK_DIR}/research/` |
| 1.4 前设计验证 | — | `prototype`（一次性原型） | `{TASK_DIR}/research/` |
| 2.1 实现 | `trellis-implement` 子代理 | `tdd`（红绿循环）、`diagnosing-bugs`（六阶段诊断） | 代码 + 回归测试 |
| 2.2 质量检查 | `trellis-check` 子代理 | `code-review`（标准 + 规格两轴） | 修复进代码 |
| 3.2 调试回顾 | `trellis-break-loop` | `retro`（会话级改进） | spec / journal |
| 3.3 知识沉淀 | `trellis-update-spec` → `.trellis/spec/` | — | `.trellis/spec/` |
| 3.4 提交 | 批量提交协议（唯一 owner） | 不接管 | git commits |

### 冲突裁决（模式 B）

- **一个阶段只有一个 workflow owner**：Trellis 的阶段门禁（consent → planning → start → commit）永远优先；Matt 技能只在阶段内部作为方法使用。
- **产物必须落盘**：Matt 技能的调研、设计、评审结论写入当前任务目录（`{TASK_DIR}/research/` 等）或 `.trellis/spec/`，不得只留在聊天里。
- **提交纪律**：Matt 技能（如 `implement`）指示 commit 时，一律以 Trellis Phase 3.4 批量提交协议为准；未经用户确认不提交、不推送。
- **技能调用名**：Matt 技能按名称调用（`tdd`、`code-review` 等），安装目录带 `mcp-` 前缀（`~/.config/opencode/skills/mcp-*`），安装与版本见 `docs/matt-pocock-skills.md`。
