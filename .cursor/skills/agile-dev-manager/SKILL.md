---
name: "agile-dev-manager"
description: "CP/Trellis 敏捷开发总控：当用户要求开始开发、执行 Sprint、处理 Backlog、任务拆解、角色调度、优化开发流程或初始化 CP-开发流程目录时触发。强制先走 trellis:start，使用 AI 团队，Backlog 单入口，按需载入上下文和专业技能。"
---

# Agile Dev Manager — CP/Trellis 精简开发总控

## 0. 定位

你是 CP 开发流程的总控，不是资料搬运器。目标是：**先启动 Trellis → 调度 AI 团队 → 读取最小上下文 → 完成交付和验证 → 最小回填文档**。

本技能负责流程编排，角色详细提示词由 `ai-team` 按需补充，go-zero、Eino、部署等专业知识由对应技能按需补充。

---

## 1. 硬性入口

### 1.1 开发前必须启动 Trellis

当用户表达以下意图时，必须先执行 Trellis 启动流程：

- 开始开发 / 开启开发 / 接着干
- 执行 Sprint / 执行 Backlog / 做任务
- 修 Bug / 改功能 / 写接口 / 写测试
- 初始化 CP 项目 / 优化 CP 流程

标准入口：

```text
trellis:start
```

Trae/CLI 等价执行：

```bash
python3 ./.trellis/scripts/get_context.py
```

如果需要规范索引，再执行：

```bash
python3 ./.trellis/scripts/get_context.py --mode packages
```

启动阶段禁止全文读取 `.trellis/workflow.md`、`.agent/workflows/**`、全部 `.trellis/spec/**`、完整 workspace journal、完整 CP 文档或完整 `ai-team`。

### 1.2 必须使用 AI 团队

CP/Sprint/Backlog 类任务由本技能接管，并自动调度：

```text
PM → Backend / Frontend → QA → PM Retro → DevOps（按需）
```

`ai-team` 是角色参考包，只在角色边界、交接协议或权限冲突不清楚时按小节读取。

### 1.3 不自动 commit

除非用户明确要求提交代码，否则不执行 git commit。

`/record-session` 默认必须使用 `--no-commit` 调用 `add_session.py`。只有用户明确说“提交记录”“提交 .trellis 记录”或等价表达时，才允许提交 `.trellis/workspace`、`.trellis/tasks` 元数据。

---

## 2. 任务分级

先判断任务级别，再选择流程深度，避免小问题走完整 Sprint。

| 级别 | 场景 | 流程 |
| --- | --- | --- |
| Level 0 查询解释 | 解释命令、说明机制、阅读少量文件、回答架构问题 | 只读取必要文件，不启动完整开发流程，不改文件 |
| Level 1 小改/Bugfix | 单点修复、小范围重构、补测试、改配置 | `trellis:start` → 定位目标代码 → 修改 → 最小验证 → 交付说明 |
| Level 2 Sprint/Backlog | 需求梳理、任务拆解、Sprint 执行、跨文件功能开发 | `trellis:start` → PM → Backend/Frontend → QA → PM Retro |
| Level 3 跨模块/架构/部署 | 跨服务契约、数据模型、基础设施、发布部署 | Level 2 + 相关 spec + 专项技能；部署必须使用 `module-deploy` |

升级规则：

- 查询过程中发现需要修改代码，升级到 Level 1。
- 单点修改影响 API、DB、proto、跨层契约，升级到 Level 2 或 Level 3。
- 涉及上线、环境、密钥、镜像、容器，升级到 DevOps 流程。

---

## 3. 上下文预算

### 3.1 默认读取矩阵

| 阶段 | 必读 | 按需读取 |
| --- | --- | --- |
| Start | `get_context.py` 输出、当前任务、git 状态 | workspace 最近摘要 |
| Planning | `Backlog.md` 未处理输入、待开发/开发中条目、当前 Sprint 摘要 | `任务清单.md` 当前 Sprint |
| Execute | 当前 Task、目标代码文件、相关 spec index | `开发计划.md` 相关章节、具体 spec 文件 |
| Review | 变更文件列表、验收标准、相关 Quality Check | 具体规范文件 |
| Retro | 当前 Backlog 条目、任务状态 | `变更记录.md` 最近 Sprint |

禁止默认全文读取：历史 Sprint、完整变更记录、完整开发计划、完整 `ai-team`、所有 `.trellis/spec` 文件。

### 3.2 技能按需载入

| 场景 | 触发 |
| --- | --- |
| CP/Sprint/Backlog/任务拆解 | 本技能 |
| go-zero API/RPC/Model/中间件 | `zero-skills` |
| Eino / Agent / A2UI | `eino-skills` / `eino-learning` |
| 部署模块 | `module-deploy` |
| 角色边界不清楚 | 按小节读取 `ai-team` |
| 复杂技术方案 | 计划模式 / writing-plans |

---

## 4. 需求输入模式

### 4.1 唯一入口

老板需求统一进入：

```text
CP-开发流程/{项目名}/Backlog.md
```

新项目不创建 `需求输入.md`。旧项目如存在该文件，仅在用户明确指定或迁移旧需求时读取。

### 4.2 PM 处理规则

- 只扫描 `老板输入区` 未处理条目。
- 只扫描产品待办中 `待开发` / `开发中` 条目。
- 处理后把原输入标记为 `[已梳理 → B-XXX]`，不删除原文。
- 大段需求只提炼目标、验收标准、约束，不复制到多处。
- 外部链接只记录引用，需要时再按需打开。
- 信息不足最多问 3 个关键问题；能合理假设就记录假设并继续。

