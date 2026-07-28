# Research: DescribeRRule 第二轮完整审计

- **Query**: 审核本地中文 `DescribeRRule` 是否借鉴当前上游 human-text 决策结构，同时保持 `rrule-go v1.8.2` occurrence 语义，并显式呈现所有已解析、可表达条件
- **Scope**: mixed
- **Date**: 2026-07-28

## Findings

### 审计结论

当前实现已经覆盖并正确表达多数常规场景：七种 Frequency、DTSTART 日期/时钟默认值、INTERVAL 相位提示、常规 BY* 交集、月/年作用域序号星期、工作日/周末/一周七天、时钟笛卡尔积、一般 BYSETPOS、COUNT/UNTIL、TZID/DST 提示以及 RDATE/EXDATE 列表。它采用按频率生成骨架、先做支持门禁、再渲染条件与边界的结构，确实是在学习上游决策组织，而不是翻译英文。

但按 PRD 的“不能输出误导描述”和“显式呈现实际 occurrence 语义”标准，当前版本尚未完全达到开发目的。确认 4 个有可复现行为差异的主要问题，以及 3 个次要边界/覆盖 caveat。

### Prioritized Findings

#### P0 — 合法耗尽规则仍使用无条件“执行”，违反“不承诺 occurrence”要求

**本地证据**：

- `common/crontask/describe.go:62-66` 无条件追加“执行”。
- `common/crontask/describe.go:432-442` 把 `DTSTART` 至 `UNTIL` 一律称为“有效期”，即使结束早于开始。
- `.trellis/tasks/07-28-audit-rrule-description/prd.md:12` 明确要求永久无候选及 `UNTIL < DTSTART` 不得承诺 occurrence。
- `.trellis/tasks/07-28-audit-rrule-description/design.md:34` 又要求不重复实现 occurrence 可达性判断，因此安全文案本身必须避免无条件承诺。

**复现 A（永久无日历候选）**：

```text
DTSTART:20260101T090000Z
RRULE:FREQ=YEARLY;COUNT=1;BYMONTH=2;BYMONTHDAY=30
```

`rrule-go` 不会产生 occurrence；当前描述仍会包含“每年2月，且各指定月份的第 30 天 09:00 执行”。这不是仅列条件，而是肯定发生执行。

**复现 B（边界已倒置）**：

```text
DTSTART:20260727T090000Z
RRULE:FREQ=DAILY;UNTIL=20260726T090000Z
```

实际 occurrence 为空；当前描述包含“每天 09:00 执行，周期规则有效期：2026-07-27 09:00:00 至 2026-07-26 09:00:00”。“有效期”倒序且“每天执行”仍作出承诺。

**期望行为/措辞契约**：仍可完整列出频率和筛选条件，但必须把执行表述限定为“形成候选/符合时执行”或明确“周期规则当前无 occurrence”；不能说一定执行。不要求项目自行证明所有日历交集可达，只要求通用描述不承诺候选必然存在。

#### P0 — 重复时钟值会改变 BYSETPOS 的内部候选序号，但 renderer 去重后描述了另一套候选集

**本地及依赖证据**：

- `common/crontask/describe.go:414-429` 对每个时钟维度调用 `sortedUniqueInts`，先去重再展示。
- `rrule-go@v1.8.2/rrule.go:243-253` 构造低频 `timeset` 时保留重复时钟值。
- `rrule-go@v1.8.2/rrule.go:615-644` 的 BYSETPOS 以 `dayset × timeset` 的实际长度和顺序定位，因而重复值占据不同候选序号；仅在定位完成后对相同 instant 去重。

**复现**：

```text
DTSTART:20260701T000000Z
RRULE:FREQ=MONTHLY;COUNT=2;BYDAY=MO;BYHOUR=9,9;BYMINUTE=0;BYSECOND=0;BYSETPOS=2
```

每个周一的内部时钟集是 `[09:00, 09:00]`。因此第 2 个候选仍是当月第一个周一 09:00；前两个 occurrence 为 `2026-07-06T09:00:00Z`、`2026-08-03T09:00:00Z`。当前 renderer 把小时去重为一个 `09`，却仍写“每月周一 09:00；……选择第 2 个执行”；按它展示的唯一候选序列，第 2 个应被理解为第二个周一，和库实际结果不同。

