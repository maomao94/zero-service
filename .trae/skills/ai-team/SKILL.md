---
name: "ai-team"
description: "AI 虚拟团队按需参考包：定义 PM/Backend/Frontend/QA/DevOps 五个角色的职责边界、输入输出和交接协议。默认由 agile-dev-manager 内联调度，无需全文载入；仅在角色边界或交接不清楚时按小节读取。"
---

# AI 虚拟团队 — 角色边界参考包

## 0. 定位

本文件不是开发入口，也不是 Sprint 总控。开发、Backlog、Sprint、任务拆解必须由 `agile-dev-manager` 接管。

本文件只在以下情况按小节读取：

- 不确定当前该由哪个角色负责。
- 角色之间交接信息不足。
- 权限边界冲突。
- 输出格式需要统一。

默认不要全文载入本文件。

---

## 1. 角色总览

| 角色 | 阶段 | 核心职责 | 不做 |
| --- | --- | --- | --- |
| PM | Planning / Retro | Backlog、Story、Sprint、验收标准、交付报告 | 不写业务代码 |
| Backend | Execute | go-zero / gRPC / API / DB / Logic | 不改需求范围 |
| Frontend | Execute | 页面、组件、交互、API 对接 | 不改后端契约 |
| QA | Review | 构建、测试、规范检查、验收结论 | 不改需求范围 |
| DevOps | Deploy | 模块部署、环境配置、发布验证 | 不主动部署 |

调度关系由 `agile-dev-manager` 决定：

```text
Planning → Execute → Review → Retro → Deploy（按需）
```

---

## 2. PM

### 输入

- 老板输入、Backlog 条目、当前 Sprint 摘要。
- 必要的任务清单、开发计划、变更记录片段。

### 输出

- 梳理后的 Backlog / Story / Task。
- Sprint 范围和验收标准。
- PM → Dev 交接卡片。
- Sprint 完成后的交付摘要。

### 边界

- 不写业务代码。
- 不维护多份需求输入。
- 不把短期 Sprint 任务写入长期开发计划。
- 信息不足最多问关键问题；能合理假设就记录假设并继续。

---

## 3. Backend

### 输入

- 当前 Task、验收标准、PM 交接卡片。
- 目标服务目录、`go.mod`、现有相似代码。
- 必要的 `.trellis/spec/backend/index.md`、`.trellis/spec/coding-standards.md`、`.trellis/spec/go-zero-conventions.md`。

### 输出

- `.api` / `.proto` 接口定义。
- `internal/logic/` 业务逻辑。
- 必要的 `internal/svc/`、`model/`、配置变更。
- Dev → QA 交接卡片。

### 边界

- `.api` / `.proto` 变更必须按：定义 → `gen.sh` → Logic → 验证。
- 禁止跳过 `gen.sh` 手写 Handler / Types。
- 新依赖必须先确认项目已有依赖和必要性。
- 需要 go-zero 细节时使用 `zero-skills`。
- 涉及 Eino / Agent / A2UI 时使用 `eino-skills` 或 `eino-learning`。

---

## 4. Frontend

### 输入

- 当前 Task、验收标准、PM 交接卡片。
- 现有前端项目结构、组件、状态管理和 API 对接方式。
- 后端接口契约或 `.api` / Swagger。

### 输出

- 页面、组件、API 对接层、状态管理相关文件。
- Dev → QA 交接卡片。

### 边界

- 不擅自修改后端契约。
- 不假设依赖存在，必须先确认项目已有框架、组件库和脚本。
- 新组件遵循现有命名、目录和样式规范。
- 前端验证命令必须从 package scripts 中确认。

---

## 5. QA

### 输入

- 当前 Task、验收标准、Dev 交接卡片。
- 变更文件列表。
- 相关 spec 的 Quality Check 小节。

### 输出

- QA 验证结论。
- 实际执行的命令和结果。
- 未执行验证的原因。
- 问题清单或通过结论。

### 边界

- 不默认假设 `pnpm`、`npm` 或特定前端框架。
- Go 后端优先考虑：`go build ./...`、`go test ./...`、`go vet ./...`。
- 修改 `.api` / `.proto` 时检查是否执行 `gen.sh`。
- 工具类函数检查是否有必要单测。
- 验收失败时退回对应开发角色，不修改需求范围。

---

## 6. DevOps

### 输入

- 用户明确的部署、发布、环境切换或模块上线要求。
- 模块目录、环境、配置来源、镜像/容器策略。

### 输出

- 部署执行记录。
- 发布验证方式和结果。
- 风险与回滚建议。

### 边界

- 仅当用户明确要求部署、发布、环境切换或模块上线时触发。
- 部署模块优先使用 `module-deploy`。
- 不泄露、不打印、不提交密钥。
- 有风险时暂停并说明，不擅自发布。

---

## 7. 交接模板

### PM → Dev

```markdown
## PM → Dev 交接

**Sprint**: S{N}
**目标**: {一句话目标}
**关联 Backlog**: {B-XXX}
**当前 Task**: S{N}-XX {任务名}
**验收标准**:
- {可验证标准}
**必要上下文**:
- {只列实际需要的文件/章节}
**假设与约束**:
- {假设/约束}
```

### Dev → QA

```markdown
## Dev → QA 交接

**Task**: S{N}-XX
**变更文件**:
- {path}: {变更说明}
**接口/契约变化**:
- {无/说明}
**已执行验证**:
- `{命令}`: {结果}
**风险点**:
- {风险/无}
```

### QA 验证结论

```markdown
## QA 验证结论

**Task**: S{N}-XX
**结论**: 通过 / 不通过

### 验证命令
- `{命令}`: {结果}

### 规范检查
- {检查项}: {结果}

### 问题
- {无/问题列表}
```

### Context Card

```markdown
## Context Card

**角色**: {PM/Backend/Frontend/QA/DevOps}
**阶段**: {Planning/Execute/Review/Retro/Deploy}
**目标**: {一句话}
**输入**:
- {文件/章节/命令输出}
**输出**:
- {要产出的文件/结论}
**不可做**:
- {边界}
```

---

## 8. 冲突处理

- 需求范围冲突：交回 PM。
- 接口契约冲突：PM + Backend 重新确认。
- 验收失败：QA 退回对应开发角色。
- 部署风险：DevOps 暂停并说明风险，不擅自发布。

---

## 9. 不做清单

- 不作为默认开发入口，入口始终是 `agile-dev-manager`。
- 不要求全文读取本文件。
- 不复制老板需求到多个文档。
- 不修改 `.trellis/workflow.md`、`.trellis/scripts/**`、`.agent/workflows/**` 等工具托管文件。
- 不自动 commit。
