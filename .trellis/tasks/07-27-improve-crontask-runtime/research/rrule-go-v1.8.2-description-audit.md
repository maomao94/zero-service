# Research: rrule-go v1.8.2 中文描述准确性核对

- **Query**: 全面读取锁定的 `github.com/teambition/rrule-go v1.8.2` 与 RFC 5545 expand/limit 规则，逐项核对 `common/crontask/describe.go` 和测试的中文描述准确性
- **Scope**: mixed
- **Date**: 2026-07-27

## Findings

### Files Found

| File Path | Description |
|---|---|
| `common/crontask/describe.go:14-53` | 描述入口及周期、条件、边界、RDATE/EXDATE 的拼装顺序 |
| `common/crontask/describe.go:56-74` | HOURLY/MINUTELY/SECONDLY 固定日内时刻简化条件 |
| `common/crontask/describe.go:85-128` | Set 形状预校验及 `rrule.Set` 提取 |
| `common/crontask/describe.go:130-158` | 高级规则拒绝策略 |
| `common/crontask/describe.go:160-175` | 七种 FREQ 与 INTERVAL 主文案 |
| `common/crontask/describe.go:177-235` | BYMONTH/BYMONTHDAY/BYDAY/BYSETPOS 文案 |
| `common/crontask/describe.go:237-286` | 时间默认值、笛卡尔积和维度压缩 |
| `common/crontask/describe.go:288-315` | DTSTART/UNTIL 边界与 RDATE/EXDATE 时区展示 |
| `common/crontask/describe_test.go:12-351` | 当前全部描述测试分支；包含基础频率、相位、BYSETPOS、COUNT+UNTIL、DST fold |
| `common/crontask/crontask.go:280-320` | 项目公用 Set parser；仅换行归一化，要求解析后有 DTSTART 和 RRULE |
| `$GOMODCACHE/github.com/teambition/rrule-go@v1.8.2/rrule.go:148-256` | 默认值、DTSTART 派生条件和 timeset 建立 |
| `$GOMODCACHE/github.com/teambition/rrule-go@v1.8.2/rrule.go:492-548` | 各 FREQ 的日期周期集合与 HOURLY/MINUTELY/SECONDLY 时间展开 |
| `$GOMODCACHE/github.com/teambition/rrule-go@v1.8.2/rrule.go:577-825` | 实际 occurrence 生成、全部 BY* 过滤、BYSETPOS、COUNT/UNTIL、INTERVAL 推进 |
| `$GOMODCACHE/github.com/teambition/rrule-go@v1.8.2/rrule.go:880-902` | DTSTART 相位及首周期 timeset 初始化 |
| `$GOMODCACHE/github.com/teambition/rrule-go@v1.8.2/str.go:168-267` | RRULE 属性 parser；大小写、未知属性、COUNT/UNTIL 等行为 |
| `$GOMODCACHE/github.com/teambition/rrule-go@v1.8.2/str.go:288-367` | RRULE Set parser；首行 DTSTART、重复/未知组件行为 |
| `$GOMODCACHE/github.com/teambition/rrule-go@v1.8.2/str.go:380-465` | DTSTART/RDATE/EXDATE 的参数、TZID、VALUE 解析 |
| `$GOMODCACHE/github.com/teambition/rrule-go@v1.8.2/rruleset.go:133-174` | RRULE 与 RDATE 合并去重、EXDATE 精确时刻排除 |

### Executive Result

当前描述器的基本 INTERVAL 文案、DTSTART 默认日期/时间补足、低频时间笛卡尔积、COUNT 上界措辞、UNTIL 时区转换、RDATE/EXDATE 列表、BYSETPOS 示例均与 `rrule-go v1.8.2` 的常规结果一致。但存在 **2 个高严重度、3 个中严重度准确性问题**，另有若干 parser/RFC 差异和 DST 残余风险。最重要的是：

1. `WKST` 完全未输出，却会改变 `WEEKLY;INTERVAL>1` 的实际周集合相位。
2. 带 DST 的命名时区遇到不存在的本地时间时，`rrule-go` 会生成另一个墙钟时间，而描述仍断言固定墙钟时刻。
3. 通用“周期 + 条件”措辞没有区分 expand 和 filter，某些合法规则会把“一年内很多次”读成“每年一次”，或产生“每天 第 1 天”这种矛盾句子。

