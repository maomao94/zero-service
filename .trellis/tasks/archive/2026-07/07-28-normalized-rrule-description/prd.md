# 使用归一化规则生成 RRULE 描述

## Goal

`DescribeRRule` 仅依据 `rrule-go` 归一化后的生效配置生成中文描述，使可视化结果反映规则最终执行语义，而不区分字段来自用户显式输入还是依赖库默认补齐。

## Requirements

- `descriptionRule` 不再保存或读取 `rule.OrigOptions`。
- 日期筛选、时间集合、`BYSETPOS`、频率推进文案及可描述性校验统一依据 `rule.Options` 和 `DTSTART`。
- 保持 RRULE Set 的持久化、字符串序列化和 occurrence 计算行为不变。
- 对 `rrule-go` 根据 `DTSTART` 自动补齐候选字段的规则，中文描述应采用补齐后的配置。

## Acceptance Criteria

- [x] `descriptionRule` 只保存一份 `rrule.ROption` 生效配置。
- [x] `FREQ=MONTHLY;BYSETPOS=1` 可使用库补齐的月日候选生成准确描述，不再因原始规则未显式声明候选过滤器而拒绝。
- [x] 现有 RRULE 描述测试通过，必要的预期按“最终配置”语义更新。
- [x] `go test ./common/crontask` 和 `git diff --check` 通过。

## Notes

- 本任务是单包内的轻量行为调整，采用 PRD-only。
- `rule.Options` 不公开所有内部预计算字段；现有描述器继续结合 `DTSTART` 处理时间默认值。
- 不修改依赖库自身保留 `OrigOptions` 的行为。
