---
name: survey_echo_flow
description: |
  问卷(ask_form_input 等人机工具) → echo 工具 的联调流水线说明。
  实现为代码内 BuildSurveyEchoAgent / NewSurveyEchoAgentTool，不再作为独立 Solo Mode。
---

# Survey → Echo 联调流水线

## 何时使用

- 验证 **人机中断**（表单问卷）与 **echo** 工具链路；
- 需要把该流水线 **封装成 AgentTool** 挂到自定义 Agent / Deep 子编排里。

## 接入方式

| 方式 | 说明 |
|------|------|
| **代码** | `modes.BuildSurveyEchoAgent` 得到顺序 `adk.Agent`；`modes.NewSurveyEchoAgentTool` 得到 `tool.BaseTool` 供 `WithTools` 注册。 |
| **本 SKILL** | 渐进披露：先暴露 `name` + `description`，命中后再加载正文，减少无关轮次上下文。 |

## Eino：智能体即工具

`adk.NewAgentTool(ctx, agent)` 将 `adk.Agent` 包成 `tool.BaseTool`，与 Deep 的 `task` 委派同属「子能力工具化」。

## 可选 frontmatter：`context`

若 skill 中间件配置了 **AgentHub**，可在本文件 frontmatter 增加 `context: fork` 或 `fork_with_context`（见 `adk/middlewares/skill`）。当前 aisolo 默认未配 AgentHub，故不写 `context`，走内联加载。

## 流水线摘要

1. **Planner**：人机工具收集答案；回复末尾 `SURVEY_JSON: {...}`。
2. **Echoer**：解析 JSON，**仅调用一次** `echo`，再向用户确认完成。

## 相关代码

- `internal/modes/blueprint_survey_echo.go` — `BuildSurveyEchoAgent`
- `internal/modes/survey_echo_agent_tool.go` — `NewSurveyEchoAgentTool`
- `internal/modes/prompts.go` — `surveyEchoPlannerPrompt` / `surveyEchoEchoPrompt`
