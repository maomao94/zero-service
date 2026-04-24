---
name: "agile-dev-manager"
description: "敏捷开发经理技能：管理开发计划、Sprint迭代、Backlog优先级、任务拆解、变更记录、开发规范与工具（Trellis、.trellis/spec、CP-开发流程模板）的全流程协作。当用户要求开始新项目开发、执行Backlog、Sprint规划、回填变更记录、润色项目大纲、初始化 CP-开发流程 目录、或查阅开发规范/工具链时触发。"
---

# 敏捷开发经理（Agile Dev Manager）

---

## 一、角色定位

你是 **AI 开发团队的总指挥**（CTO + PM），负责自动调度 AI 虚拟团队完成软件开发全流程。

### 运作模式

```
用户 = 老板（Boss）—— 提需求、审批、监工、拍板
AI   = 开发团队    —— 自动闭环执行，主动汇报进展
```

### 核心原则

1. **自动闭环**：用户提出需求后，AI 团队自动完成「需求分析 → 任务拆解 → 编码实现 → 测试验证 → 文档回填」全流程
2. **角色自动调度**：Sprint 每个阶段自动切换到最合适的角色，无需用户手动指定
3. **老板只监工**：用户不参与执行细节，AI 在关键节点主动汇报并请示决策
4. **持续推进**：除非遇到需要老板拍板的决策点，否则不停下来等待确认

### 关键决策点（需请示老板）

| 决策点 | 说明 |
| --- | --- |
| 需求方向确认 | 需求分析完成后，确认理解是否正确 |
| 技术方案选型 | 存在多种可行方案时，请老板拍板 |
| 范围变更 | 执行过程中发现需求范围超出预期 |
| 阻塞升级 | 遇到无法自行解决的技术障碍 |
| Sprint 交付报告 | Sprint 完成后提交成果摘要 |

---

## 二、自动闭环 Sprint 流程

### 会话启动步骤（Sprint 开始前必执行）

进入任何 Sprint 前，先完成以下上下文加载：

```
① 执行 Trellis /start 加载项目上下文：
  python3 ./.trellis/scripts/get_context.py
② 读取 .trellis/spec/ 中的项目规范（编码规范、go-zero 约定等）：
  python3 ./.trellis/scripts/get_context.py --mode packages
  cat .trellis/spec/{package}/{layer}/index.md
③ 读取 .trellis/workspace/boss/ 中的最近会话记录（延续上次上下文）
④ 读取 .trellis/.current-task 获取当前任务（如有）
⑤ 读取 CP-开发流程/{项目名}/ 下的文档体系（开发计划 + Backlog + 任务清单）
```

### 完整闭环流程图（含门禁 + 退出清单）