## RFC 5545 Expand / Limit 与 rrule-go 真实行为

RFC 5545 §3.3.10 的顺序是：`FREQ`/`INTERVAL` → BYMONTH → BYWEEKNO → BYYEARDAY → BYMONTHDAY → BYDAY → BYHOUR → BYMINUTE → BYSECOND → BYSETPOS → COUNT/UNTIL。表中 `Expand` 表示在当前频率周期内增加候选，`Limit` 表示筛除候选，`N/A` 表示 RFC 禁止该组合。

| BY part | YEARLY | MONTHLY | WEEKLY | DAILY | HOURLY | MINUTELY | SECONDLY | rrule-go v1.8.2 实现依据 |
|---|---|---|---|---|---|---|---|---|
| BYMONTH | Expand | Limit | Limit | Limit | Limit | Limit | Limit | 年周期 dayset 是全年，其余周期更小；统一在 `rrule.go:594` 按月份过滤，因 dayset 大小自然形成 expand/limit |
| BYMONTHDAY | Expand | Expand | RFC N/A；库实际 Limit | Limit | Limit | Limit | Limit | 年/月 dayset 包含多日；周/日及更高频仅过滤，`rrule.go:599-601` |
| BYDAY | Expand（有 BYYEARDAY/BYMONTHDAY 时为 Limit；有 BYWEEKNO 有特殊限制语义） | Expand（有 BYMONTHDAY 时为 Limit） | Expand | Limit | Limit | Limit | Limit | 普通 weekday mask 与序号 weekday mask 均在日集合上过滤，`rrule.go:444-479,596-597`；效果取决于周期 dayset |
| BYHOUR | Expand | Expand | Expand | Expand | Limit | Limit | Limit | 低于 HOURLY 的 FREQ 预建 `hours × minutes × seconds`，`rrule.go:243-253`；HOURLY+ 在推进循环过滤，`rrule.go:745,768-769,797-800` |
| BYMINUTE | Expand | Expand | Expand | Expand | Expand | Limit | Limit | HOURLY 的 timeset 展开分钟和秒，`rrule.go:526-535`；MINUTELY+ 推进过滤 |
| BYSECOND | Expand | Expand | Expand | Expand | Expand | Expand | Limit | MINUTELY timeset 展开秒，`rrule.go:536-541`；SECONDLY 推进过滤 |

`BYSETPOS` 在每个 FREQ 周期内，对“过滤后的有效日期 × 当前 timeset”排序后取位置，而不是独立过滤器；见 `rrule.go:614-644`。例如当前测试规则：

```text
DTSTART:20260701T000000Z
RRULE:FREQ=MONTHLY;COUNT=2;BYDAY=MO,TU;BYHOUR=9,17;BYMINUTE=0;BYSECOND=0;BYSETPOS=2
```

实际为 `2026-07-06 17:00Z, 2026-08-03 17:00Z`。当前“每个周期内取符合上述条件的第 2 个”准确。

## FREQ / INTERVAL 推进核对

| FREQ | rrule-go 推进 | DTSTART 相位 | 当前主文案结论 |
|---|---|---|---|
| YEARLY | `year += interval`，保持初始月/日作为迭代锚点，`rrule.go:696-703` | 首年从 DTSTART 之后截断 | “每年”/“按 N 年间隔”本身准确 |
| MONTHLY | `month += interval` 并归一化年份，`rrule.go:704-720` | 从 DTSTART 所在月起；无效月日不补偿 | “每月”/“按 N 个月间隔”本身准确 |
| WEEKLY | 按 WKST 对齐后增加 `interval*7`，`rrule.go:721-728` | 首周从 DTSTART 起截断；WKST 决定间隔周分组 | 主文案遗漏 WKST，见高严重度问题 1 |
| DAILY | `day += interval`，`rrule.go:729-731` | 从 DTSTART 日期相位推进 | 准确 |
| HOURLY | 在绝对的本地字段上 `hour += interval`，跨日归一，`rrule.go:732-749` | DTSTART 小时相位；BYHOUR 是过滤 | 非 DST 情况准确；DST 见高严重度问题 2 |
| MINUTELY | `minute += interval`，跨小时/日归一，`rrule.go:750-773` | DTSTART 分钟相位；BYHOUR/BYMINUTE 是过滤 | 非 DST 情况准确；`INTERVAL>1` 不简化为每日固定时刻是正确的 |
| SECONDLY | `second += interval`，跨分钟/小时/日归一，`rrule.go:774-803` | DTSTART 秒相位；BYHOUR/BYMINUTE/BYSECOND 是过滤 | 非 DST 情况准确 |

