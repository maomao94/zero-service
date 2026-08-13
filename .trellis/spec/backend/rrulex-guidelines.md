# rrulex RRULE 扩展规范

## 1. 适用范围

修改 `common/rrulex` 的完整 Set 解析、查询起点平移、批量 occurrence、谓词过滤或中文描述时读取。Scheduler、lease、Store 和任务状态属于 [crontask 调度规范](./crontask-guidelines.md)；Trigger/ISP 如何生成规则仍由各自领域规范拥有。

`common/rrulex` 只补充 `github.com/teambition/rrule-go` 缺少的跨服务机制，不导入具体服务的 proto、model 或 `internal/` 包。依据：`common/rrulex/rrulex.go`、`query.go`、`describe.go` 及其测试。

## 2. 公开签名与所有权

```go
func ParseSet(value string) (*rrule.Set, error)
func Validate(value string) error
func ShiftSetForQuery(set *rrule.Set, after time.Time) *rrule.Set
func NextRuns(value string, dt time.Time, inc bool, count int, invalid func(time.Time) bool) ([]time.Time, error)
func Describe(value string) (string, error)

var ErrUnsupportedDescription error
```

- `ParseSet` / `Validate` 拥有完整 RRULE Set 的结构校验。
- `ShiftSetForQuery` 只拥有向前查询的性能优化，不改变持久化规则。
- `NextRuns` 是批量 occurrence 与谓词过滤入口。
- 单次、无谓词的查询使用官方原生 API：`ParseSet` 后调用 `set.After(dt, inc)`；不新增 `NextAfter` 包装。
- `Describe` 只解释已生成的 RFC 5545 Set，不依赖 Trigger/ISP 输入模型。

## 3. 解析、查询与平移契约

### 完整 Set

- `ParseSet` 先 trim，并把 CRLF 归一化为 LF；非空输入必须同时有显式 `DTSTART` 和 `RRULE`。
- `Validate("") == nil`：空字符串由 crontask 解释为一次性任务。`ParseSet("")` 不承担一次性语义，会因缺少 DTSTART/RRULE 返回 error。
- `RDATE` 加入 RRULE 候选并集，`EXDATE` 从合并结果排除；相同候选由官方 Set 迭代器去重。
- 结构校验不等于日历可达性证明。永久空交集、任意谓词是否最终接受等一般问题不能通过无界遍历验证。

### `NextRuns`

- `inc=true` 的首个候选满足 `!v.Before(dt)`；`inc=false` 满足 `v.After(dt)`，与官方 `Set.After` 边界一致。
- `count <= 0` 返回非 nil 空切片和 nil error；规则耗尽返回已收集结果和 nil error；语法/结构错误返回 error。
- `count` 只统计最终接受的候选。被 `invalid` 判为 true 的候选不推进 `dt` 游标。
- 当前约定中 `invalid` 先于 inc 边界判断执行，因此谓词会看到 DTSTART 锚点或等于 `dt`、但最终不被接受的候选。谓词必须无副作用、可重复调用，不能把调用次数当作 occurrence 数量。
- 无边界永久规则若搭配永久返回 true 的谓词不会自然结束。调用方必须保证规则有 COUNT/UNTIL，或谓词在有限未来恢复为 false；不得在 rrulex 中添加任意扫描上限并静默丢弃远期合法候选。

### `ShiftSetForQuery`

- 仅用于 `After` / `Iterator` 等向前查询；`Before` 必须使用原始 Set。
- 新锚点必须：不晚于查询点、保持 DTSTART 的低位时钟/日期相位，并按 `INTERVAL` 的倍数对齐。任一条件不能证明时返回 nil，由调用方使用原始 Set。
- 带 `COUNT`、`BYWEEKNO`、`BYYEARDAY`、`BYEASTER`、`BYSETPOS` 的规则必须回退，因为移动 DTSTART 会改变计数、周期分组或位置语义。
- YEARLY/MONTHLY 使用日历年/月差；`AddDate` 导致月日钳制（如 2 月 29 日、月末 31 日）时回退。
- WEEKLY/DAILY 属于墙钟日历频率。将查询点转换到 DTSTART 的 Location 后，若两者 UTC offset 不同（DST 前拨/回拨），绝对小时数不能安全代表日历天数，必须回退原 Set。
- HOURLY/MINUTELY/SECONDLY 使用 duration 相位，并检查继承自 DTSTART 的分钟/秒低位；非整小时 offset 变化破坏相位时回退。
- 所有频率最终统一检查 `shifted.After(after)`；未来锚点无条件回退。

## 4. Validation & Error Matrix