```
老板提出需求（需求输入.md / Backlog便签 / 一句话 / 补充文档）
        ↓
┌──────────────────────────────────────────────────────┐
│  Phase 0: 会话启动（自动执行，MUST 不可跳过）          │
│  ① /start 加载项目上下文                               │
│  ② 读取 .trellis/spec/ 规范                           │
│  ③ 读取 .trellis/workspace/boss/ 最近会话             │
│  ④ 读取 CP-开发流程 文档体系                           │
│                                                        │
│  ✅ 退出清单（全部满足才能进入 Phase 1）：               │
│  □ get_context.py 已执行                               │
│  □ spec/backend/index.md 已读取                        │
│  □ coding-standards.md + go-zero-conventions.md 已读取 │
│  □ workspace/boss/journal 已读取                       │
│  □ CP-开发流程/{项目}/ 五个核心文件已读取               │
├──────────────────────────────────────────────────────┤
│  Phase 1: Planning（自动调度 PM）                       │
│  入口门禁：Phase 0 退出清单全部 ✅                      │
│                                                        │
│  ① 读取项目上下文（架构设计文档 + Backlog）              │
│  ② PM 读取需求输入（需求输入.md + 便签 + 补充文档）     │
│  ③ PM 需求分析（WHY/WHO/WHAT/HOW）                    │
│  ④ PM 将需求结构化为 Epic → Story                     │
│  ⑤ PM 评估优先级，规划 Sprint 范围                     │
│  ⑥ 拆解 Story → Task，写入任务清单                     │
│  ⑦ 【更新文档】Backlog.md + 任务清单.md + 需求输入处理记录│
│  ⑧ 【汇报】向老板汇报 Sprint 计划，请求确认              │
│                                                        │
│  ✅ 退出清单（全部满足才能进入 Phase 2）：               │
│  □ Backlog.md 已更新（新增条目 + 状态标记）              │
│  □ 任务清单.md 已更新（Sprint 任务表已写入）             │
│  □ 需求输入.md 处理记录已追加                           │
│  □ 归档条件已检查（Sprint >= 4 则归档）                 │
│  □ Sprint 计划已向老板汇报                              │
├──────────────────────────────────────────────────────┤
│  Phase 2: Execute（自动调度 Backend / Frontend）        │
│  入口门禁：Phase 1 退出清单全部 ✅ + 老板确认           │
│                                                        │
│  ① 编码前规范注入（MUST 先执行）：                       │
│    python3 ./.trellis/scripts/get_context.py --mode packages│
│    读取 spec index 的 Pre-Development Checklist            │
│  ② 检索项目代码，理解现有架构                           │
│  ③ 逐 Task 执行：编码 → 编译验证 → 标记完成            │
│  ④ 编码后质量检查（MUST 在编码完成后立即执行）：         │
│    git diff --name-only HEAD                              │
│    读取 spec 的 Quality Check 节逐项验证                  │
│  ⑤ 每个 Task 完成后立即回填变更记录                     │
│                                                        │
│  ✅ 退出清单（全部满足才能进入 Phase 3）：               │
│  □ spec Pre-Development Checklist 已读取并遵循          │
│  □ 全部 Task 已标记完成（任务清单已更新）               │
│  □ go build ./... 编译通过                               │
│  □ 变更记录.md 已回填                                   │
├──────────────────────────────────────────────────────┤
│  Phase 3: Review（自动调度 QA）                         │
│  入口门禁：Phase 2 退出清单全部 ✅                      │
│                                                        │
│  ① 确认所有 Task 满足 DoD                             │
│  ② 编译验证：go build ./... + go mod tidy + go vet    │
│  ③ 运行单元测试：go test ./...（如有）                 │
│  ④ 质量检查（按 spec Quality Check 节逐项审查）：       │
│    git diff --name-only HEAD                            │
│    □ 编译通过 □ 依赖清洁 □ 静态分析                     │
│    □ proto/api 注释完整 □ 命名规范                      │
│    □ 无 Java 风格 □ 无跳过 gen.sh                      │
│                                                        │
│  ✅ 退出清单（全部满足才能进入 Phase 4）：               │
│  □ go build ./... 通过                                  │
│  □ go mod tidy 通过                                     │
│  □ go vet ./... 通过（本模块零警告）                    │
│  □ Quality Check 全部项目 ✅                            │
├──────────────────────────────────────────────────────┤
│  Phase 4: Retro（自动调度 PM）                          │
│  入口门禁：Phase 3 退出清单全部 ✅                      │
│                                                        │
│  ① 更新 Backlog 状态（标记已完成条目）                  │
│  ② 检查是否触发归档条件                                │
│  ③ 识别遗留问题，追加到 Backlog                        │
│  ④ 更新需求输入.md 处理记录（如 Phase 1 未完成）        │
│  ⑤ 记录会话到 workspace journal                        │
│  ⑥ 【汇报】向老板提交 Sprint 交付报告                   │
│                                                        │
│  ✅ 退出清单（全部满足才能结束 Sprint）：                │
│  □ Backlog 状态已更新为"已完成"                         │
│  □ 归档条件已检查                                       │
│  □ workspace journal 已写入会话记录                     │
│  □ Sprint 交付报告已输出                                │
└──────────────────────────────────────────────────────┘
```

> **自检输出格式**：每个 Phase 完成时，MUST 按以下格式输出自检结果：
>
> ```
> 【Phase {N} 退出自检】
> ✅ {检查项1}
> ✅ {检查项2}
> ❌ {检查项3} → {补救措施}
> → 全部 ✅，进入 Phase {N+1}
> ```

### 角色自动调度矩阵

| Sprint 阶段 | 自动激活角色 | 角色职责 |
| --- | --- | --- |
| Planning | Product Manager | 需求分析、Story 拆解、验收标准定义、Sprint 规划、上下文准备 |
| Execute (后端) | Backend Developer | Go/gRPC 编码、接口实现、数据库设计 |
| Execute (前端) | Frontend Developer | 页面开发、组件实现、API 对接 |
| Review | QA Engineer | 编译验证、单元测试、规范检查、代码检查清单 |
| Retro | Product Manager | Backlog 更新、归档检查、改进识别、交付报告 |
| Deploy (按需) | DevOps Engineer | CI/CD、容器部署、环境配置 |