`INTERVAL=0` 被库归一为 1（`rrule.go:154-158`），虽然 RFC 要求正整数。当前描述也归一为 1，因此描述符合库结果，但输入不是严格 RFC 5545。

## Issues by Severity

### High 1 — WEEKLY + INTERVAL 的 WKST 相位被文案遗漏

- **规则 A**：

  ```text
  DTSTART:20240103T090000Z
  RRULE:FREQ=WEEKLY;INTERVAL=2;COUNT=8;WKST=SU;BYDAY=SU,MO
  ```

- **真实 occurrence**：`2024-01-14, 01-15, 01-28, 01-29, 02-11, 02-12, 02-25, 02-26`，均为 09:00Z。
- **对照规则（仅 WKST=MO）**：`2024-01-07, 01-15, 01-21, 01-29, 02-04, 02-12, 02-18, 02-26`。
- **当前文案**：两者都是“按 2 周间隔 周一、周日 09:00 执行……”，无法区分不同 occurrence 集合。
- **准确文案/拒绝策略**：需要表达周起始和 DTSTART 相位，例如“以周日为每周起始，按 2 周间隔，在周日、周一 09:00 执行”；若不展示 WKST，则该组合应拒绝描述。`INTERVAL=1` 且没有 BYWEEKNO 时 WKST 通常不改变集合，可不必展示。
- **源码依据**：`rrule.go:181` 保存 WKST；`rrule.go:501-517` 构造周周期；`rrule.go:721-728` 以 WKST 对齐间隔周。当前 `describe.go:160-175,177-221` 完全未消费 `option.Wkst`。

### High 2 — DST spring-forward 缺口中“每天固定墙钟时间”与库实际结果不一致

- **规则**：

  ```text
  DTSTART;TZID=America/New_York:20240309T023000
  RRULE:FREQ=DAILY;COUNT=4
  ```

- **真实 occurrence（rrule-go + Go time）**：
  - `2024-03-09 02:30:00 -05:00`
  - `2024-03-10 01:30:00 -05:00`（02:30 当地时间不存在，被 `time.Date` 归一到 01:30）
  - `2024-03-11 02:30:00 -04:00`
  - `2024-03-12 02:30:00 -04:00`
- **当前文案**：“每天 02:30 执行”。3 月 10 日实际不是 02:30。
- **准确文案/拒绝策略**：若目标是描述库的真实执行，应对有 DST 的命名时区及可能落入缺口的墙钟规则拒绝固定时刻描述，或明确披露“遇夏令时不存在时刻按 Go 时区归一结果执行”。不能继续无条件断言“每天 02:30”。
- **RFC 差异**：RFC 5545 §3.3.10 要求无效日期/不存在的本地时间忽略且不计数；库没有忽略，而是在 `time.Date` 构造 occurrence 时发生 Go 归一化。
- **源码依据**：`rrule.go:243-249` 以 DTSTART location 构造 timeset；`rrule.go:669-674` 用 `time.Date` 合成 occurrence。当前 `describe.go:253-269` 只格式化名义时刻。
- **补充**：fall-back fold 的每日 01:30 只生成第一次 01:30（例如 2024-11-03 为 `01:30 -04:00`），当前文案不区分 fold 的哪一个 instant。现有测试 `describe_test.go:334-351` 只验证两个显式 RDATE 在 fold 中不被错误去重，没有覆盖 RRULE 自身的 DST 行为。

### Medium 1 — expand 语义被渲染成模糊或矛盾的“周期 + 第 N 天/星期”

#### Case A：YEARLY + BYMONTHDAY（无 BYMONTH）

- **规则**：`DTSTART:20240115T090000Z\nRRULE:FREQ=YEARLY;COUNT=8;BYMONTHDAY=1`
- **真实 occurrence**：`2024-02-01, 03-01, 04-01, 05-01, 06-01, 07-01, 08-01, 09-01` 09:00Z。BYMONTHDAY 在 YEARLY 内展开为每个月的 1 日；首年 1 月 1 日因早于 DTSTART 被截断。
- **当前文案**：“每年 第 1 天 09:00 执行”。这通常会被理解为每年 1 月 1 日一次，并且“第 1 天”未说明是每月第 1 天。
- **准确文案**：“每年内各月第 1 天 09:00 执行”，或等价、明确表达一年内多次的措辞。

