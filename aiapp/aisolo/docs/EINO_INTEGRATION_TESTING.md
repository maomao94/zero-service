# Eino 集成联测指南（aisolo）

面向「只用 Eino ADK 现成 Agent 形态 + 内置工具」的联测，不涉及单独业务服务。

## 1. 工具（Kit / Policy）

| 目标 | 做法 |
|------|------|
| compute / io / human 是否注册 | 启动 aisolo，选 **agent** mode，让模型调用 `echo`、`calculator`、`time`、某一 `ask_*`。 |
| 策略裁剪 | 在 `modes` 某 Blueprint 里改 `tool.NewPolicy().Allow...`，对比 list / 调用失败情况。 |

## 2. Skill（渐进披露 + 可选 fork）

| 目标 | 做法 |
|------|------|
| 目录加载 | `etc/aisolo.yaml` → `Skills.Dir`（默认 `./skills`），放子目录 + `SKILL.md`。 |
| 内联正文 | frontmatter 仅 `name` + `description`（不写 `context`），模型命中后再拉全文。 |
| fork / fork_with_context | 需在 `buildSkillHandlers` 侧为 `skill.Config` 提供 **AgentHub**（当前未接）；接好后再在 `SKILL.md` 写 `context: fork`。 |

## 3. Deep / Workflow / Supervisor 与子 Agent

| 形态 | 联测要点 |
|------|----------|
| **Deep** | 选 **deep** mode；观察 `deep_research` / `deep_synthesis` 子 Agent 是否在合适步骤被 task 委派（前端 SSE 中带 `agent_name`）。 |
| **Workflow** | **workflow** mode：顺序子 Agent 输出是否串联；并行/循环若启用则看事件顺序。 |
| **Supervisor** | **supervisor** mode：`researcher` vs `interactor` 分工与中断归属。 |

## 4. 前端可观测性

- 流式事件 `message.*` / `tool.call.*` / `interrupt` 的 `data.agent_name` 来自 ADK `AgentEvent.AgentName`。
- Solo Web：消息区与中断面板顶部展示 **Agent 名**，便于区分主 Agent / 子 Agent。