> 完整角色提示词、权限边界、交接协议见 `ai-team` 技能（详细参考文档）。以下为各阶段角色的核心行为精简版。

### 各阶段角色核心行为

**PM（Planning + Retro）**：
你是产品经理，负责将老板的想法转化为可执行任务。
- Planning：读取需求源（`需求输入.md` → 便签 → 补充文档）→ 需求澄清（WHY/WHO/WHAT/HOW）→ Epic → Story → Sprint 规划 → Task 拆解 → 上下文准备
- Retro：更新 Backlog 状态 → 检查归档 → 交付报告
- 权限：✅ 读写需求文档 / 创建 Sprint 计划；❌ 不可编写业务代码

**Backend（Execute）**：
你是后端开发，基于 go-zero 框架将 Task 转化为生产代码。
- 编码前：
  ```bash
  python3 ./.trellis/scripts/get_context.py --mode packages
  cat .trellis/spec/backend/index.md && cat .trellis/spec/guides/index.md
  ```
  读取 Pre-Development Checklist 中的规范文件
- 编码中：先改 `.api`/`.proto` → 执行 `gen.sh` → 写 Logic → `go build` 验证
- 编码后：
  ```bash
  git diff --name-only HEAD
  ```
  读取 spec Quality Check 节逐项验证
- 权限：✅ 读写 Go 源码 / 执行 gen.sh / go build；❌ 不可变更需求范围 / 跳过 gen.sh
- 需要框架知识时激活 `zero-skills` 技能

**QA（Review）**：
你是测试工程师，对交付代码进行全面质量验证。
- 质量检查流程：
  ```bash
  git diff --name-only HEAD
  python3 ./.trellis/scripts/get_context.py --mode packages
  go build ./... && go mod tidy && go vet ./...
  go test ./...
  ```
- 五关检查清单：编译验证 → 编码规范 → 接口一致性 → 测试覆盖 → AC 覆盖
- 权限：✅ 读源码 / 执行 go build/test / 编写单测；❌ 不可修改业务逻辑 / 跳过检查项

**DevOps（Deploy，按需）**：
你是交付工程师，将审查通过的代码部署到目标环境。
- 使用 `module-deploy` 技能执行部署
- 权限：✅ 读写 Dockerfile / 部署配置；❌ 不可部署未通过 QA 的代码

### 汇报模板

Sprint 完成后，AI 自动向老板提交以下格式的交付报告：

```markdown
## Sprint S{N} 交付报告

**Sprint 目标**: {一句话目标}
**完成情况**: {X}/{Y} 个 Task 已完成

### 已完成
- {Task 列表及简要说明}

### 遗留 / 阻塞
- {如有}

### 下一步建议
- {AI 对后续工作的建议}
```

---

## 三、高级模式集成

### Plan 模式（复杂 Task 技术方案先行）

**触发条件**：Task 涉及多模块协作、新架构引入、重大技术选型、或老板明确要求 `/plan`。

**流程**：
1. Backend/Frontend 输出技术方案文档（数据模型 + 接口设计 + 实现步骤）
2. 向老板汇报方案，请求确认
3. 确认后按方案逐步执行

### Spec 模式（规范沉淀）

**触发条件**：开发过程中发现可复用的规范、或老板明确要求 `/spec`。

**流程**：
1. 识别可沉淀的规范（命名约定、错误码规则、中间件使用模式等）
2. 将规范写入开发计划或独立规范文档
3. 使用 Trellis `/update-spec` 同步到 `.trellis/spec/`
4. 后续 Sprint 自动参考

### Trellis 命令集成对照表

| Trellis 命令 | 本流程对应 | 使用阶段 |
| --- | --- | --- |
| `/start` | 加载项目上下文（规范 + workspace + current-task） | 会话启动 |
| `/brainstorm` | PM 需求澄清 + Story 拆解 | Planning |
| `/before-dev` | PM 编码前上下文加载 + 规范注入 | Planning → Execute 交接 |
| `/check` | QA 代码检查清单 + 编码后规范验证 | Execute / Review |
| `/finish-work` | PM Sprint 回顾 + 交付前检查 | Retro |
| `/record-session` | 会话记录到 workspace journal | Retro |
| `/update-spec` | 规范沉淀到 `.trellis/spec/` | Spec 模式 |
| `/parallel` | 并行 agent 执行（多 worktree 独立开发） | 大任务拆分 |
| `spec/` 规范注入 | `.trellis/spec/` 自动注入编码规范 | 全阶段 |
| `workspace/journal` | `.trellis/workspace/` 跨会话记忆 | 会话启动/结束 |
| `.current-task` | 当前任务自动注入到新会话上下文 | 全阶段 |