#### Case B：YEARLY + BYDAY（无 BYMONTH/BYWEEKNO 等）

- **规则**：`DTSTART:20240115T090000Z\nRRULE:FREQ=YEARLY;COUNT=8;BYDAY=MO`
- **真实 occurrence**：`2024-01-15, 01-22, 01-29, 02-05, 02-12, 02-19, 02-26, 03-04` 09:00Z。
- **当前文案**：“每年 周一 09:00 执行”。可能被理解为每年某一个周一，而实际是全年所有周一。
- **准确文案**：“每年内每个周一 09:00 执行”。

#### Case C：DAILY + BYMONTHDAY

- **规则**：`DTSTART:20240115T090000Z\nRRULE:FREQ=DAILY;COUNT=4;BYMONTHDAY=1`
- **真实 occurrence**：`2024-02-01, 03-01, 04-01, 05-01` 09:00Z；此处 BYMONTHDAY 是日频率过滤器。
- **当前文案**：“每天 第 1 天 09:00 执行”，语法和语义矛盾。
- **准确文案**：“每天推进，仅每月第 1 天 09:00 执行”，或更自然的等价过滤措辞。

- **共同源码依据**：周期 dayset 见 `rrule.go:492-523`，过滤见 `rrule.go:591-612`；RFC expand/limit 表。当前 `describe.go:33-39` 对所有 FREQ 使用同一连接方式，`renderDateFilters` 不知道每个条件在该 FREQ 下是 expand 还是 filter。

### Medium 2 — “有效期”错误暗示限制整个 Set，但 UNTIL 只限制 RRULE

- **规则**：

  ```text
  DTSTART:20240110T090000Z
  RRULE:FREQ=DAILY;UNTIL=20240112T090000Z
  RDATE:20240101T090000Z,20240201T090000Z
  ```

- **真实 occurrence**：`2024-01-01, 01-10, 01-11, 01-12, 02-01` 09:00Z。RDATE 可以早于 DTSTART 或晚于 UNTIL。
- **当前文案**：“……有效期：2024-01-10 09:00:00 至 2024-01-12 09:00:00，额外执行：2024-01-01……、2024-02-01……”。“有效期”容易被理解为整个 schedule 的硬边界，但额外执行落在两端之外。
- **准确文案**：边界作用域需明确为“周期规则有效期”；RDATE 单独作为不受该边界约束的额外 occurrence。
- **源码依据**：UNTIL 只在 RRULE iterator 中判断，`rrule.go:646-679`；RDATE 与 RRULE 后续合并，`rruleset.go:133-174`。当前 `describe.go:50-52,288-315` 没有给边界加“周期规则”限定。

### Medium 3 — parser 接受部分非 RFC 数值，描述把它们静默正规化为普通无限规则

- `COUNT=-1`：`StrToROption` 接受整数；`buildRRule` 在 `rrule.go:160-163` 改为 0，即无限，不显示 COUNT。RFC COUNT 必须为正整数。
- `COUNT=0`：同样成为无限规则。
- `INTERVAL=0`：在 `rrule.go:154-158` 改为 1；描述器 `frequencyText` 也显示普通每单位一次。
- `BYDAY=0MO`：parser 可解析，库按无序号 MO 处理；RFC ordinal weekday 不允许 0。
- **当前策略**：均可能返回成功文案，掩盖输入不符合 RFC 的事实。
- **准确策略**：如果 API 契约是“RFC 5545 RRULE Set”，这些值应作为解析/校验错误拒绝，而不是描述为另一个规则。这里是输入合法性问题，不是 occurrence 渲染问题。
- **源码依据**：`str.go:225-250` 只做 Atoi；`rrule.go:259-311` 未校验 COUNT 正数，也只拒绝负 INTERVAL。

## DTSTART、相位、首周期截断、无效日期

