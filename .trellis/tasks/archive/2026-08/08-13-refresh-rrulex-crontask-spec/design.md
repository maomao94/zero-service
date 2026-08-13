# 设计：rrulex 与 crontask 规格边界

## 现状

- `common/rrulex` 已拥有 `ParseSet`、`Validate`、`ShiftSetForQuery`、`NextRuns`、`Describe` 及其差分/边界测试。
- `common/crontask` 只通过 `rrulex.NextRuns` 计算和预览计划，但现有 `crontask-guidelines.md` 仍承载大量 rrule 算法与描述器细节。
- ISP/Trigger Store 在 Enable 时使用 `rrulex.ParseSet` + 官方 `Set.After`；ISP 的无效区间通过 `InvalidTimePredicate` 接入 Scheduler。

## 目标文件

### backend/rrulex-guidelines.md

负责：

- 完整 RRULE Set 的解析与校验。
- 单次官方 `Set.After` 与批量 `NextRuns` 的选择。
- `inc`、`count`、predicate、RDATE/EXDATE、耗尽和错误语义。
- `ShiftSetForQuery` 的安全回退、INTERVAL 相位、DST 和差分测试。
- `Describe` 与 `ErrUnsupportedDescription`。
- 当前实现限制及对应测试缺口。

### backend/crontask-guidelines.md

负责：

- TaskConfig 时间字段、claim/lease、完成 CAS、RunNow、MaxDelay、Store 适配。
- Scheduler 如何绑定 `InvalidTimePredicate` 并调用 `rrulex.NextRuns`。
- 创建、完成推进、Preview、Enable 的一致性要求。
- 指向 rrulex 规范，不重复算法实现细节或描述器完整契约。

## 已知限制的写法

规格区分三类内容：

1. 已有可靠契约：有源码与测试共同证明。
2. 使用约束：当前调用方必须遵守，否则可能不终止或产生性能问题。
3. 已知实现限制：代码审阅已确认，但尚未修复；必须明确标注，不能伪装成保障。

## 测试设计

### 官方差分基准

- 参考实现始终使用 `rrulex.ParseSet(value)` 返回的原始 `*rrule.Set`，不调用 `ShiftSetForQuery`。
- 单点基准用官方 `set.After(dt, inc)`；多点基准使用一个原始 `set.Iterator()` 或连续官方查询，结果与 `rrulex.NextRuns` 逐项比较。
- 永久空日历不得无界迭代；测试必须加 COUNT/UNTIL 或选择可达规则。

### DTSTART 与平移

- 查询点早于、等于、晚于 DTSTART。
- `INTERVAL > 1` 的周期锚点必须与 DTSTART 同相位。
- 平移锚点必须不晚于查询点；无法满足时返回 nil 回退原 Set。
- YEARLY/MONTHLY 使用日历月/年差；WEEKLY/DAILY 必须覆盖 DST 回拨和前拨，不能用绝对小时数误判日历天数。
- HOURLY/MINUTELY/SECONDLY 使用 duration 相位，并保留低位时钟校验。

### RRULE 外业务校验

- `InvalidTimePredicate` 仅排除候选，不改变 RRULE 集合；ISP 窗口为 `[start, end]` 闭区间。
- 测试分别断言 start、窗口内、end 为 invalid，窗口前后为 valid。
- 谓词一般可达性无法由 rrulex 证明；测试只使用可耗尽规则或最终恢复为 false 的有限窗口。

## 最小修复策略

- 优先添加差分测试再修复。
- 修复只调整 `common/rrulex` 的平移/迭代正确性，不引入业务专用分支。
- 任一平移结果若晚于查询点、改变 DTSTART/INTERVAL 相位或与官方原始 Set 的 next 不一致，回退原始 Set 优先于冒险优化。
- 不为无法通用求解的永久谓词增加任意扫描上限，以免静默丢失合法候选。

## 验证

- 搜索占位符与旧 API 名称。
- 检查 backend index 与文件集。
- 对规格引用的路径和符号运行 `rg`。
- 运行 rrulex/crontask/ISP/Trigger 的聚焦测试，确认文档签名与当前代码一致。
- 运行全仓测试、race 和 vet，确认新增差分用例与最小修复无回归。