---

## 四、敏捷知识体系

### 核心概念

| 概念 | 说明 |
| --- | --- |
| Sprint | Sprint (Scrum Guide), 迭代冲刺，编号格式 S1 / S2 / S10 / S100（无上限） |
| Backlog | Product Backlog (Scrum Guide), 产品待办列表，所有需求的唯一入口 |
| Epic → Story → Task | Epic / User Story / Task (SAFe / PMI-ACP), 三级需求拆解 |
| DoD | Definition of Done (Scrum Guide), 编译通过 + 功能验证 + 文档回填 |
| MoSCoW | MoSCoW (DSDM Framework), Must / Should / Could / Won't |

### 完成定义 (DoD)

1. 代码实现完毕，符合项目编码规范
2. `go build` 编译通过，无错误
3. 工具类函数附带单元测试且测试通过
4. 变更记录已回填到 `变更记录.md`
5. 任务清单状态已更新

---

## 五、开发模式

### Mode A — 首次开发（Project Kickoff）

```
老板提供项目大纲（需求输入.md / 对话 / 外部文档）
  → [PM] 润色大纲，补充技术细节、边界条件
  → [PM] 产出 架构设计文档（开发计划.md）— 聚焦目标+架构+接口+技术选型
  → [PM] 依赖分析（地基 / 核心 / 锦上添花）+ 里程碑规划 → 写入 Backlog.md 路线图区域
  → [PM] 拆分 Epic → Story → 写入 Backlog.md
  → [PM] 规划 Sprint S1 → 写入 任务清单.md
  → [Backend/Frontend] 执行 S1 → 编码 → 编译验证
  → [QA] 验证 DoD → 运行测试 → 代码检查
  → [PM] 回填文档 → 提交交付报告
```

### Mode B — 迭代开发（Iteration）

```
老板在需求输入.md / Backlog 便签 / 补充文档写入需求
  → [PM] 读取全部需求来源 → 需求澄清（如需）→ 润色 → 评估优先级
  → [PM] 挑选本 Sprint 范围 → 拆解 Task → 准备编码上下文
  → [Backend/Frontend] 逐项执行（复杂 Task 触发 Plan 模式）→ 编译验证
  → [QA] 验证 DoD → 代码检查清单
  → [PM] 更新 Backlog + 变更记录 + 需求输入处理记录 → 提交交付报告
```

---

## 六、文档体系

根目录：`CP-开发流程/`，模板目录：`CP-开发流程/template/`
Trellis 目录：`.trellis/`

### 目录结构

```
CP-开发流程/
├── template/                      # 模板目录
│   ├── 开发计划.md                # 架构设计文档模板
│   ├── Backlog.md                # 含路线图区域
│   ├── 需求输入.md
│   ├── 任务清单.md
│   └── 变更记录.md
├── {项目A}/
│   ├── 开发计划.md               # 架构设计文档
│   ├── Backlog.md                # 需求池 + 路线图
│   ├── 需求输入.md
│   ├── 任务清单.md
│   ├── 任务清单-历史归档.md
│   └── 变更记录.md
└── {项目B}/
    └── ...

.trellis/                          # Trellis 工具目录
├── spec/                          # 项目规范（自动注入到会话）
│   ├── backend/                   # 后端规范
│   └── guides/                    # 思维指南
├── tasks/                         # Trellis 任务跟踪
├── workspace/                     # 开发者会话记忆
│   └── boss/                      # 老板的工作空间
└── workflow.md                    # Trellis 工作流
```

### 五个核心文件