- ROption 未提供 DTSTART 时，库默认 `time.Now().UTC()` 并截断到秒，`rrule.go:165-170`。项目的 `parseRRuleSet` 又要求 Set 中显式 DTSTART，因此 `DescribeRRule` 的公开入口不会依赖动态 now；这一点正确。
- YEARLY 无任何日选择器时，库补 DTSTART 月和日；MONTHLY 补 DTSTART 日；WEEKLY 补 DTSTART weekday，`rrule.go:184-199`。描述器在 `describe.go:181-190` 做相同展示推导，准确。
- 低频规则未显式给 BYHOUR/BYMINUTE/BYSECOND 时，库按频率层级补 DTSTART 的更低位时间，`rrule.go:218-238`；描述器 `describe.go:241-250` 同步补足，准确。
- 第一 FREQ 周期不是完整周期：生成结果统一要求 `!res.Before(dtstart)`，`rrule.go:650-690`。例如 `DTSTART=2024-01-03 12:00Z; FREQ=WEEKLY;BYDAY=MO,WE,FR;BYHOUR=9` 的首个结果是 1 月 5 日 09:00，而不是首周已经过去的周一/周三。当前“开始于 DTSTART”能提供边界，但没有直接解释首周期截断；不构成独立错误。
- 无效日期不补到月底。`DTSTART=2024-01-31;FREQ=MONTHLY;COUNT=4` 实际是 `2024-01-31, 03-31, 05-31, 07-31`。当前“每月 第 31 天”可结合常识理解为没有 31 日的月份不执行，但没有显式写“无效日期跳过”。这是残余展示风险。
- RFC 要求无效日期和不存在的本地时间忽略且不计 COUNT；库对不存在本地时间存在上述 DST 归一化偏差。

## COUNT / UNTIL / WKST / BYSETPOS / RDATE / EXDATE

- **COUNT**：在 RRULE 产生一个不早于 DTSTART 的候选时递减，`rrule.go:650-689`；之后 Set 才应用 EXDATE，所以 EXDATE 不会“补足”COUNT。描述器写“周期规则最多生成 N 次”准确，且没有声称最终 Set 总数。
- **UNTIL**：inclusive，只有 `res.After(until)` 才停止，`rrule.go:646-677`。UTC UNTIL 会被 `time.Time` instant 比较，描述转 DTSTART location，准确。
- **COUNT + UNTIL**：RFC 5545 规定二者不得同时出现；库允许并取先到上界。例如 COUNT=10、UNTIL=2024-01-03 09:00Z 最终 3 次。当前同时显示“最多生成 10 次”和有效至日期，准确描述了库行为，但该输入不是严格 RFC。
- **WKST**：默认 MO；影响 interval 周分组及 BYWEEKNO。描述器从不渲染，详见 High 1。
- **BYSETPOS**：当前已正确放在日期和时间展开之后描述。验证只允许有 BYMONTHDAY/BYDAY/BYHOUR/BYMINUTE/BYSECOND 之一；这比库保守，例如仅 BYMONTH+BYSETPOS 会被拒绝，属于明确拒绝而非误导。
- **RDATE**：与 RRULE 做有序并集并以 `time.Equal` 去重，`rruleset.go:138-170`。可在 DTSTART/UNTIL 外。当前排序去重展示正确。
- **EXDATE**：按完整 instant 精确排除 RRULE 或 RDATE，`rruleset.go:145-170`。`EXDATE;VALUE=DATE:20240102` 在 DTSTART location 解析成当天 00:00，只会排除 midnight occurrence；不会排除当天 09:00。当前展示成明确的 00:00 datetime，符合库真实行为。

## 时区与 DST

- DTSTART 有 TZID 时，Set parser 将其 location 作为无时区 RDATE/EXDATE/UNTIL 的默认 location，`str.go:319-327,341-352`。
- 显式 Z 时间保持 UTC instant；描述器统一 `.In(dtstart.Location())`，`describe.go:288-315`，符合需求。
- RDATE/EXDATE 可各带不同 TZID；描述统一转换 DTSTART location。DST fold 中两个不同 instant 即使墙钟文字相同也通过 offset 区分，现有 `describe_test.go:334-351` 覆盖正确。
- RRULE 在 DST gap 的实际异常见 High 2。HOURLY/MINUTELY/SECONDLY 同样依赖 Go `time.Date` 与字段归一化，跨 DST 的“每 N 小时”不应被理解成稳定 elapsed-duration 间隔。

