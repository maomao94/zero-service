# 技术设计

## 边界

本任务修复 `common/crontask` 的中文描述，并补强 Trigger/ISP 生成端的 `Asia/Shanghai` 契约。不实现完整 iCalendar，不修改 `rrule-go`，也不改变调度解析边界。

## 数据流

```text
Trigger / ISP 业务字段
  -> rrule.ROption（Dtstart/Until 使用 Asia/Shanghai）
  -> rrule.Set.String（DTSTART;TZID=Asia/Shanghai）
  -> parseRRuleSet（rrule-go 解析 + DTSTART/RRULE 必需检查）
  -> ValidateRRule / NextAfter
  -> DescribeRRule（中文 renderer）
```

历史或外部完整 Set 可以显式使用 UTC 或其他 TZID；解析结果决定执行和展示时区。项目自身不得生成 floating DTSTART。

## 解析边界

`parseRRuleSet` 必须保持原有简单职责：

- 规范化首尾空白和 CRLF。
- 调用 `rrule.StrToRRuleSet`。
- 检查解析结果包含 DTSTART 和 RRULE。

不得把描述层支持策略、Set 组件白名单、TZID 冲突判断或 ROption 限制加入 `parseRRuleSet`、`ValidateRRule`、`NextAfter`。`DescribeRRule` 直接采用 `parseRRuleSet` 返回的依赖库解析结果，不再解析原始 Set 字符串或维护第二套组件校验。

## 描述安全

- 支持且能无歧义表达的规则继续生成中文。
- 依赖库支持但当前中文无法安全表达的高级组合返回 `ErrUnsupportedDescription`。
- 永久空的 `BYMONTH x BYMONTHDAY` 交集和 `UNTIL < DTSTART` 保持合法耗尽语义；不重复实现依赖库的 occurrence 可达性判断。
- YEARLY 文案只出现一次主频率作用域，子条件使用“各月”“该年内”等局部措辞。
- 借鉴 rrule.js 的 human text 结构，按 Frequency 生成骨架，并对工作日、周末、一周七天、序号星期和负月日做中文语法特化；所有已解析且可表达的条件（包括冗余的一周七天 BYDAY）均显式展示，不通过英文中间文案翻译。
- renderer 读取 `rule.Options` 的最终生效配置；DTSTART 默认日期/时刻用于补齐依赖库未全部回写的默认 clock parts。
- 支持矩阵必须显式覆盖每个 `ROption` 字段，未实现的高级字段返回 `ErrUnsupportedDescription`，不使用“近似”描述。

## 兼容性

- 不改变合法 UTC、合法 TZID、RDATE/EXDATE 和已有基础 RRULE 的 occurrence。
- 描述器新增拒绝不扩散到 `ValidateRRule` / `NextAfter`，避免改变既有调度兼容性。
- floating 历史输入暂时保持 rrule-go 默认 UTC 行为；生成端不再产生此类规则。

## 验证策略

测试以可观察行为为主：频率相位、最终 Options、YEARLY 作用域和星期集合归并同时断言文案与 rrule-go occurrence。生成端解析序列化结果并断言 location 为 `Asia/Shanghai`。

## 回滚

改动集中在描述纯函数和生成端测试。若兼容性出现问题，可回滚 renderer 文案归并；`parseRRuleSet` 不在本任务中变更。
