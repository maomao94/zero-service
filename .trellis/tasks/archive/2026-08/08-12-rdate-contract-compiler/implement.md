# Implementation Plan

1. 修改 Trigger Proto 中共享 Plan/CronJob 请求与 CronJob 管理视图字段，配置每列表最多 1000 项和 19 字符元素校验。
2. 运行 `app/trigger/gen.sh`，审查生成字段名、JSON 名和无关生成噪声。
3. 扩展共享 Schedule 编译签名与实现，解析并校验 `specified_times` / `excluded_times`。
4. 保持 `exclude_dates` 整日排除，并覆盖当天指定时间。
5. 补充 Schedule 和 Set 行为测试，不接入业务 Logic 或数据库。
6. 运行目标测试、生成一致性检查和 `git diff --check`。

## Rollback

- Proto 字段、生成物、编译参数和测试作为一个单元回滚。
- 不修改 rrule-go 或公共 crontask 解析器。