## RRULE Set Parser 实际行为

以下分“库自身”与“当前 DescribeRRule 包装层”说明，避免混淆：

| 输入特征 | rrule-go v1.8.2 | 当前 DescribeRRule |
|---|---|---|
| DTSTART 不在第一行 | 不识别为 Set DTSTART；该行若后续出现会被 switch 忽略 | 最终因缺 DTSTART 拒绝 |
| 重复 DTSTART | 第一条生效，后续 DTSTART 被未知 switch 分支静默忽略 | `validateDescriptionSetShape` 预先拒绝 |
| 重复 RRULE | 后一条调用 `set.RRule` 覆盖前一条 | 以 `ErrUnsupportedDescription` 拒绝 |
| 未知 Set 组件（如 FOO、EXRULE） | `str.go:339-363` 无 default，静默忽略 | 预校验以 `ErrUnsupportedDescription` 拒绝 |
| LF | 支持 | 支持 |
| CRLF | 库直接调用会把 `\r` 留在值中而常解析失败 | `parseRRuleSet` 先改为 LF，支持 |
| RFC 行折叠 CRLF + SPACE/TAB | 不做 unfolding，失败 | 不做 unfolding；预校验将续行当独立非法行，失败 |
| 小写 property 名 | `processRRuleName` 大写名称，通常可识别 | 预校验也大写名称 |
| 小写 RRULE key/value | RRULE switch 和 `StrToFreq` 大小写敏感，失败；与 RFC 名称不区分大小写有差异 | 同样失败 |
| RRULE 未知 key | 明确 `unknown RRULE property` error | 同样失败 |
| DTSTART 参数 | 仅接受单个 `TZID=...` 形态；`VALUE=DATE` 等会被 `parseTZID` 拒绝 | 同样失败 |
| RDATE/EXDATE 参数 | 支持 `TZID`、`VALUE=DATE-TIME`、`VALUE=DATE`；未知参数拒绝；不支持 PERIOD | 同样行为，随后按 datetime 展示 |
| RDATE/EXDATE 多行及逗号多值 | 都追加到 Set | 支持并排序去重展示 |

重复与未知组件的生产描述入口行为符合“无法准确描述则拒绝”的项目策略。行折叠虽然 RFC 要求 parser unfolding，但当前项目 parser 明确不支持；应作为已知输入限制记录，而不能声称接受任意 RFC content line。

## 当前描述器全部输出分支核对

| 输出分支 | 位置 | 核对结果 |
|---|---|---|
| 周期文案 | `describe.go:160-175` | 七种 FREQ 与 INTERVAL 数值准确；WEEKLY 缺 WKST；单独主句不能表达 expand/limit |
| 日期条件 | `describe.go:177-221` | 排序去重、负月日、序号星期格式正确；“第 N 天”没有说明月内含义，跨 FREQ 时出现误导，见 Medium 1 |
| 固定时间简化 | `describe.go:56-74` | `INTERVAL=1` 且显式高位过滤时的 HOURLY/MINUTELY/SECONDLY 简化，在非 DST 常规日期上等价；BYSETPOS 和 INTERVAL>1 正确禁止简化；DST gap 不准确 |
| 时间小集合笛卡尔积 | `describe.go:253-269` | 与库 timeset 的 hour × minute × second 一致，阈值 24 只影响展示方式 |
| 时间条件压缩 | `describe.go:272-285` | 各维度列表准确，但对于等于/高于 FREQ 的维度实际是相位推进上的过滤，文字“时间条件”足够中性 |
| BYSETPOS | `describe.go:42-44,224-235` | 在已接受组合中顺序和“每周期取位置”准确 |
| COUNT | `describe.go:47-49` | “最多生成”准确覆盖 UNTIL、EXDATE/RDATE；没有错称最终 Set 次数 |
| DTSTART/UNTIL 边界 | `describe.go:288-300` | 时区转换正确；“有效期”应限定为周期规则，见 Medium 2 |
| 额外/排除 | `describe.go:302-315` | 精确 datetime、offset、排序、按 instant 去重均准确；RDATE 可越过边界需结合边界措辞理解 |
| 错误策略 | `describe.go:105-157` | 明确拒绝未知/重复组件、BYYEARDAY/BYWEEKNO/BYEASTER、部分 BYSETPOS、周/日频率序号 BYDAY；但非 RFC 数值仍被库正规化，见 Medium 3 |

