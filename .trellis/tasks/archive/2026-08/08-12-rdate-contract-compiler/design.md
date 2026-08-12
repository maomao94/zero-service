# Shared Exact-Time Contract Design

## Scope

本子任务只建立 Trigger 公共请求字段和共享 RRULE Set 编译能力。Plan Logic、CronJob Logic、数据库模型与用户文档由后续子任务负责。

## Contract

相关请求新增 `specified_times` 与 `excluded_times`，JSON 名称分别为 `specifiedTimes`、`excludedTimes`。每项是 `Asia/Shanghai` 下的 `yyyy-MM-dd HH:mm:ss`，每个列表最多 1000 项。

`specified_times` 编译为 RFC 5545 RDATE，`excluded_times` 编译为精确 EXDATE。现有 `exclude_dates` 继续表示整日排除。

## Compiler

共享编译函数在规范化 Plan 3 年或 CronJob 100 年范围后解析精确时间。所有值必须位于闭区间 `[start_time, end_time]`。最终 Set 语义为 `RRULE ∪ RDATE - EXDATE`；排除日期还要为当天所有指定时间写入 EXDATE。

## Compatibility

新增 repeated 字段使用各消息未占用字段号。旧请求不传字段时行为不变。生成代码只通过 `app/trigger/gen.sh` 更新。

## Validation

测试覆盖空列表、1000/1001 边界、格式和范围错误、边界命中、并集去重、精确排除及整日排除。
