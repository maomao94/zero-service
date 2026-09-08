# Trigger Plan 规则描述

## Scenario: CalcPlanTaskDate 规则描述

### 1. Scope / Trigger

- `CalcPlanTaskDate` 同时返回日期预览和面向用户的规则描述时适用。

### 2. Signatures

```proto
message CalcPlanTaskDateRes {
  repeated string planDates = 1;
  string scheduleDescription = 2;
  string rruleStr = 3;
}
```

### 3. Contracts

- `scheduleDescription` 必须由展开 `planDates` 的同一个 `rrule.Set` 生成。
- `rruleStr` 返回该 Set 的 RFC 5545 原文，供排障和与持久化快照比对。
- Logic 在完成 DTSTART、RRULE 和 EXDATE 组装后调用 `rrulex.Describe(set.String())`。
- proto 是契约源，修改后执行 `app/trigger/gen.sh`，不得手改生成文件。

### 4. Validation & Error Matrix

- 请求或 RRULE 生成失败 -> 参数错误。
- RRULE 描述失败 -> 参数错误，不返回只有日期而缺少描述的部分响应。
- 规则有效且可描述 -> 同时返回 `planDates` 与非空 `scheduleDescription`。

### 5. Good/Base/Bad Cases

- Good: 每天 09:30 且排除一天 -> 日期列表移除当天，描述包含同一排除时间。
- Base: 未传 start/end -> 使用 Logic 规范化后的本年边界生成日期和描述。
- Bad: 直接从 `PlanRulePb` 拼中文 -> 容易遗漏 DTSTART 默认值、UNTIL 时区和 EXDATE。

### 6. Tests Required

- 断言每天 09:30 的描述、规范化有效期和排除日期。
- 断言 `planDates` 数量与 EXDATE 后结果一致。
- 运行 `app/trigger/gen.sh` 后确认 descriptor 和 Go 生成类型均包含字段 2。

### 7. Wrong vs Correct

#### Wrong

```go
description := describePlanRule(in.Rule)
```

#### Correct

```go
description, err := rrulex.Describe(set.String())
```