**期望行为/措辞契约**：对于带 BYSETPOS 的规则，文案展示的候选多重集与排序必须和依赖库实际索引对象一致；如果无法无歧义展示重复候选，应安全拒绝，而不能去重后继续描述位置。

补充：无 BYSETPOS 时，重复时钟也会在 `RRule` 内重复消耗 COUNT，随后被 `Set.Iterator` 去重（`rruleset.go:149-170`）。现有“周期规则最多生成 N 次”是上界措辞，尚不构成同等级的硬错误；但它说明 duplicate clock 不是纯展示冗余。

#### P1 — RDATE 被描述成保证“额外执行”，但 EXDATE 优先且 Set 会去重

**本地及依赖证据**：

- `common/crontask/describe.go:72-73` 固定输出“额外执行”和“排除执行”。
- `rrule-go@v1.8.2/rruleset.go:133-174` 先合并 RRULE/RDATE、去重相同 instant，再由 EXDATE 排除；RDATE 只是 inclusion source，不保证最终执行。
- `.trellis/spec/backend/crontask-guidelines.md:176` 要求 COUNT 只描述周期规则次数，但同一 Set 组合语义同样应由最终 occurrence 决定。

**复现**：

```text
DTSTART:20260727T090000Z
RRULE:FREQ=DAILY;COUNT=1
RDATE:20260727T090000Z,20260730T100000Z
EXDATE:20260727T090000Z,20260730T100000Z
```

最终 Set 为空：第一个 RDATE 与 RRULE 重合且被排除，第二个 RDATE 也被排除。当前描述仍声称两项“额外执行”，随后才列“排除执行”，没有说明排除优先，产生相互冲突的业务结论。

**期望行为/措辞契约**：RDATE 应被表述为额外纳入的日期/候选，并明确 EXDATE 从 RRULE 与 RDATE 的合并结果中排除；不能把 inclusion date 直接称为最终执行。

#### P1 — `SECONDLY + BYSETPOS` 的可表达单候选语义被支持门禁误拒绝

**本地及依赖证据**：

- `common/crontask/describe.go:137-143` 要求 BYSETPOS 同时存在某个“selectable filter”。
- `common/crontask/describe.go:414-429` 对 SECONDLY 且无显式 clock BY* 返回三个空维度。
- `rrule-go@v1.8.2/rrule.go:526-547, 774-803` 对每个 SECONDLY 周期仍建立一个当前秒的单元素 timeset；BYSETPOS=1 或 -1 可正常选择该唯一候选。

**复现**：

```text
DTSTART:20260727T091005Z
RRULE:FREQ=SECONDLY;INTERVAL=2;COUNT=2;BYSETPOS=1
```

依赖库的 occurrence 是 `2026-07-27T09:10:05Z`、`2026-07-27T09:10:07Z`；当前 `DescribeRRule` 返回 `ErrUnsupportedDescription: BYSETPOS without a selectable filter`。该条件虽然冗余，但已经解析、语义稳定且能明确表达，不属于“无法准确表达”的高级组合。

**期望行为/措辞契约**：要么完整呈现“每个 2 秒周期唯一候选中的第 1 个”，要么任务文档必须把该组合明确列为有意不支持；当前 option matrix 没有覆盖该组合，且现有拒绝理由与依赖候选模型不符。

#### P1 — 月作用域过大负序号星期会在 rrule-go 运行时 panic，但描述器给出肯定文案

**依赖证据**：

- `rrule-go@v1.8.2/rrule.go:299-305` 只验证 ordinal 位于 -53..53。
- `rrule-go@v1.8.2/rrule.go:444-477` 计算负 ordinal 时在检查月范围之前访问 `wdaymask[i]`；例如一月 `-6MO` 可形成负下标。
- `common/crontask/describe.go:145-163` 仅拒绝低于 MONTHLY 的 ordinal 和普通/ordinal 淁用，对 MONTHLY 的 -6..-53 不拒绝。
- 第一轮 `.trellis/tasks/07-28-audit-rrule-description/research/scenario-audit.md:23-25` 已记录该依赖风险，本轮确认它仍存在于当前代码边界。

