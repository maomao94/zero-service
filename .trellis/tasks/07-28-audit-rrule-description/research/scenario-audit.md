# RRULE 描述场景审查

> 最终范围决定：下述“缺陷与风险”保留为依赖库审计记录，不在本任务中转化为项目校验。`parseRRuleSet` 继续只是 `rrule.StrToRRuleSet` 加 DTSTART/RRULE 必需检查；`DescribeRRule` 直接消费同一解析结果。

## 范围

- 项目版本：`github.com/teambition/rrule-go v1.8.2`
- 实现：`common/crontask/describe.go`、`common/crontask/crontask.go`
- 生成端：`app/trigger/internal/cronjob/schedule.go`、`app/ispagent/internal/crontask/task_rule.go`

## 已确认正确

- YEARLY 至 SECONDLY 的基础频率、INTERVAL 相位和 DTSTART 默认日期/时刻。
- BYMONTH、正负 BYMONTHDAY、普通 BYDAY 的交集表达。
- YEARLY ordinal BYDAY 在有无 BYMONTH 时的月/年作用域区分。
- BYSETPOS 对周期内日期与时间候选笛卡尔积的位置筛选。
- WEEKLY 的 INTERVAL 或 BYSETPOS 场景展示 WKST。
- COUNT 使用“最多生成 N 次”，UNTIL 按 DTSTART 时区展示。
- RDATE/EXDATE 转到 DTSTART 时区，DST fold 中不同 instant 不被错误去重。

## 缺陷与风险

### 月作用域 ordinal BYDAY

`MONTHLY;BYDAY=-6MO/53MO` 以及 `YEARLY;BYMONTH=...;BYDAY=-6MO` 可能在 rrule-go occurrence 计算中数组越界。该风险本次仅记录，不向共享解析或调度入口增加项目自定义限制。

### 重复时钟值

`BYHOUR=9,9;BYMINUTE=0,0;BYSECOND=0,0` 会扩增 rrule-go timeset 并重复消耗 COUNT，但 renderer 会排序去重。该风险本次不通过共享解析校验改变调度兼容性。

### 永久空候选

`BYMONTH=2;BYMONTHDAY=30` 永远无 occurrence，但这是依赖库可安全处理的合法耗尽规则。项目不应重复实现日历可达性校验；`NextAfter` 返回零时间。

### 倒置边界

`UNTIL < DTSTART` 的 occurrence 为空，属于依赖库可安全处理的合法耗尽规则，不应由项目重复拒绝。

### YEARLY 文案

YEARLY + BYMONTHDAY + ordinal BYDAY 可出现“每年 每年内……且每年内……”的重复作用域，应保持语义同时简化措辞。

### Set 形状

rrule-go 会静默忽略未知组件和 EXRULE、覆盖重复 RRULE、忽略后续 DTSTART。最终决定不在项目中补充第二套 Set 校验，所有入口直接采用依赖库解析结果。

### 时区冲突

`DTSTART;TZID=Asia/Shanghai:...Z` 会被库解析成 UTC 并丢失 TZID，存在 8 小时偏移。本任务不增加自定义解析拒绝；项目生成端统一生成合法的 `TZID=Asia/Shanghai`，显式 UTC/其他 TZID 仍兼容。

## 有意不支持

- BYYEARDAY、BYWEEKNO、BYEASTER 的中文表达。
- 低于 MONTHLY 频率的 ordinal BYDAY。
- 普通和 ordinal BYDAY 混用。
- EXRULE、多个 RRULE、PERIOD、VALUE=DATE 全天事件、RFC content-line unfolding。

这些场景必须明确报错，不得静默忽略或输出近似文案。

## rrule.js human text 设计借鉴

参考 `jkbrzt/rrule` 的 `src/nlp/totext.ts`、`src/nlp/index.ts`、`src/nlp/i18n.ts` 和 `test/nlp.test.ts`：

- `ToText.IMPLEMENTED` 按 Frequency 声明可完整转换的 option；不支持字段不能无提示遗漏。
- `toString` 先生成频率骨架，再追加 UNTIL/COUNT；各 Frequency 方法负责自己的自然语言顺序。
- BYDAY 会预分类为普通星期、序号星期、工作日集合和一周七天集合，再选择自然文案。
- BYMONTHDAY 正负值统一排序，`-1` 特化为 last；列表连接由统一 helper 负责。
- 语言词典和日期格式器可替换，但语法仍由 renderer 决定，说明中文不能只替换英文 token。
- `origOptions` 用于识别用户是否显式提供字段，`options` 用于生效值；本项目按已确认需求以最终 `rrule.Options` 为可视化语义，不照搬这一点。

本项目采用的迁移结论：保留现有 Go 的 frequency/filter renderer，增加完整可转换门禁和常见星期集合归并；不引入英文中间文本，也不翻译 `toText()` 输出。