| 条件 | 结果 |
| --- | --- |
| `Validate("")` | nil，一次性配置合法 |
| 裸 `FREQ=...`、缺 DTSTART 或缺 RRULE | 解析/校验 error |
| 合法 Set 已耗尽 | `NextRuns` 返回空/部分结果和 nil error |
| `count <= 0` | 空切片、nil error，不解析规则 |
| 平移规则含复杂计数/位置条件 | `ShiftSetForQuery` 返回 nil，回退原 Set |
| 平移被日期钳制、DST offset 改变或越过查询点 | 返回 nil，回退原 Set |
| 描述器无法准确表达合法规则 | 包装 `ErrUnsupportedDescription` |
| 永久规则 + 永久拒绝谓词 | 不保证终止；属于调用约束，不是解析错误 |

## 5. Good / Base / Bad

- Good：用原始官方 Set 作为差分基准，证明平移前后 occurrence 序列一致。
- Good：批量预览调用 `NextRuns(value, dt, false, count, predicate)`，predicate 只做纯判定。
- Base：单次 after 用 `set, err := ParseSet(value)` + `set.After(dt, false)`。
- Base：无法安全平移时返回 nil，接受较慢的原始迭代，不猜测日历相位。
- Bad：用 `after.Sub(dtstart).Hours()/24` 跨 DST 推导 DAILY/WEEKLY 日历天数，却不检查 offset 和未来锚点。
- Bad：为永久规则和任意谓词添加固定最大扫描次数，然后把尚未找到候选写成自然耗尽。
- Bad：用 `Set.All()` 展开长期规则，或循环调用官方 `After` 生成前 N 个结果导致重复从起点迭代。
- Bad：把 ParseSet 的结构通过描述成“保证一定存在 occurrence”。

## 6. 测试要求

- 查询优化必须以未经平移的官方 `rrule.Set` 为参考实现；参考路径不得调用 `ShiftSetForQuery` 或 `NextRuns`。
- 表驱动覆盖 YEARLY/MONTHLY/WEEKLY/DAILY/HOURLY/MINUTELY/SECONDLY，查询点在 DTSTART 前、等于、之后，且覆盖 inc=true/false。
- `INTERVAL > 1` 断言锚点与 DTSTART 同相位；任何非 nil 平移锚点都必须 `<= after`。
- 覆盖 RDATE/EXDATE、COUNT、UNTIL、规则耗尽，以及 COUNT/BY* / AddDate 钳制的安全回退。
- DAILY/WEEKLY 必须覆盖 DST 前拨和回拨，并断言 offset 变化时回退；HOURLY/MINUTELY 覆盖跨 DST 后与官方 occurrence 一致。
- predicate 测试明确当前“先 predicate、后 inc”顺序；永久拒绝只能搭配有界规则。
- 描述器测试使用有界规则验证永久空交集，不能迭代无 UNTIL 的永久空位置规则。

验证命令：

```bash
go test -count=1 ./common/rrulex
go test -race -count=1 ./common/rrulex
go vet ./common/rrulex
```

依据：`common/rrulex/rrulex_test.go`、`common/rrulex/describe_test.go`。

## 7. Wrong vs Correct

### Wrong：把优化结果当权威

```go
got, _ := NextRuns(value, dt, false, 10, nil)
// 只断言 got 非空，无法证明平移没有改变序列。
```

### Correct：与原始 Set 差分

```go
set, err := ParseSet(value)
if err != nil {
    return err
}
want := collectFromIterator(set.Iterator(), dt, false, 10)
got, err := NextRuns(value, dt, false, 10, nil)
```

差分必须逐项比较 occurrence；发现不一致时优先让 `ShiftSetForQuery` 回退，而不是放宽测试。

## 中文描述

- `Describe("")` 返回 `"", nil`；非空输入复用 `ParseSet`。
- 周期条件只说明候选如何形成，不承诺一定存在 occurrence。
- `RDATE` 描述为“额外纳入候选”，`EXDATE` 描述为从 RRULE/RDATE 合并结果排除，不能直接称为最终执行。
- `COUNT` 只描述周期规则最多生成次数，不等于 Set 合并后的最终总数。
- 无法准确表达的 BYYEARDAY、BYWEEKNO、BYEASTER、混合普通/序号 BYDAY、危险月序号星期或 BYSETPOS 重复时钟维度返回 `ErrUnsupportedDescription`。
- 描述器以 `rrule.Options` 的归一化结果为准，并结合 DTSTART 补齐默认时分秒；不要重新扫描原始文本实现第二套解析器。

更完整的描述断言见 `common/rrulex/describe_test.go`。