**复现**：

```text
DTSTART:20260101T090000Z
RRULE:FREQ=MONTHLY;COUNT=1;BYDAY=-6MO
```

`DescribeRRule` 可生成“每月倒数第 6 个周一 09:00 执行”，但 occurrence 迭代可能以数组负下标 panic。YEARLY + BYMONTH 的月作用域 ordinal 有同类风险。

**期望行为/措辞契约**：就描述 API 而言，不能为依赖库无法安全迭代的组合输出肯定执行文案；应安全拒绝或明确暴露依赖风险。注意：按既定设计，不应把该限制扩散到 `parseRRuleSet`、`ValidateRRule` 或 `NextAfter`；这是依赖库行为 caveat，不等于要求共享解析层新增校验。

#### P2 — 显式但当前不影响 occurrence 的 WKST 没有可见性，和“所有已解析条件均显式展示”存在口径差异

- `common/crontask/describe.go:34-36` 只在 WEEKLY 且 `INTERVAL > 1` 或有 BYSETPOS 时显示 WKST。
- 对 `FREQ=WEEKLY;INTERVAL=1;WKST=SU;BYDAY=MO`，WKST 已解析但不会显示；对 DAILY/MONTHLY 的 WKST 也不会显示。
- `.trellis/spec/backend/crontask-guidelines.md:149` 只硬性要求在会影响相位/分组的两种 WEEKLY 场景显示，因此当前实现满足这条窄契约；但用户本轮提出“explicitly rendering all parsed/expressible conditions”，而一周七天即使冗余也被要求可见。二者口径需由主代理确认。
- renderer 只读 normalized `rule.Options`（`describe.go:106-119`），显式 `WKST=MO` 与默认 MO 不可区分；这与设计的 normalized-options 决策一致，不应机械改为扫描原字符串。

#### P2 — 中文骨架语义基本准确，但低频/无条件频率存在机械空格与名词堆叠

- SECONDLY 无时钟维度时由 `describe.go:44-66` 形成“每秒 执行”或“按 2 秒间隔 执行”，中间多一个空格。
- HOURLY/MINUTELY 常形成“每小时 时间条件：……”或“按 2 分钟间隔 时间条件：……”，可理解但不像稳定自然中文。
- “每天，仅限周一至周日”显式保留冗余 BYDAY，语义符合验收；逗号后的“仅限”略显生硬，但不构成 occurrence 错误。
- YEARLY 重复“每年 每年内”已消除；现有测试 `describe_test.go:742-755` 只排除该字面串，当前代表性输出在日期作用域上未发现新的硬错误。

### 已确认符合目的的部分

| 审计项 | 结论 | 证据 |
|---|---|---|
| Frequency 骨架 | 七种频率均有独立中文单位；不引入 JS/英文中间层 | `describe.go:167-190`; `describe_test.go:305-381` |
| INTERVAL phase | `INTERVAL > 1` 明示以开始时间为基准；离散 clock filter 不误写成均匀间隔 | `describe.go:185-189`; `describe_test.go:64-97` |
| DTSTART 日期默认 | YEARLY/MONTHLY/WEEKLY 从 normalized Options/DTSTART 补齐并与 occurrence 一致 | `describe.go:209-223`; `describe_test.go:189-259,305-381` |
| DTSTART 时钟默认 | 对低于 clock 频率的缺省时/分/秒按 DTSTART 补齐 | `describe.go:414-429`; `rrule.go:218-238` |
| BY* 交集与作用域 | BYMONTH、正负 BYMONTHDAY、普通 BYDAY 使用不同维度交集措辞；YEARLY ordinal 的年/月 scope 与库一致 | `describe.go:224-306`; `describe_test.go:597-700` |
| 混合普通/ordinal BYDAY | 安全拒绝，符合 rrule-go 内部两类 mask 同时过滤的交集语义 | `describe.go:152-163`; `rrule.go:596-601` |
| 工作日/周末/七天 | 排序后精确集合归并；一周七天仍显式出现 | `describe.go:309-332`; `describe_test.go:757-829` |
| 时钟笛卡尔积 | 24 项以内展开完整组合；超过 24 项保留全部维度 | `describe.go:376-412`; `describe_test.go:99-102` |
| 一般 BYSETPOS | 正负位置自然排序；按日期×时钟候选解释；WEEKLY 显示 WKST | `describe.go:341-363`; `describe_test.go:261-303,641-672` |
| COUNT/UNTIL | COUNT 明确是周期规则上界；UNTIL 转换到 DTSTART location | `describe.go:68-72,432-443`; `describe_test.go:383-406` |
| timezone/DST | DTSTART location 统一用于边界和列表；fold 列表保留不同 instant/offset；gap 警告承认库归一化 | `describe.go:446-466`; `describe_test.go:408-525` |
| 高级字段 gate | BYYEARDAY/BYWEEKNO/BYEASTER 返回可识别的 `ErrUnsupportedDescription` | `describe.go:122-136`; `describe_test.go:703-740` |
| 解析边界 | 直接消费 `parseRRuleSet`，没有第二套原字符串扫描或组件白名单 | `describe.go:106-119`; `crontask.go:308-320` |