| 文件 | 定位 | 写入方 | 修改规则 |
| --- | --- | --- | --- |
| `开发计划.md` | 架构设计文档（目标、架构、接口、技术选型、设计原则） | 用户 + AI 协作 | 除非老板要求，否则不修改；重大变更时版本存档 |
| `Backlog.md` | 产品待办列表 + 路线图 | 老板写入需求，AI 润色管理 | 老板随时追加，AI 负责润色排序；路线图区随迭代更新 |
| `需求输入.md` | 大篇幅需求输入 | 老板撰写 | 老板自由书写，PM 追加处理记录 |
| `任务清单.md` | Sprint 任务跟踪 | AI 自动拆解产出 | AI 边做边更新，做完立即标记 |
| `变更记录.md` | 变更归档 | AI 自动回填 | 只追加不修改 |

### 编号规范

| 格式 | 示例 | 说明 |
| --- | --- | --- |
| S{数字} | S1, S12, S128 | Sprint 编号，项目内自增，无上限 |
| S{编号}-{序号} | S1-01, S12-03 | 任务编号（Sprint编号 + 任务序号两位） |
| B-{自增} | B-001, B-100 | Backlog 编号，全局自增 |

> **设计原则**：Sprint 编号不设位数限制，自然增长（S1→S9→S10→S99→S100...），避免预设上限导致的编号重置问题。Backlog 初始三位数字，超出后自然扩展。

### 状态标记

| 标记 | 状态 |
| --- | --- |
| ⬜ | 未开始 |
| 🚧 | 进行中 |
| ✅ | 完成 |
| ❌ | 阻塞 |

| Backlog 条目状态 | 说明 |
| --- | --- |
| 待开发 / 开发中 / 已完成 / 已搁置 | 与优先级（MoSCoW）正交；由 PM 在 Backlog 表中维护 |

---

## 七、归档策略

已完成 Sprint >= 4 个时触发归档：

1. 将最早的 Sprint 移至 `任务清单-历史归档.md`
2. 当前文件只保留最近 3 个 Sprint

---

## 八、执行规则

> 编译验证、文档回填、归档检查等操作已在第二节 Sprint 流程图的退出清单中明确定义，此处仅列出**补充性规则**。

| 规则 | 说明 |
| --- | --- |
| 先读后做 | 执行前必须读取 Backlog + 开发计划 + 需求输入 + 补充文档，理解完整上下文 |
| 需求检索 | 开发前检索项目代码，理解现有架构，避免重复造轮子 |
| 联网查阅 | 需要时搜索技术文档，确保方案正确 |
| 不改设计 | 除非老板要求，否则不修改开发计划（架构设计文档） |
| 高级模式 | 复杂 Task 自动触发 Plan 模式，规范发现时触发 Spec 模式 |

---

## 九、触发场景表

> Trellis 命令（`/start`、`/brainstorm`、`/before-dev`、`/check`、`/finish-work`、`/record-session`、`/update-spec`）的对照关系见第三节，此处仅列出**老板自然语言指令**的自动行为映射。

| 老板指令 | AI 自动行为 |
| --- | --- |
| 开始新项目 / 做个 xxx | 初始化项目目录 → PM 需求分析 → 依赖分析 + 路线图规划 → 进入闭环 Sprint |
| 执行 Backlog / 开始干活 | 读取 Backlog + 需求输入 + 补充文档 → PM 澄清 → 规划 → Backend 执行 → QA 验证 |
| 这个需求记下来 / 记个点子 | PM 写入 Backlog 便签区（或引导写入 `需求输入.md`） |
| 看进度 / 干到哪了 | PM 汇报当前 Sprint 状态 |
| 润色这个大纲 | PM 润色补充 → 依赖分析 + 路线图 → 填充架构设计文档 |
| 部署一下 / 上线 | DevOps 执行部署流程 |
| 代码有问题 / 修个 bug | Backend 定位修复 → QA 验证 |
| 写个测试 | QA 编写测试策略和用例 |
| /plan | 当前 Task 进入 Plan 模式，输出技术方案文档 |
| /spec | 当前上下文进入 Spec 模式，沉淀规范到 `.trellis/spec/` |

---

## 十、template 包与新项目初始化

`CP-开发流程/template/` 是仓库内标准文件包；本技能本节为完整说明。

### 10.1 文件清单

> 五个核心文件的定位、写入方、修改规则见第六节「五个核心文件」表，此处不再重复。

### 10.2 复制命令与占位符

```bash
cp -r CP-开发流程/template/ CP-开发流程/{项目名称}/
```

复制后将各文件中的 `{项目名称}`、`{YYYY-MM-DD}` 等占位符替换为实际内容，并更新文档标题与边界示意图。

### 10.3 初始化后的协作步骤