## Code Patterns

### rrule-go 的统一实现方式

库没有直接编码一张 expand/limit 表，而是通过“FREQ 决定 dayset/timeset 大小 + 同一套过滤器”实现等价效果：

```go
// rrule.go:585-601（节选）
setStart, setEnd := iterator.ii.calcDaySet(r.freq, iterator.year, iterator.month, iterator.day)
iterator.fillDaySetMonotonic(setStart, setEnd)
for dayIndex, day := range dayset {
    i := day.Int
    if len(r.bymonth) != 0 && !contains(r.bymonth, iterator.ii.mmask[i]) ||
       ... ||
       (len(r.bymonthday) != 0 || len(r.bynmonthday) != 0) && ... {
        dayset[dayIndex].Defined = false
    }
}
```

YEARLY 的 dayset 是全年，MONTHLY 是整月，WEEKLY 是到下个 WKST 前的一周，DAILY 及更高频只有当天（`rrule.go:492-523`），所以相同过滤语句会自然表现成 expansion 或 limitation。

### Set 的并集、去重和排除顺序

`rruleset.go:133-174` 先将 RDATE 和单个 RRULE 的 iterator 合并排序，以 `time.Equal` 去重，然后用 EXDATE 精确 instant 排除。因此：

```text
Final Set = unique(RRULE occurrences ∪ RDATE) − EXDATE
```

COUNT 和 UNTIL 只在 RRULE 内部作用，不控制 RDATE，也不在 EXDATE 后重新计数。

## External References

- [RFC 5545 §3.3.10 Recurrence Rule](https://www.rfc-editor.org/rfc/rfc5545.html#section-3.3.10) — expand/limit 表、处理顺序、COUNT/UNTIL 互斥、无效日期/不存在本地时间忽略规则
- [RFC 5545 §3.1 Content Lines](https://www.rfc-editor.org/rfc/rfc5545.html#section-3.1) — CRLF、75 octet line folding 与 parser 必须 unfolding
- [RFC 5545 §3.8.5.1 EXDATE](https://www.rfc-editor.org/rfc/rfc5545.html#section-3.8.5.1) — recurrence set 排除语义
- [RFC 5545 §3.8.5.2 RDATE](https://www.rfc-editor.org/rfc/rfc5545.html#section-3.8.5.2) —额外 recurrence date 语义
- [rrule-go v1.8.2 module](https://github.com/teambition/rrule-go/tree/v1.8.2) — 项目锁定实现；本报告以本机 Go module cache 的精确版本源码为准

## Related Specs

- `.trellis/spec/backend/crontask-guidelines.md:130-186` — RRULE 中文业务描述契约，特别是 INTERVAL 相位、维度并交关系、固定时间简化和 COUNT 上界措辞
- `.trellis/tasks/07-27-improve-crontask-runtime/prd.md:27-39` — DescribeRRule 输入、时区、支持字段与错误策略需求
- `.trellis/tasks/07-27-improve-crontask-runtime/design.md:51-87` — normalization、rendering、BYSETPOS、EXDATE 设计

## Caveats / Not Found

- 临时 Go harness 位于系统临时目录 `/var/folders/.../T/opencode/rrule-research/`，未在生产目录创建临时文件。
- RFC 5545 的 expand/limit 表包含本任务明确不支持的 BYWEEKNO/BYYEARDAY；本报告仅为解释 BYDAY 特殊语义提及，当前描述器拒绝它们是稳定且不误导的行为。
- `rrule-go v1.8.2` README 自称源自 RFC 2445/python-dateutil，但代码并非完全执行 RFC 5545 验证，尤其 COUNT+UNTIL、非法数值、行折叠和 DST gap。描述“真实行为”与宣称“严格 RFC”必须区分。
- 没有发现当前测试覆盖 `WKST` 导致的不同 occurrence、RRULE 自身 DST gap、YEARLY 下 BYMONTHDAY/BYDAY expansion、DAILY 下 BYMONTHDAY filter、RDATE 越过 UNTIL 边界或非正 COUNT/INTERVAL。
- 即使修正文案，所有带命名时区的长期规则仍有未来 tzdata 变化风险；描述是在当前运行环境时区数据库下解释规则。
