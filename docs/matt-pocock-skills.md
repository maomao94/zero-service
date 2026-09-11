# Matt Pocock 技能全量介绍

> Matt Pocock 技能集是 [mattpocock/skills](https://github.com/mattpocock/skills) 出品的一套开源 AI 编程技能集，通过 OpenCode 技能机制安装。它聚焦于代码库设计、调试诊断、需求拆分、写作和仓库管理等工程实践，帮助 AI 在开发时遵循结构化的流程。
>
> 本仓库通过符号链接安装自 `~/.skills/vendor/mattpocock-skills`。本文档覆盖全量 37 个技能。
>
> **版本信息**：`v1.2.3-39-g6654f6b`（commit `6654f6b`，分析于 2026-09-04；Matt 会持续更新，更新后请同步本文档）

## 设计哲学

- **按需触发**：每个技能定义明确的触发条件，命中时自动调用。
- **两轴评审**：code-review 技能沿"标准合规"和"规格忠实"两个独立轴线审查。
- **无情质询**：grilling 技能通过多轮追问暴露假设和盲点，直到达成共识。
- **深度模块**：codebase-design 技能追求模块接口的深度，拒绝浅层抽象。
- **垂直切片**：to-tickets 技能要求工单是端到端可交付的完整路径。

## 技能全景

### 工程类 (Engineering)

| 技能 | 触发时机 | 核心概念 |
| --- | --- | --- |
| [ask-matt](#ask-matt) | 不确定该用哪个技能时 | 路由器：idea → ship 主流程 |
| [code-review](#code-review) | 审查分支/PR/变更时 | 两轴独立审查（标准 + 规格） |
| [codebase-design](#codebase-design) | 设计或改进模块接口时 | 深度、接缝、适配器、杠杆点 |
| [diagnosing-bugs](#diagnosing-bugs) | 报告某物损坏/报错/慢时 | 6 阶段诊断循环 |
| [domain-modeling](#domain-modeling) | 讨论文档术语/记录 ADR 时 | 挑战模糊术语，交叉验证代码 |
| [grill-with-docs](#grill-with-docs) | 在工作目录中工作时 | 调用 grilling + domain-modeling |
| [implement](#implement) | 有规格后开始实现时 | TDD + 类型检查 + 测试 |
| [improve-codebase-architecture](#improve-codebase-architecture) | 有空闲维护代码库健康时 | 扫描浅模块 → HTML 报告 |
| [prototype](#prototype) | 验证设计问题时 | 一次性原型，无持久化 |
| [research](#research) | 需要调研主题时 | 后台代理 → 引用文档 |
| [resolving-merge-conflicts](#resolving-merge-conflicts) | 在 git 冲突中时 | 查找一手来源 → 按意图解决 |
| [setup-matt-pocock-skills](#setup-matt-pocock-skills) | 首次使用工程技能前 | 设置 issue tracker、标签、文档布局 |
| [tdd](#tdd) | 想要先写测试再写代码时 | 红→绿循环，只在接缝处测试 |
| [to-spec](#to-spec) | 讨论充分后需要正式化为规格时 | 综合已有讨论，不包含代码路径 |
| [to-tickets](#to-tickets) | 需要将大工作拆分为可执行任务时 | 垂直切片，每个工单声明阻塞边 |
| [triage](#triage) | bug 和请求堆积时 | 5 状态分类状态机 |
| [wayfinder](#wayfinder) | 模糊的大想法，超出单会话容量时 | 地图 + 决策工单，战争迷雾 |
| [wizard](#wizard) | 配置基础设施/设置凭证时 | 交互式 bash 向导脚本 |

### 生产力类 (Productivity)

| 技能 | 触发时机 | 核心概念 |
| --- | --- | --- |
| [grill-me](#grill-me) | 在非工作目录环境中打磨计划时 | 调用 grilling 原语 |
| [grilling](#grilling) | 想要压力测试想法时 | 设计树 → 轮次 → 前沿 → 推荐 |
| [handoff](#handoff) | 需要将工作转交给另一个代理时 | 写入交接文档到临时目录 |
| [teach](#teach) | 用户想学习某个主题时 | 知识→技能→智慧三层教学 |
| [to-questionnaire](#to-questionnaire) | 阻塞信息在别人脑中时 | 审问发送方，针对知识缺口 |
| [wait-what](#wait-what) | 对话中没理解时 | ASD-STE100 简化技术英语重述 |
| [writing-for-agents](#writing-for-agents) | 创建/编辑技能或 AGENTS.md 时 | 上下文指针、信息层次、引导词 |

### 杂项类 (Misc)

| 技能 | 触发时机 | 核心概念 |
| --- | --- | --- |
| [git-guardrails-claude-code](#git-guardrails-claude-code) | 想要防止破坏性 git 操作时 | PreToolUse hook 拦截危险命令 |
| [migrate-to-shoehorn](#migrate-to-shoehorn) | 替换测试中的 `as` 类型断言时 | `fromPartial()` / `fromAny()` |
| [scaffold-exercises](#scaffold-exercises) | 搭建课程练习目录结构时 | problem/solution/explainer 子目录 |
| [setup-pre-commit](#setup-pre-commit) | 添加提交前钩子时 | Husky + lint-staged + Prettier |

### 进行中类 (In-progress)

| 技能 | 触发时机 | 核心概念 |
| --- | --- | --- |
| [claude-handoff](#claude-handoff) | 需要无缝接力工作时 | 后台代理摘要交接 |
| [implement-spec](#implement-spec) | 有规格和工单后实现时 | 任务图 + 并行子代理 |
| [loop-me](#loop-me) | 设计可重复的工作流时 | 循环 → 工作流规格 |
| [retro](#retro) | 想要改进代理环境时 | 7 个改进候选类别 |
| [setup-ts-deep-modules](#setup-ts-deep-modules) | 让每个包都成为深度模块时 | dependency-cruiser 4 条规则 |
| [writing-beats](#writing-beats) | 有原始素材后逐步构建文章时 | 节拍 → 接地 → 选择你的冒险 |
| [writing-fragments](#writing-fragments) | 想要拓宽可写内容空间时 | 挖掘原始片段，静默追加 |
| [writing-shape](#writing-shape) | 承诺结构并填充文章时 | 逐段增长 → 接地系统 |

---

## 工程类技能详解

### ask-matt

**触发时机**：不确定该用哪个技能时。

**核心概念**：路由器技能，定义了主流程（idea → ship）、入口流程和独立流程。

**关键规则**：
- 1-3 步应在一个无中断上下文窗口中完成
- 遵循 smart zone 限制

---

### code-review

**触发时机**：审查分支、PR、工作中的变更。

**核心概念**：两个并行子代理分别审查 Standards（编码规范）和 Spec（规格忠实度），结果并列展示。

**关键规则**：
- 两轴独立报告，不合并排序
- 包含 Fowler 代码异味基线

---

### codebase-design

**触发时机**：设计或改进模块接口、寻找深化机会、决定接缝位置。

**核心概念**：Module、Interface、Depth、Seam、Adapter、Leverage、Locality。

**关键规则**：
- 深度是接口的属性而非实现的属性
- 删除测试验证模块价值

---

### diagnosing-bugs

**触发时机**：报告某物损坏/报错/慢。

**核心概念**：6 阶段流程：构建反馈循环 → 复现最小化 → 假设 → 诊断 → 修复+回归测试 → 清理。

**关键规则**：
- 没有红色能力的命令就不进入 Phase 2
- 假设必须可证伪
- 秘密必须脱敏

---

### domain-modeling

**触发时机**：讨论代码库术语、编写 CONTEXT.md、记录 ADR。

**核心概念**：挑战模糊术语、讨论具体场景、交叉验证代码。

**关键规则**：
- CONTEXT.md 纯粹是术语表，不含实现细节
- ADR 只在满足三条件时创建

---

### grill-with-docs

**触发时机**：在工作目录中工作时。

**核心概念**：调用 grilling 和 domain-modeling 两个技能。

---

### implement

**触发时机**：有规格后开始实现。

**核心概念**：尽可能使用 TDD，定期运行类型检查和测试。

**关键规则**：
- 完成后运行 code-review，然后提交

---

### improve-codebase-architecture

**触发时机**：有空闲时维护代码库健康。

**核心概念**：发现浅模块 → HTML 可视化报告 → 选择 → grilling 迭代。

**关键规则**：
- YAGNI 原则——优先关注最近变更的热点
- 报告存放在系统临时目录

---

### prototype

**触发时机**：验证状态模型是否合理、探索 UI 外观。

**核心概念**：两条分支：逻辑原型（HTML 文件）和 UI 原型（多变体单页）。

**关键规则**：
- 从第一天就是一次性的
- 默认无持久化
- 跳过打磨
- 完成后捕获为原始来源

---

### research

**触发时机**：需要调研主题、收集文档/API 事实。

**核心概念**：后台代理调查 → 写入引用文档 → 存放在仓库约定位置。

**关键规则**：
- 只使用一手来源（官方文档、源代码、规范）

---

### resolving-merge-conflicts

**触发时机**：已经在 git merge/rebase 冲突中时。

**核心概念**：查找一手来源 → 按意图解决每个 hunk → 运行自动检查 → 完成合并。

**关键规则**：
- 永远不 `--abort`
- 尽可能保留双方意图

---

### setup-matt-pocock-skills

**触发时机**：首次使用其他工程技能前运行一次。

**核心概念**：设置 issue tracker、分类标签、文档布局。

**关键规则**：
- 优先编辑已存在的 CLAUDE.md 或 AGENTS.md
- 不会覆盖用户编辑

---

### tdd

**触发时机**：想要先写测试再写代码。

**核心概念**：红 → 绿循环；只在预确认的接缝处测试。

**关键规则**：
- 红先于绿
- 一次一个切片
- 重构不属于循环的一部分

---

### to-spec

**触发时机**：讨论充分后需要正式化为规格。

**核心概念**：无面试，只综合已有讨论；包含问题陈述、方案、用户故事、实现决策。

**关键规则**：
- 不包含具体文件路径或代码片段
- 应用 `ready-for-agent` 标签

---

### to-tickets

**触发时机**：需要将大工作拆分为可执行任务。

**核心概念**：垂直切片（端到端完整路径）；每个工单声明阻塞边。

**关键规则**：
- 宽重构是例外（用 expand-contract 模式）
- 一次只解决一个工单

---

### triage

**触发时机**：bug 和请求堆积时。

**核心概念**：5 个状态角色（needs-triage → needs-info → ready-for-agent / ready-for-human / wontfix）。

**关键规则**：
- AI 生成的评论必须以 "This was generated by AI during triage." 开头

---

### wayfinder

**触发时机**：模糊的大想法，超出单个代理会话容量。

**核心概念**：地图（索引）+ 决策工单；战争迷雾（Not yet specified）；前沿（frontier）。

**关键规则**：
- 默认只规划不执行
- 每个会话最多解决一个工单
- 用名称引用而非 ID

---

### wizard

**触发时机**：配置基础设施、设置凭证/CI secrets、操作第三方仪表板。

**核心概念**：阶段式向导脚本；基于 template.sh 构建。

**关键规则**：
- 不手动编辑库代码
- 默认临时使用
- 静态验证而非运行

---

## 生产力类技能详解

### grill-me

**触发时机**：在非工作目录环境中（无 repo）打磨计划或设计。

**核心概念**：调用 grilling 原语。

---

### grilling

**触发时机**：想要压力测试想法。

**核心概念**：设计树 → 轮次 → 前沿 → 推荐答案。

**关键规则**：
- 找事实是代理的工作，决策是用户的工作
- 前沿为空时结束

---

### handoff

**触发时机**：需要将工作转交给另一个代理。

**核心概念**：写入手动目录（OS 临时目录）；包含建议技能部分。

**关键规则**：
- 不重复已捕获在其他制品中的内容
- 脱敏敏感信息

---

### teach

**触发时机**：用户想学习某个主题。

**核心概念**：知识→技能→智慧三层；教学空间（MISSION.md、lessons、reference 等）。

**关键规则**：
- 区分流利度和存储强度
- 每课应短小精悍
- 使用检索练习和间隔重复

---

### to-questionnaire

**触发时机**：阻塞信息在别人脑中。

**核心概念**：审问发送方（而非主题）；针对知识缺口设计问题。

**关键规则**：
- 最重要的问题优先
- 每个问题一个想法

---

### wait-what

**触发时机**：对话中没理解时。

**核心概念**：用 ASD-STE100 简化技术英语重新表述；使用 CONTEXT.md 通用语言。

---

### writing-for-agents

**触发时机**：创建/编辑技能、AGENTS.md 或 CLAUDE.md。

**核心概念**：上下文指针、信息层次（步骤 vs 参考）、引导词、裁剪。

**关键规则**：
- 每个含义一个真实来源
- 环境是来源之一
- 按行检查相关性

---

## 杂项类技能详解

### git-guardrails-claude-code

**触发时机**：想要防止破坏性 git 操作。

**核心概念**：PreToolUse hook 拦截 git push / reset --hard / clean / branch -D 等。

**关键规则**：
- 可安装为项目级或全局级
- 阻止时返回退出码 2

---

### migrate-to-shoehorn

**触发时机**：替换测试中的 `as` 类型断言。

**核心概念**：`fromPartial()` 替换 `as Type`；`fromAny()` 替换 `as unknown as Type`。

**关键规则**：
- 仅用于测试代码，不用于生产代码

---

### scaffold-exercises

**触发时机**：搭建课程练习目录结构。

**核心概念**：每个练习含 problem/solution/explainer 子目录。

**关键规则**：
- 每个子目录需要非空 readme.md
- 移动用 git mv 保持历史

---

### setup-pre-commit

**触发时机**：添加提交前钩子。

**核心概念**：Husky + lint-staged + Prettier 三件套。

**关键规则**：
- 自动检测包管理器
- Husky v9+ 不需要 shebang

---

## 进行中类技能详解

### claude-handoff

**触发时机**：需要无缝接力工作。

**核心概念**：写入摘要 → 通过 `claude --bg --name` 启动后台代理。

**关键规则**：
- 必须传 `-n` / `--name` 描述性名称

---

### implement-spec

**触发时机**：有规格和工单后在代码中实现。

**核心概念**：工单是任务图（含阻塞关系）；并行实现子代理最大化并发。

**关键规则**：
- 通信尽量稀疏
- 通过上下文指针传递信息

---

### loop-me

**触发时机**：设计可重复的工作流。

**核心概念**：循环（loop） → 工作流（workflow）；触发器、检查点、brief。

**关键规则**：
- 工作流规格完成标准是实现代理无需提问即可构建

---

### retro

**触发时机**：想要改进代理环境。

**核心概念**：7 个改进候选类别：导航、自动检查、编码标准、AGENTS.md、工具经济、空操作、信息访问。

**关键规则**：
- 审查代理负责强制编码标准，而非实现代理

---

### setup-ts-deep-modules

**触发时机**：让每个包都成为深度模块。

**核心概念**：4 条规则：入口点边界、包内自由、测试通过入口点、无循环。

**关键规则**：
- 公共 vs 私有由深度决定
- 包是扁平的
- 不使用 barrel 文件

---

### writing-beats

**触发时机**：有原始素材后，逐步构建文章。

**核心概念**：节拍（beat） → 接地（grounding） → 选择你的冒险式写作。

**关键规则**：
- 每次只追加一个节拍
- 每次写入前从磁盘重读文件

---

### writing-fragments

**触发时机**：想要拓宽可写内容的空间。

**核心概念**：片段是可能存活到最终文章中的任何文本碎片。

**关键规则**：
- 静默追加
- 不编辑原始材料文件
- 最有价值的片段是引导词

---

### writing-shape

**触发时机**：有原始素材后，承诺结构并填充。

**核心概念**：逐段增长 → 接地系统 → 格式论证。

**关键规则**：
- 不编辑原始材料文件
- 每块写入前从磁盘重读文件
- 用户决定何时完成

---

## 本仓库的轻量工作流

本项目直接使用 Matt 技能，不再依赖额外的任务生命周期框架。根目录 `AGENTS.md` 只保存默认行为和知识入口，稳定工程契约与领域不变量由 OpenViking 记忆系统提供（`viking://resources/project/zero-service`）。

### Idea → Ship

1. 在仓库中讨论新需求时用 `grill-with-docs`，把稳定术语与重要决策留在项目文档中。
2. 一次会话可以完成的需求直接用 `implement`。
3. 多会话需求依次用 `to-spec`、`to-tickets`，再为每张无阻塞 ticket 单独调用 `implement`。
4. `implement` 内部按需使用 `tdd`，结束时用 `code-review` 检查标准与需求符合度。

遇到难复现故障用 `diagnosing-bugs`；需要运行代码回答设计问题时用 `prototype`；只有路线本身无法在一次会话中看清的大型工作才用 `wayfinder`。

### 知识归属

| 信息 | 位置或技能 |
| --- | --- |
| 每次任务都需要的少量默认行为 | `AGENTS.md` |
| 稳定工程契约、领域不变量和任务路由 | OpenViking 记忆系统（`viking://resources/project/zero-service`） |
| 领域术语表与重要决策 | `domain-modeling` 管理的 `CONTEXT.md` 与 ADR |
| 功能需求与实施切片 | `to-spec`、`to-tickets` 产生的文档或 issue |
| 一手资料调研 | `research` 产生的带引用 Markdown |

功能 spec 是一次交付的需求来源，Agent 知识库是跨任务复用的项目记忆。只有稳定、非直观且会影响未来正确性的结论进入知识库；完整 API、当前依赖版本和目录清单继续由源码、契约、`go.mod` 与文件系统负责。

### 上下文卫生

需求澄清到 ticket 拆分尽量保持同一上下文。每张 ticket 的实现使用独立上下文，只检索 OpenViking 记忆（`openviking_search`）命中的一份工程规范和一份领域契约；文档明确指出依赖时再展开。

---

## 安装与维护

### 安装位置

| 类型 | 路径 |
| --- | --- |
| 源仓库 | `~/.skills/vendor/mattpocock-skills` |
| OpenCode 技能目录 | `~/.config/opencode/skills/mcp-*` |

### 更新方法

```bash
# 拉取最新变更
cd ~/.skills/vendor/mattpocock-skills && git pull

# 重新链接（符号链接自动指向最新）
# 无需额外操作，符号链接已指向源目录
```

### 版本记录

安装时版本：`v1.2.3-39-g6654f6b`（commit `6654f6b`）

> 更新后请修改本文档顶部的版本信息。
