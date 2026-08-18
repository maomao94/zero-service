# Superpowers 技能全量介绍

> Superpowers 是 [obra](https://github.com/obra/superpowers) 出品的一套开源 AI 编程技能集（skill 集合），随 OpenCode 插件安装。它把经过实战验证的软件工程方法论编码成可执行的技能，让 AI 在开发时遵循纪律化的流程：先澄清需求、再写计划、测试先行、系统化调试、代码互审、验证后交付。
>
> 本仓库通过插件 `superpowers@git+https://github.com/obra/superpowers.git` 安装。本文档覆盖全量 14 个技能。

## 设计哲学

- **强制触发**：任何任务先检查是否有技能适用，命中就必须用，不允许跳过（"1% 可能适用就必须调用"）。
- **纪律优先**：多个技能内置"铁律"（Iron Law），并配有"常见合理化借口"（Rationalization Table）和"红旗清单"（Red Flags）来对抗偷懒。
- **计划驱动**：从 brainstorming 到 writing-plans，再到 executing-plans / subagent-driven-development，形成完整流水线。
- **验证证据**：声称完成前必须运行验证命令并确认输出，证据先于断言。
- **优先级**：项目技能 > 个人技能 > superpowers 技能。

## 技能全景

| 技能 | 触发时机 | 核心铁律 |
| --- | --- | --- |
| [brainstorming](#brainstorming) | 任何创造性工作之前 | 未经人类批准不得进入实现 |
| [writing-plans](#writing-plans) | 有 spec/需求，动代码之前 | 计划不能有占位符 |
| [executing-plans](#executing-plans) | 执行已写好的实现计划 | 卡住就停，不要猜 |
| [subagent-driven-development](#subagent-driven-development) | 用子代理执行计划任务 | 每任务一子代理 + 双阶段评审 |
| [dispatching-parallel-agents](#dispatching-parallel-agents) | 2+ 个独立任务可并行 | 一个代理负责一个独立问题域 |
| [test-driven-development](#test-driven-development) | 实现任何功能/bugfix 之前 | 没有失败测试就没有生产代码 |
| [systematic-debugging](#systematic-debugging) | 遇到任何 bug/异常行为 | 没有根因调查就没有修复 |
| [requesting-code-review](#requesting-code-review) | 完成任务/合并前 | 及早评审、频繁评审 |
| [receiving-code-review](#receiving-code-review) | 收到评审反馈时 | 先验证再实现，不盲从 |
| [finishing-a-development-branch](#finishing-a-development-branch) | 实现完成、测试通过后 | 验证测试 → 检测环境 → 呈现选项 → 执行选择 → 清理 |
| [verification-before-completion](#verification-before-completion) | 声称完成/修复/通过之前 | 没有新鲜的验证证据就没有完成声明 |
| [using-git-worktrees](#using-git-worktrees) | 需要隔离工作区时 | 先检测现有隔离，再用原生工具，最后 git 回退 |
| [writing-skills](#writing-skills) | 创建/编辑/验证技能时 | 没有失败测试就没有技能 |
| [using-superpowers](#using-superpowers) | 每次对话开始时 | 响应前先调用适用的技能 |

---

## brainstorming

**触发时机**：创建功能、构建组件、增加功能、修改行为等任何创造性工作之前。**必须使用**。

**三条路径**（先分类，再走对应流程）：

| 路径 | 适用 | 产物 |
| --- | --- | --- |
| Spike（探针） | "能不能做到"这类可行性问题 | 一个答案/建议，代码标为一次性 |
| Bounded（有界） | 对仓库已有代码的小改动 | 聊天内的简短设计，无 spec 文件 |
| Architectural（架构级） | 新项目、新子系统、重构组件关系 | 完整设计文档 → spec → 交给 writing-plans |

**硬性门禁（HARD-GATE）**：无论走哪条路径，在明确告诉人类意图并获得批准之前，不得调用任何实现技能、写任何代码。批准的是意图本身，与任务复杂度无关——简单任务只是设计更短，批准环节永不省略。

**核心流程（架构级）**：
1. 探索项目上下文（文件、文档、近期提交）
2. 逐个提出澄清问题（一次一个）
3. 提出 2-3 个方案及权衡，给出推荐
4. 分节展示设计，每节获得确认
5. 写入 `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md` 并提交
6. spec 自审（占位符/自洽/范围/歧义）
7. 请用户评审 spec 文件
8. 通过后调用 writing-plans 技能

**常见红牌**："太简单不用设计"（简单=短设计≠无设计）、"我已经理解这类应用所以算有界"（有界看仓库不看熟悉度）、"用 spike 试试"（spike 输出是答案不是代码）。

**注意**：架构级路径唯一允许的下游技能是 writing-plans，不得直接跳进实现技能。

---

## writing-plans

**触发时机**：已有 spec 或需求，准备动代码之前。

把 spec 转成"零上下文也能执行"的逐任务实现计划，保存到 `docs/superpowers/plans/YYYY-MM-DD-<feature-name>.md`。

**关键要求**：

- **咬碎粒度**：每个 step 是单个动作（2-5 分钟），如"写失败测试 → 运行确认失败 → 写最小实现 → 运行确认通过 → 提交"。
- **任务右规格**：任务是最小的、自带测试循环、值得一次全新评审的单元。设置/配置/脚手架并入受益任务，只有评审者可能"批准一个而拒绝相邻"时才拆分。
- **无占位符**：禁止 "TBD"、"TODO"、"添加适当的错误处理"、"为上述写测试"（无实际代码）、"类似任务 N"（要重复代码）。代码步骤必须带真实代码块。
- **计划头部**：固定格式，含 Goal / Architecture / Tech Stack / Spec 路径 / Global Constraints（全局约束逐字复制自 spec）。
- **任务结构**：每个任务含 Files（建/改/测）、Interfaces（Consumes/Produces 精确签名）、带 checkbox 的步骤。
- **自审**：spec 覆盖度、占位符扫描、跨任务类型/签名一致性。
- **执行交接**：保存后给出两个选项——子代理驱动（推荐，每任务新子代理）或内联执行。

---

## executing-plans

**触发时机**：拿到已写好的实现计划，在独立会话中执行。

**流程**：
1. 加载计划并批判性评审——有疑问先向人类提出，无疑问则建 todo 逐条执行
2. 逐任务执行：标记 in_progress → 严格按步骤 → 运行验证 → 标记 completed
3. 全部完成后调用 finishing-a-development-branch 收尾

**何时停止求助**：遇到阻塞（依赖缺失、测试失败、指令不清）、计划有致命缺口、不理解指令、验证反复失败。**不要硬闯，停下问。**

**何时回到评审**：人类更新了计划、需要重新思考根本方案。

**注意**：如果平台支持子代理，优先用 subagent-driven-development 而非本技能；绝不在 main/master 分支上未经明确同意就开始实现。

---

## subagent-driven-development

**触发时机**：在当前会话中执行"任务间相互独立"的实现计划。每任务派一个全新子代理实现，之后做任务评审（spec 合规 + 代码质量），最后做全分支评审。

**核心原则**：每任务新子代理 + 任务评审 + 终局评审 = 高质量、快速迭代。子代理永远不继承你的会话上下文，你精确构造它需要的一切。

**何时用**（决策流）：有计划 → 任务大体独立 → 留在本会话 → 用本技能；若跨会话则用 executing-plans。

**流程要点**：
- **Setup**：先用 using-git-worktrees 隔离工作区；用台账（ledger）文件跟踪进度（会话压缩后靠台账 + `git log` 恢复，不信记忆）。
- **任务循环**：记录 BASE 提交 → 用 `task-brief` 生成任务简报 → 派发实现子代理 → 处理报告（DONE/DONE_WITH_CONCERNS/NEEDS_CONTEXT/BLOCKED 四种状态）→ `review-package` 生成评审包 → 派发任务评审。
- **修复循环**：每任务最多 5 轮。第 1-3 轮恢复原实现子代理；第 4-5 轮换更强的模型派新代理。每轮 = 一次修复派发 + 一次限定范围的复审。5 轮后仍不干净则你自己裁决（只有在"每条路都是猜"时才停止）。
- **批次合并**：同形状的小改动合并成一个派发，不必每任务一个代理。
- **模型选择**：用能胜任角色的最廉价模型；最终全分支评审用最强可用模型；转数成本常高于 token 价格。
- **终局评审**：全分支评审用最强调度模型，发现的问题一次派发一个修复代理（不是每个 finding 一个），然后一次限定复审。
- **rulings**：冲突、歧义、计划缺陷由你裁决并记录到台账（`Ruling: <决定> — <为什么> — <错了的代价>`），不停摆。只有四类情况可停下问人：不可逆/破坏性操作、安全敏感动作、工作区外的副作用（合并/推送/发布）、计划坏到每条路都是猜。
- **禁止**：实现子代理派生自己的评审子代理（重复座位）；多个实现子代理并行（冲突）。

---

## dispatching-parallel-agents

**触发时机**：面对 2+ 个独立任务，可无共享状态、无顺序依赖地并行工作。

**核心原则**：每个独立问题域派一个代理，让它们并发工作。串行逐个调查浪费大量时间。

**决策流**：多个失败？→ 相互独立？→ 能并行吗？→ 是则并行派发（同一条消息里的多次派发 = 并行执行；每条消息一个 = 串行）。

**何时不用**：失败相互关联（修一个可能连带修好其他）、需要理解完整系统状态、代理会互相干扰（改同一文件/共享资源）。

**代理提示词结构**：聚焦（一个问题域）、自包含（所有上下文）、明确输出契约（返回什么）。

**常见错误**：范围太宽（"修所有测试"）、无上下文（"修竞态"）、无约束（代理可能重构一切）、输出含糊（"修好它"）。

---

## test-driven-development

**触发时机**：实现任何功能、bug 修复、重构、行为变更之前。

**铁律**：

```
NO PRODUCTION CODE WITHOUT A FAILING TEST FIRST
没有失败测试就没有生产代码
```

先写代码？删掉重来。不允许"留作参考"、"边写测试边改"、"看一眼"。

**红-绿-重构循环**：
1. **RED**：写一个最小失败测试（一个行为、清晰命名、用真实代码，避免 mock）
2. **验证 RED**（强制）：运行测试，确认"失败"且失败原因符合预期（功能缺失而非笔误）
3. **GREEN**：写恰好能通过的最小代码，不加多余功能
4. **验证 GREEN**（强制）：测试通过、其他测试不回归、输出干净
5. **REFACTOR**：去重、改好名字、提取辅助，保持绿灯，不加行为
6. 重复

**写测试前的自问**：说出"哪个生产改动会让这个测试失败"——如果答不上来，测试没写到点上。

**异常情况**（需问人类）：一次性原型、生成代码、配置文件。其余任何"就这一次跳过 TDD"都是偷懒的合理化。

**常见合理化**："太简单不用测"、"之后补测"（补测直接通过证明不了什么）、"手动测过了"（无记录不可复现）、"删掉浪费 X 小时"（沉没成本谬误）、"留作参考"、"TDD 会拖慢我"。

**铁律底线**：生产代码 → 必须先有且亲眼目睹其失败的测试；否则不叫 TDD。

---

## systematic-debugging

**触发时机**：遇到任何 bug、测试失败、意外行为、性能问题、构建失败、集成问题，提出修复之前。

**铁律**：

```
NO FIXES WITHOUT ROOT CAUSE INVESTIGATION FIRST
没有根因调查就没有修复
```

**四阶段**（必须逐阶段完成）：

| 阶段 | 关键活动 | 成功标准 |
| --- | --- | --- |
| 1. 根因调查 | 细读错误信息、稳定复现、检查近期改动、多组件系统逐层加诊断日志、回溯数据流 | 理解 WHAT 和 WHY |
| 2. 模式分析 | 找同代码库中可用的相似实现、完整读参考实现、列出每个差异、理解依赖 | 找出差异 |
| 3. 假设与测试 | 单一假设、最小改动验证、一次一个变量 | 假设确认或换新 |
| 4. 实现修复 | 先建失败测试、单一修复、验证、提交 | bug 解决、测试通过 |

**关键规则**：
- 多组件系统（CI→构建→签名、API→服务→数据库）先加诊断探针，一次运行定位在哪一层断。
- 深层调用栈错误用反向追踪（root-cause-tracing），从坏值溯源到源头，修源头不修症状。
- **3+ 次修复失败 = 架构问题**：停下来质疑基础模式是否根本有问题，与人类讨论，不要试第 4 个修复。
- 修复必须有失败测试先证明（用 TDD 技能），验证用 verification-before-completion 技能。

**红旗**："快速修一下以后再说"、"先试改 X 看看"、"一次加多个改动跑测试"、"跳过测试手动验证"、"大概是 X 让我修"、"我其实不完全懂但这样也许行"。

**"找不到根因"的情况**：确实环境性/时序性/外部性的问题占 5%，其余 95% 都是调查不彻底。

**配套技巧**（同目录文档）：`root-cause-tracing.md` 反向溯源、`defense-in-depth.md` 根因后多层防御、`condition-based-waiting.md` 用条件轮询替代硬编码超时。

---

## requesting-code-review

**触发时机**：完成任务时、实现主要功能后、合并到主干之前（强制）；卡壳求新视角、重构前基线检查、修完复杂 bug 后（可选但有价值）。

**核心原则**：及早评审、频繁评审。

**流程**：
1. 取 git SHA：`BASE_SHA`（如 `HEAD~1` 或 `origin/main`）与 `HEAD_SHA`
2. 派发 `general-purpose` 子代理，填充 `code-reviewer.md` 模板（DESCRIPTION / PLAN_OR_REQUIREMENTS / BASE_SHA / HEAD_SHA）
3. 处理反馈：Critical 立即修、Important 继续前修、Minor 记下稍后处理、评审员错了就带理由反驳

**反馈分级**：Critical（阻断）→ Important（继续前必修）→ Minor（记录延迟）。

**常见合理化**："我自己看 diff 就行"（内联评审烧掉你协调所需的上下文，diff 应该交给评审子代理）、"评审员需要我的全部会话历史"（给它精确构造的上下文，而非你的思考过程）。

---

## receiving-code-review

**触发时机**：收到评审反馈，尤其反馈不清或技术上可疑时，在实施建议之前。

**核心原则**：验证先于实现，询问先于假设，技术正确性先于社交舒适。

**响应模式**：阅读（不反应）→ 理解（用自己的话复述或提问）→ 验证（对照代码库现实）→ 评估（对本代码库技术上是否成立）→ 回应（技术性确认或带理由反驳）→ 实施（一次一项，逐项测试）。

**禁止回应**："你说得完全对！"、"好点子！"、"我现在就实现"（验证之前）。

**对不明确的反馈**：停下来，先澄清所有不明确项再动手。部分理解 = 错误实现。

**何时反驳**：建议破坏现有功能、评审员缺乏完整上下文、违反 YAGNI（未使用功能）、技术上对当前技术栈不正确、遗留/兼容原因、与人类架构决策冲突。

**正确反馈的回应**：直接说明修了什么（"已修复，[位置] 处 [具体改动]"），不要用"谢谢""好主意"等客套——行动说话，代码本身证明你听到了。

---

## finishing-a-development-branch

**触发时机**：实现完成、所有测试通过，需要决定如何集成工作时。

**流程**：
1. **验证测试**：跑全量测试。失败则报告并停止，绿灯后才走菜单。
2. **检测环境**：判断在普通仓库 / 具名分支 worktree / 分离 HEAD worktree，决定菜单与清理方式。
3. **确定基线分支**：确认 fork 来源，合并前确认（合并错基线代价高）。
4. **呈现选项**（原样呈现，不增删）：

   普通仓库 / 具名分支 worktree：
   ```
   1. Merge back to <base-branch> locally
   2. Push and create a Pull Request
   3. Keep the branch as-is (I'll handle it later)
   ```
   分离 HEAD（无合并选项）：
   ```
   1. Push as new branch and create a Pull Request
   2. Keep as-is (I'll handle it later)
   ```

   丢弃工作只在人类明确要求时执行，且必须确认输入 `discard`。

5. **执行选择**：
   - 合并：切回基线分支 → pull → merge → **在合并结果上再跑一次测试** → 绿则清理 worktree + `git branch -d`；红则停在原地调查。
   - 推 PR：推送并创建 PR，保留 worktree 以便在 PR 反馈上迭代。
   - 保持原样：报告分支和 worktree 保留位置。
6. **清理 worktree**：只清理 `.worktrees/` 或 `worktrees/` 下的（superpowers 创建的可清理，宿主环境的留给宿主）。移除被拒（有未提交文件）时**绝不擅自 `--force`**，展示文件清单让人选择提交/移动/删除。

---

## verification-before-completion

**触发时机**：声称任何工作完成、修复、通过，或提交/建 PR 之前。**证据先于断言，永远。**

**铁律**：

```
NO COMPLETION CLAIMS WITHOUT FRESH VERIFICATION EVIDENCE
没有新鲜的验证证据就没有完成声明
```

如果在当前消息中没运行过验证命令，就不能声称它通过。

**门函数**：识别（哪个命令能证明这个断言）→ 运行（完整命令、新鲜、完整）→ 读取（完整输出、检查退出码、统计失败数）→ 验证（输出是否确认断言）→ 只有此时才下结论。

**典型失败**：

| 声称 | 需要 | 不够 |
| --- | --- | --- |
| 测试通过 | 测试命令输出 0 失败 | 之前的运行、"应该通过" |
| 静态检查干净 | 检查输出 0 错误 | 部分检查、外推 |
| 构建成功 | 构建命令退出码 0 | 静态检查通过、日志正常 |
| bug 已修复 | 原始症状测试通过 | 代码改了、假设修好了 |
| 回归测试有效 | 红-绿循环已验证 | 测试通过一次 |
| 代理已完成 | VCS diff 显示改动 | 代理报告"成功" |
| 需求满足 | 逐行 checklist | 测试通过 |

**红旗**：用"应该/大概/似乎"、验证前表达满意（"好！""完美！""搞定！"）、未经验证就提交/推送/建 PR、相信代理的成功报告、部分验证、"就这一次"。

---

## using-git-worktrees

**触发时机**：开始需要隔离的功能开发时，或在执行实现计划之前。确保工作在隔离工作区发生。

**核心原则**：先检测现有隔离，再用原生工具，然后 git 回退，绝不跟 harness 对着干。

**Step 0 检测现有隔离**：比较 `GIT_DIR` 与 `GIT_COMMON`。不等（且非子模块，用 `--show-superproject-working-tree` 验证）→ 已在 worktree，跳过创建。相等 → 普通仓库，需要征求是否创建 worktree（除非指令已有声明偏好）。

**Step 1 创建工作区**：
- **原生 worktree 工具优先**（如 `EnterWorktree`/`WorktreeCreate`/`/worktree` 命令/`--worktree` 标志）。有就不用 `git worktree add`（会产生 harness 看不见的幻影状态）。
- **git 回退**（无原生工具时）：目录优先级为 指令声明偏好 > 现有 `.worktrees/`（优）或 `worktrees/` > 默认 `.worktrees/`。**必须验证目录被 gitignore**（`git check-ignore`），未忽略则先加进 .gitignore 并提交——防止 worktree 内容被误提交进仓库。命令：`git worktree add <path> -b <branch>`。

**Step 2 项目设置**：按项目类型自动安装依赖（package.json → npm install；go.mod → go mod download；等）。

**Step 3 验证干净基线**：跑测试确保工作区起步干净。失败则报告并询问，通过则报告。

**常见合理化**："我显然不在 worktree，不用查"（检测命令才能定论）、"`git worktree add` 更快"（绕过原生工具是 #1 错误）、"worktree 目录肯定已被忽略"（必须跑 `git check-ignore`）、"工作区是新的，基线测试可以等等"（脏基线让后续失败无法定位）。

---

## writing-skills

**触发时机**：创建新技能、编辑既有技能、部署前验证技能。

**本质**：技能写作就是"把 TDD 应用于过程文档"。用子代理做压力场景测试，观察失败（基线行为），写技能，观察通过（代理合规），重构（堵漏洞）。

**铁律**：

```
NO SKILL WITHOUT A FAILING TEST FIRST
没有失败测试就没有技能
```

新技能和编辑既有技能都适用。先写技能后测试？删掉重来。

**TDD 映射**：测试用例→压力场景；生产代码→SKILL.md；RED→无技能时代理违规（记录逐字合理化借口）；GREEN→有技能时代理合规；REFACTOR→保持合规前提下堵漏洞。

**何时创建技能**：技巧不是一眼就显然、跨项目可复用、模式广泛适用、他人受益。不创建：一次性方案、别处已有标准实践、项目特定约定（放 instructions 文件）、机械约束（能用正则/校验强制就自动化）。

**技能结构**：
- 目录：`skills/skill-name/SKILL.md`（必需）+ 支撑文件（仅在需要时）。
- frontmatter：`name`（字母数字连字符）+ `description`（第三人称，只写"何时用"不写"做什么"）。
- 正文：Overview → When to Use → Core Pattern → Quick Reference → Implementation → Common Mistakes。

**发现性优化（SDO）**：description 只描述触发条件，**绝不总结技能的工作流**（代理会照 description 做而跳过正文）。用搜索关键词、主动语态命名、控制 token（高频技能 <200 词）。

**防合理化**：明确列出每个漏洞（"No exceptions"）、加"违反字面即违反精神"原则、建立合理化表（Excuse/Reality）、建立红旗清单、把违规症状写进 description。

**形式匹配失败类型**：违规 → 禁止 + 合理化表 + 红旗；输出形状错 → 正面配方/契约（说输出是什么）；漏必需元素 → 结构化必需槽位；行为取决于条件 → 条件化（基于可观测谓词）。

**部署门禁**：每个技能必须完成测试循环才能部署；不批量生产技能；`render-graphs.js` 可把技能流程图渲染成 SVG。

---

## using-superpowers

**触发时机**：每次对话开始时。定义如何发现和使用技能，要求在任何响应（包括澄清问题）之前先调用技能。

**核心规则**：

> 如果认为有 1% 的可能某个技能适用于当前任务，就必须调用它。这不是可协商的。

**流程**：先调用相关技能（哪怕事后发现不适用）→ 宣布 "Using [skill] to [purpose]" 并严格遵循 → 有 checklist 就为每项建 todo。

**技能优先级**：流程技能先行（决定方法），实现技能随后（执行）。如："我们来做 X" → 先 brainstorming；"修复这个 bug" → 先 systematic-debugging。

**平台适配**：如果 harness 出现在参考列表中，先读其平台参考文件（Codex/Pi/Antigravity/Hermes Agent）。

**用户指令优先**：人类指令（CLAUDE.md、AGENTS.md、直接请求）优先于技能，技能又优先于默认行为。

**红旗（停止）**："这只是个简单问题"、"我需要更多上下文先"、"让我先探索代码库"、"我可以快速查 git/文件"、"这个不算任务"、"我记得这个技能"、"这不需要正式技能"、"这技能小题大做"。

---

## 附：OpenCode 工具映射

技能里描述的抽象动作在 OpenCode 上对应：

| 技能里的说法 | OpenCode 工具 |
| --- | --- |
| "create a todo" / "mark complete" | `todowrite` |
| `Subagent (general-purpose):` | `task` 工具，`subagent_type: "general"`（探索用 `"explore"`） |
| "Invoke a skill" | 原生 `skill` 工具 |
| "Read a file" | `read` |
| "Create/edit/delete a file" | `write` / `edit` |
| "Run a shell command" | `bash` |
| "Search file contents" / "find files" | `grep` / `glob` |
| "Fetch a URL" | `webfetch` |

## 更新与排查

- **更新**：插件经 git-backed spec 安装，个别 OpenCode/Bun 版本会锁定解析后的 git 依赖，重启可能拿不到新提交。清缓存或重装可解决；可固定版本：`"plugin": ["superpowers@git+https://github.com/obra/superpowers.git#v5.0.3"]`。
- **排查**：日志检查 `opencode run --print-logs "hello" 2>&1 | grep -i superpowers`；`skill` 工具列出已发现技能；`OPENCODE_PURE=1` 跳过外部插件。
- **文档**：https://github.com/obra/superpowers 与 https://opencode.ai/docs/