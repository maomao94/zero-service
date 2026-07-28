# 全面审查 RRULE 中文描述

## Goal

全面审查并修正 `DescribeRRule`，确保它对项目支持的 calendar/RRULE Set 场景输出与 `rrule-go` 实际 occurrence 一致的简体中文文案；无法安全表达的输入必须明确拒绝，不能输出误导描述。

## Requirements

- 覆盖 `YEARLY` 至 `SECONDLY`、`INTERVAL`、`WKST`、`COUNT`、`UNTIL`、全部 BY* 维度、`DTSTART` 默认值、时区/DST、`RDATE`、`EXDATE` 和 Set 形状。
- 可描述规则的日期作用域、频率相位、候选交集、时钟笛卡尔积、`BYSETPOS` 和边界文案必须与 `rrule-go v1.8.2` 的实际 occurrence 一致。
- `BYYEARDAY`、`BYWEEKNO`、`BYEASTER` 等未实现中文表达的高级筛选允许返回 `ErrUnsupportedDescription`；“覆盖全部场景”不等于为所有合法组合生成可能失真的文案。
- 永久无候选及 `UNTIL < DTSTART` 属于合法但已耗尽的规则；描述文案不得承诺一定产生 occurrence。
- 修正文案中重复或错误的 YEARLY 作用域表达。
- `parseRRuleSet` 保持为 `rrule.StrToRRuleSet` 加必需 DTSTART/RRULE 检查的简单包装；本任务不向 `ValidateRRule` 或 `NextAfter` 增加自定义 Set/ROption 校验。
- `DescribeRRule` 直接消费 `parseRRuleSet` 的包内解析结果，不额外解析或校验原始 Set 字符串。
- 项目内规则生成端必须固定使用 `TZID=Asia/Shanghai`；描述与执行入口仍兼容显式 UTC 或其他合法 `TZID` 的历史/外部规则。
- 未标注时区及其他日期时间解析行为沿用依赖库，不由描述器猜测或增加调度路径校验。
- 不扩大为通用 iCalendar 解析器：内容行折叠、`EXRULE`、`PERIOD` 和全天事件不属于当前时刻调度模型。

## Acceptance Criteria

- [ ] 新增场景矩阵测试覆盖全部 Frequency 和每个 ROption 字段，明确标记“正确描述”或“安全拒绝”。
- [ ] `parseRRuleSet`、`ValidateRRule` 和 `NextAfter` 不引入描述层策略或额外自定义校验。
- [ ] YEARLY 组合文案不出现“每年 每年内”一类重复作用域。
- [ ] `MO,TU,WE,TH,FR` 稳定归并为“工作日”，`SA,SU` 归并为“周末”；一周七天筛选归并后仍需显式展示，且 occurrence 差分一致。
- [ ] `DescribeRRule` 直接采用 `rrule-go` 的 Set 解析结果，不重复解析原始字符串或增加组件白名单。
- [ ] Trigger 与 ISP 的规则生成测试断言 `DTSTART;TZID=Asia/Shanghai`，相关 `EXDATE` 同样固定为 `Asia/Shanghai`。
- [ ] 时区、DST fold/gap、RDATE/EXDATE、COUNT/UNTIL 和现有 occurrence 差分测试继续通过。
- [ ] `go test ./common/crontask`、`go test -race ./common/crontask`、`go vet ./common/crontask`、`git diff --check` 通过。

## Notes

- 权威执行语义为项目锁定的 `github.com/teambition/rrule-go v1.8.2`，中文文案必须用实际 occurrence 差分验证。
- renderer 借鉴 `jkbrzt/rrule` 的 `src/nlp/totext.ts` 频率骨架、option 支持门禁和星期集合分类，但直接生成中文，不翻译英文且不引入 JavaScript 依赖。
- floating DATE-TIME 当前由依赖库按 UTC 解析；本任务不改变历史/外部输入的兼容行为，项目生成端通过强制 `TZID=Asia/Shanghai` 避免产生 floating 规则。
- 标准 `VALUE=DATE`、内容行 unfolding 和多 RRULE 不纳入新增支持范围。