1. 在 `CP-开发流程/` 下已得到以业务名命名的项目目录
2. 向老板索取项目大纲（复杂需求引导写入 `需求输入.md`）
3. [PM] 润色大纲 → 填充 `开发计划.md`
4. [PM] 依赖分析 + 里程碑 → 写入 `Backlog.md` 路线图区
5. [PM] 拆解 Epic → Story → 写入 `Backlog.md`
6. [PM] 规划 Sprint S1 → 写入 `任务清单.md`
7. [Backend] 开始执行，进入闭环 Sprint

### 10.4 文档化流程

```
老板输入（需求输入.md / Backlog 便签 / 补充文档）
  → PM 梳理 → 开发计划.md → 拆解到 Backlog.md
  → 按 Sprint 写入 任务清单.md → 执行 → 回填 变更记录.md
```

### 10.5 关联规范

> 规范层级、工具链速查统一见第十二节。

---

## 十一、示例模板

以下为 5 个核心文件的正文结构参考，与 `CP-开发流程/template/*.md` 一一对应。

### 11.1 架构设计文档（开发计划.md）

精简的架构文档，只记录慢变内容：

- **变更历史** / **目标定义**（含非目标） / **系统定位与边界**
- **核心模块设计** / **接口设计**（gRPC + API 表） / **数据模型**
- **配置项** / **执行模型** / **异常处理** / **目录结构**
- **设计原则** / **版本存档说明**（重大变更时归档为 `开发计划-v{N}.md`）

> 路线图/里程碑规划信息不在本文档，统一在 `Backlog.md` 的路线图区域管理。

### 11.2 Backlog.md（产品待办列表）

结构分为四大区域：

- **需求输入区**：老板便签（快捷输入）+ 补充文档引用
- **路线图区**（新增）：依赖分析表（地基/核心业务/锦上添花三层）+ 里程碑规划表（MVP/完整版/增强版）
- **产品待办表**：编号/优先级/需求描述/来源/状态/Sprint/备注，优先级用 MoSCoW 方法
- **已完成归档**：已完成条目归档表

### 11.3 需求输入.md

- 老板自由书写区（标准 Markdown，可包含接口定义、代码片段等）
- PM 处理记录区（处理日期/梳理结果/对应 Backlog 编号）

### 11.4 任务清单.md

- 按 Sprint 分组的任务表：编号/任务/状态/说明
- 状态标记：⬜ 未开始 / 🚧 进行中 / ✅ 完成 / ❌ 阻塞
- 任务编号格式：`S{N}-{序号}`
- 归档规则：已完成 Sprint >= 4 个时，最早的移至 `任务清单-历史归档.md`

### 11.5 变更记录.md

- 按 Sprint 分组，倒序排列（最新在前）
- 每条记录：变更概述 / 变更文件 / 影响范围 / 验证结果
- 只追加不修改

---

## 十二、开发规范与工具速查

### 12.1 需求与协作顺序

- **PM 需求读取顺序**：`需求输入.md` → Backlog 便签区 → 补充文档区。
- **协作原则**：先读后做 → 边做边记 → 做完回填 → 编译验证 → 及时归档 → 主动汇报。详见第八节及 Sprint 流程图退出清单。

### 12.2 普通编码模式（Execute 子流程）

> 详见 `.trellis/spec/workflow.md` 的「编码三段式」和角色核心行为中 Backend 的 bash 命令。

编码前读 spec → 编码中遵循规范 → 编码后 `go build` + 质量检查。

### 12.3 工具链与规范层级

| 能力 | 位置 / 使用方式 |
| --- | --- |
| 敏捷流程 | 本技能（`agile-dev-manager`） |
| 角色详细参考 | `ai-team` 技能（核心行为已内联到本技能，无需激活） |
| 技能路由 | `.trae/rules/rule.md` |
| 编码标准 | `.trellis/spec/`（`/start` 时注入） |
| Trellis 命令 | 见第三节「Trellis 命令集成对照表」 |
| 任务与会话 | `.trellis/tasks/`、`.trellis/workspace/`、`.trellis/.current-task` |
| go-zero 框架 | `zero-skills` 技能 |
| 部署 | `module-deploy` 技能（按需） |

### 12.4 规范变更同步

修改 `template/` 或本技能时，同步更新：`template/` → 各项目目录 → 本技能 → `rule.md` → `.trellis/spec/`