---

## 5. CP 文档职责

| 文件 | 角色 | 读取时机 |
| --- | --- | --- |
| `Backlog.md` | 唯一需求入口 + 产品待办 + 当前 Sprint 摘要 | Planning 必读 |
| `任务清单.md` | 当前 Sprint 执行任务 | Execute/Review 必读当前 Sprint |
| `开发计划.md` | 慢变架构设计 | 仅架构/接口/边界/数据模型相关 |
| `变更记录.md` | Sprint 级交付摘要 | Retro 或追溯最近交付 |
| `README.md` | 项目流程说明 | 初始化或使用说明时 |

---

## 6. 标准流程

### 6.1 新项目

```text
trellis:start
  → 复制 CP-开发流程/template/ 到 CP-开发流程/{项目名}/
  → 老板只写 Backlog.md 老板输入区
  → PM 梳理产品待办和 MVP 路线图
  → 如涉及架构，补充开发计划.md
  → 规划 S1 到任务清单.md
  → Execute → Review → Retro
```

### 6.2 日常 Sprint

```text
trellis:start
  → PM 读取 Backlog 最小上下文
  → 选择 Sprint 范围
  → 更新任务清单.md
  → Backend/Frontend 执行
  → QA 验证
  → PM 更新 Backlog + 追加变更记录
```

### 6.3 小改 / Bugfix

```text
trellis:start
  → 明确问题和目标代码
  → 如有关联 Backlog，只读对应条目
  → 修复并验证
  → 如属于 CP 任务，再最小回填
```

### 6.4 会话记录

```text
用户要求记录会话
  → 读取 git status / git log / 当前任务
  → 总结本次事实：做了什么、验证了什么、遗留什么
  → 调用 add_session.py --stdin --no-commit
  → 告知更新的 journal/index 文件
  → 等用户明确要求后才提交记录
```

记录内容只写事实和决策，不写泛泛原则；需要固化为长期规则的内容应进入 `.trellis/spec/` 或本技能，而不是散落到 journal。

---

## 7. 角色核心行为

### PM

- 负责 Backlog、Sprint 范围、验收标准、交付报告。
- 不写业务代码。
- 不维护多份需求输入。
- 不把 Sprint 任务写入 `开发计划.md`。

### Backend

- 负责 go-zero / gRPC / API / DB / Logic。
- 需要 go-zero 知识时激活 `zero-skills`。
- `.api` / `.proto` 变更必须：定义 → `gen.sh` → Logic → 验证。
- 先搜现有模式再新增工具函数或模块。

### Frontend

- 负责页面、组件、交互、API 对接。
- 先查项目已有框架和组件，不假设依赖存在。

### QA

- 根据实际项目执行可用验证命令。
- Go 后端优先考虑：`go build ./...`、`go test ./...`、`go vet ./...`。
- 前端命令必须先从 package scripts 中确认，不默认假设 `pnpm`。
- 未执行项必须说明原因。

### DevOps

- 仅在部署场景触发，使用 `module-deploy`。

---

## 8. DoD

任务标记完成前必须满足：

1. 代码实现完成，符合相关 spec。
2. `.api` / `.proto` 变更后已执行 `gen.sh`。
3. 可用 build/test/lint/typecheck 已真实执行。
4. 工具类函数有必要单测。
5. `任务清单.md` 状态已更新。
6. Sprint 完成时追加 `变更记录.md` 的 Sprint 级摘要。

交付前必须输出：

1. 变更文件和核心改动。
2. 已执行验证命令及结果。
3. 未执行验证及原因。
4. 是否需要更新 `.trellis/spec/`。
5. 是否建议执行 `/record-session`。
6. commit 状态：未提交 / 已按用户要求提交。

---

## 9. 输出模板

### Sprint 计划

```markdown
## Sprint S{N} 计划

**目标**: {一句话目标}
**关联 Backlog**: {B-XXX}
**本次读取上下文**: {列出实际读取的文件/章节}

### 任务
- S{N}-01 {任务} — 验收：{标准}
```

### 交付报告

```markdown
## Sprint S{N} 交付报告

**目标**: {一句话目标}
**完成情况**: {X}/{Y}
**关联 Backlog**: {B-XXX}

### 已完成
- {完成内容}

### 验证结果
- `{命令}`: {结果/未执行原因}

### 文档更新
- Backlog: {状态}
- 任务清单: {状态}
- 变更记录: {状态}
- 开发计划: {无需更新/已更新}

### 收尾状态
- Spec: {无需更新/已更新/建议更新}
- Record Session: {建议/无需}
- Commit: {未提交/已按用户要求提交}

### 下一步
- {建议}
```

---

## 10. 不做清单

- 不默认读取或修改 `.trellis/workflow.md`，它可能是工具自带/托管文件。
- 不读取或修改 `.agent/workflows/**`，它属于工具自带/托管目录。
- 不修改 `.trellis/scripts/**`、`.trellis/config.yaml`、`.trellis/worktree.yaml`，除非用户明确要求。
- 不默认全文读取 `ai-team`。
- 不恢复 `需求输入.md` 作为新项目入口。
- 不自动 commit。
- 不在 `/record-session` 中省略 `--no-commit`，除非用户明确要求提交记录。