### Upstream human-text 结构审计

当前上游仓库 `jkbrzt/rrule` 的 master HEAD 是 `9f2061febeeb363d03352efe33d30c33073a0242`（2023-11-10）；`src/nlp/totext.ts` blob 为 `e77fd123c5c0f104b6716032dae1247eefacb14a`。该仓库虽然多年未更新，但这是 GitHub 当前 master 内容。

#### 可借鉴的决策结构

- `ToText.toString()` 先选择 Frequency 方法生成主骨架，再追加 UNTIL/COUNT；本地同样先生成频率与条件，再追加边界和 Set 日期。
- constructor 把 BYDAY 分成普通 `allWeeks` 与 ordinal `someWeeks`，并识别工作日和一周七天；本地的 `groupedWeekdayText` 和 mixed ordinal gate吸收了这一思路。
- BYMONTHDAY 正负值分别排序后合并，`-1` 特化为 last；本地也排序并将 -1 渲染成“最后一天”。
- `ToText.IMPLEMENTED` 按 Frequency 声明可转换 option；本地没有照抄静态矩阵，而是依据 rrule-go 的可表达字段做显式 gate，这更符合本任务目标。
- 上游保留 `origOptions` 判断用户是否显式给值，同时读 normalized `options` 获取生效值；本任务设计明确选择最终 `rrule.Options`，仅用 DTSTART 补齐未回写 clock defaults，因此不应机械复制 origOptions 分支。

#### 不应机械复制的上游行为

- `ToText.isFullyConvertible` 在遍历 `origOptions` 时，遇到 `dtstart/tzid/wkst/freq` 使用 `return true` 而不是继续遍历；若该 key 先出现，会跳过后续 unsupported key 检查。这一当前上游 gate 自身并不可靠，不能作为本地支持完整性的权威。
- unsupported option 的上游行为是省略后附加 `(~ approximate)`；本任务要求安全拒绝，不能采用近似描述。
- 上游 `Language` 仅提供 dayNames、monthNames 和 parser tokens；renderer 的词序仍硬编码在 Frequency 方法中。`gettext` 与 `dateFormatter` 可替换 token/日期格式，但不足以把英文语法转换成中文。因此本地直接生成中文是正确边界。
- 当前上游 `ToText.IMPLEMENTED` 不含 BYSETPOS、BYMINUTE、BYSECOND、时区 Set/RDATE/EXDATE 等本任务关键语义；`test/nlp.test.ts` 也主要测试基础英语 round-trip、weekday/monthday 排序和日期 formatter，不能作为项目场景矩阵。

### Files Found

