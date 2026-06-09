---
apply: 始终
---

# zero-service 项目规则

## 项目画像

- Go 微服务项目。
- 核心技术栈：go-zero、Eino AI 框架、Trellis（`.trellis/`）。
- 协作方式：以 Trellis 任务、项目 spec、workflow 和技能路由驱动需求拆解、开发前检查、编码、测试和收尾。
- 开发时优先遵循 Go、go-zero、Eino、Trellis 约定，不套用 Java 分层和命名习惯。

## AI 工具边界

- `.aiassistant/rules/**` 是 GoLand AI / JetBrains AI 配置。
- `.opencode/rules/**` 是 OpenCode 规则配置。
- `.qoder/rules/**` 是 Qoder 规则配置。
- 三个目录下规则文件名和内容保持一致，便于后续覆盖同步。
- 当前项目不再依赖已删除的旧技能包或旧工作流入口；开发上下文以 Trellis SessionStart、当前任务材料和 `.trellis/spec/**` 为准。

## 规范层级

| 层级 | 位置 | 加载时机 |
|------|------|---------|
| 通用 AI 规则 | `.opencode/rules/ai-rule.md` / `.aiassistant/rules/ai-rule.md` / `.qoder/rules/ai-rule.md` | 会话规则加载时 |
| 项目规则 | `.opencode/rules/project-rule.md` / `.aiassistant/rules/project-rule.md` / `.qoder/rules/project-rule.md` | 会话规则加载时 |
| Trellis 项目规范 | `.trellis/spec/` | 开发前按任务范围按需读取 `backend/index.md`、`guides/index.md` 和任务相关具体规范 |
| Trellis 任务上下文 | `.trellis/tasks/` | 有活跃 Trellis 任务时读取 |

## 技术规范

编码规则、构建验证、质量门禁等所有技术细节统一放在 `.trellis/spec/` 中，由 `trellis-before-dev` 按需注入。开发或审查代码时优先遵循 spec，不依赖本文件中的技术指引。