| File Path | Description |
|---|---|
| `common/crontask/describe.go` | 中文 renderer、支持 gate、频率/日期/时间/BYSETPOS/边界/Set 日期实现 |
| `common/crontask/describe_test.go` | 当前 829 行场景、occurrence differential 和 option-field matrix 测试 |
| `common/crontask/crontask.go` | `parseRRuleSet`、`ValidateRRule`、`NextAfter` 的共享解析/执行边界 |
| `common/crontask/errors.go` | `ErrUnsupportedDescription` 定义 |
| `go.mod` | 锁定 `github.com/teambition/rrule-go v1.8.2` |
| `.trellis/spec/backend/crontask-guidelines.md` | RRULE Set、描述、INTERVAL、BYSETPOS、时区和错误契约 |
| `.trellis/spec/backend/quality-guidelines.md` | 纯函数、调度和公共 API 的验证要求 |
| `.trellis/tasks/07-28-audit-rrule-description/prd.md` | 本轮验收目的和不可承诺 occurrence 的明确要求 |
| `.trellis/tasks/07-28-audit-rrule-description/design.md` | normalized Options、解析边界及中文直接生成设计 |
| `.trellis/tasks/07-28-audit-rrule-description/research/scenario-audit.md` | 第一轮场景矩阵与依赖风险记录 |

### External References

- [jkbrzt/rrule `src/nlp/totext.ts`](https://github.com/jkbrzt/rrule/blob/master/src/nlp/totext.ts) — Frequency 骨架、BYDAY 分类、orig/normalized options 和自然语言 renderer。
- [jkbrzt/rrule `src/nlp/index.ts`](https://github.com/jkbrzt/rrule/blob/master/src/nlp/index.ts) — `ToText.IMPLEMENTED` 的按频率 option 支持声明。
- [jkbrzt/rrule `src/nlp/i18n.ts`](https://github.com/jkbrzt/rrule/blob/master/src/nlp/i18n.ts) — Language 只含日期名称与 parser token，不提供完整语言语法。
- [jkbrzt/rrule `test/nlp.test.ts`](https://github.com/jkbrzt/rrule/blob/master/test/nlp.test.ts) — 当前上游 human-text 测试范围。
- [teambition/rrule-go v1.8.2 `rrule.go`](https://github.com/teambition/rrule-go/blob/v1.8.2/rrule.go) — 项目权威候选生成、默认值、过滤与 BYSETPOS 语义。
- [teambition/rrule-go v1.8.2 `rruleset.go`](https://github.com/teambition/rrule-go/blob/v1.8.2/rruleset.go) — RRULE/RDATE 合并、去重及 EXDATE 排除顺序。
- [teambition/rrule-go v1.8.2 `str.go`](https://github.com/teambition/rrule-go/blob/v1.8.2/str.go) — Set、TZID、RDATE/EXDATE 和 ROption 解析行为。

### Related Specs

- `.trellis/spec/backend/crontask-guidelines.md` — 完整 RRULE Set、中文描述及 occurrence 一致性契约。
- `.trellis/spec/backend/quality-guidelines.md` — 调度/公共纯函数的风险验证范围。
- `.trellis/spec/backend/coding-standards.md` — 结构化解析、错误可识别性及最小范围原则。

## Verification Performed

- `go test ./common/crontask` — 通过（cached）。
- `go test -race ./common/crontask` — 通过（cached）。
- `go vet ./common/crontask` — 通过，无输出。
- `git diff --check` — 通过，无输出。
- `git status --short`、目标文件 staged/unstaged diff — 均无输出；审计时工作树没有未提交变更，因此“including uncommitted changes”没有额外 diff 可读。

测试通过只证明现有断言为绿；P0/P1 复现维度目前不在 `describe_test.go` 中。

## Caveats / Not Found

- 本轮只写研究文件，没有修改生产代码、测试或 spec。
- 没有发现 JS 依赖、英文翻译中间层或重复扫描原始 Set 的实现。
- floating DATE-TIME、未知组件、重复 RRULE、EXRULE、PERIOD、VALUE=DATE 和 content-line unfolding 继续沿用任务已决定的解析边界，本轮不把它们重新定义为 renderer 缺陷。
- MONTHLY ordinal panic 属于锁定依赖的运行时缺陷；本报告只描述其对“安全文案”的影响，不主张改变共享调度入口的兼容边界。
- DST gap/fold 的 occurrence 取决于 Go 时区库和 `time.Date` 归一化；当前 warning 已明确以库结果为准，本轮未发现 renderer 私自模拟 DST 的行为。
