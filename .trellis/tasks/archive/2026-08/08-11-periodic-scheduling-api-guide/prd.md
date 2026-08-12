# Document periodic scheduling API scenarios

## Goal

为 Trigger CronJob 对接方提供一份可直接使用的中文 API 指南，说明如何创建常见周期任务、限定生效区间、创建固定时间单次任务、创建后立即补触发一次，以及对已经创建好的任务人工执行一次。

## Background

- `CreateCronJob` 使用 `PlanRulePb` 编译完整 RRULE Set，创建成功后返回 `job_id` 和 `next_run`。
- `PlanRulePb.freq` 取值为 `0-YEARLY`、`1-MONTHLY`、`2-WEEKLY`、`3-DAILY`、`4-HOURLY`、`5-MINUTELY`。
- `hours` 与 `minutes` 都至少需要一个元素，秒数由服务端固定为 0。
- `start_time` 与 `end_time` 定义规则生效区间，跨度不能超过 3 年，使用 `Asia/Shanghai` 时区。
- `start_time` 与 `end_time` 都是可选字符串，省略或传空字符串均会进入默认补齐逻辑：
  - 两者都为空：开始时间为创建时所在年份的 1 月 1 日 00:00:00，结束时间为同年 12 月 31 日 23:59:59。
  - 只传 `start_time`：结束时间为该开始时间所在年份的 12 月 31 日 23:59:59。
  - 只传 `end_time`：开始时间仍为创建时所在年份的 1 月 1 日 00:00:00；如果结束时间早于开始时间则请求失败。
  - 两者相同是合法的，可配合规则形成固定时间单次任务；实现只拒绝 `end_time < start_time`。
- 默认区间以 `Asia/Shanghai` 当前时间计算并写入编译后的 RRULE；管理字段仍保留调用方原始输入，因此空字符串不会被回填成默认日期。生产配置建议显式传入开始和结束时间。
- 当前请求没有 `interval` 字段，因此“每三个月”只能使用固定月份集合表达，例如每年 1、4、7、10 月执行；不能表达从任意开始月份起滚动每隔三个月。
- `CreateCronJob.skip_time_filter=true` 时，首次 `next_run` 最多选择一个已经发生的计划点，使新创建的任务尽快被调度；它不会追赶全部历史周期，后续仍按原 RRULE 周期执行。
- `RunCronJob` 只用于已经创建好的 CronJob，通过 `job_id` 人工异步执行一次，不用于创建任务，也不修改原周期计划。

## Requirements

- 新增的独立文档现为 `docs/trigger-rrule-api-guide.md`，最初定位为 CronJob 创建和执行场景指南。
- 说明 `CreateCronJob`、`SubmitCronJob` 与 `RunCronJob` 的使用边界；周期示例以 `CreateCronJob` 请求体为主。
- 提供 `PlanRulePb` 频率值、月份、月日、星期、小时、分钟和生效区间的字段说明。
- 单独提供“开始与结束时间”章节，使用表格说明必填性、格式、时区、四种传值组合、默认值、最大 3 年跨度和结束时间不能早于开始时间。
- 提供可独立理解的完整 JSON 示例，至少覆盖：
  - 每分钟第 0 秒执行一次。
  - 每 10 分钟第 0 秒执行一次。
  - 每小时整点执行一次。
  - 每年固定季度月份（1、4、7、10 月）1 日 09:00 执行，作为当前 API 的“每三个月”示例。
  - 每月 1 日 09:00 执行一次。
  - 每周一、周三、周五 09:00 执行一次。
  - 在指定开始和结束日期区间内每天 09:00 执行。
  - 在固定时间只执行一次。
  - 使用 `CreateCronJob` 下发创建后立即补触发一次的周期任务，提供完整请求 JSON，并设置 `skip_time_filter: true`。
  - 对已经创建好的任务使用 `RunCronJob` 人工异步执行一次，提供 `job_id` 请求和 `trace_id` 响应 JSON。
- 说明 `exclude_dates`、`lock_timeout`、`max_delay`、`skip_time_filter`、`payload`、`extra` 的关键语义和单位。
- 所有完整创建示例显式填写 `start_time` 与 `end_time`；另给出省略字段时的默认行为，避免示例依赖调用当年的隐式区间。
- 明确“创建后立即执行”不是调用 `RunCronJob`，而是创建请求设置 `skip_time_filter: true`。
- 明确立即补触发最多选择一个已发生的计划点，不追赶全部历史周期；创建后任务仍按原周期继续执行。
- 明确 `RunCronJob` 必须基于已经创建好的任务和 `job_id`，不会创建新任务，也不会修改周期 `next_run`、启停状态或 `last_scheduled_run`。
- 明确创建成功只表示任务已持久化并返回首次 `next_run`，不表示异步业务回调已经成功；下游仍需幂等。
- 在 `docs/trigger.md` 的 CronJob 部分和 `docs/README.md` 文档索引中增加相对链接。
- 基于新增指南回看并润色 `docs/trigger.md`：优化 Trigger 服务能力介绍、CronJob 适用场景、创建/提交/立即执行 API 说明和专项指南导航，保持总览与操作指南职责清晰。
- `docs/trigger.md` 保留异步任务、Plan 和 CronJob 的服务总览，不复制专项指南中的全部 JSON；完整请求示例只放在 CronJob API 指南中。
- `docs/README.md` 同时保留 Trigger 服务总览入口，并新增 CronJob API 场景指南入口及准确摘要。
- 示例不得包含真实凭据、内网地址或个人路径。

## Acceptance Criteria

- [ ] 对接方可以仅根据新文档构造上述全部创建或立即执行请求。
- [ ] 每个示例都符合 `trigger.proto` 的字段校验，包括非空 `hours`、`minutes` 和 `dept_code`。
- [ ] 文档准确说明 `start_time`、`end_time` 可为空及所有默认组合，并明确采用 `yyyy-MM-dd HH:mm:ss` 和 `Asia/Shanghai`。
- [ ] 文档明确规则区间允许 `start_time == end_time`、拒绝 `end_time < start_time`，且跨度不能超过 3 年。
- [ ] 文档区分编译后的默认区间与管理视图保留的原始空字符串，生产示例显式配置区间。
- [ ] “每三个月”被准确描述为固定月份集合，不宣称当前 API 支持通用间隔字段。
- [ ] 固定时间单次任务使用同一秒的 `start_time` 和 `end_time`，并说明该有界规则只有一个候选执行点。
- [ ] “创建后立即执行”章节包含完整 `CreateCronJob` 请求 JSON，使用 `skip_time_filter: true`，并说明最多补一个历史计划点且后续周期不变。
- [ ] “已有任务立即执行”章节包含 `RunCronJob` 请求和响应 JSON，并说明它以已有 `job_id` 为前提且不改变周期计划。
- [ ] 新文档从 `docs/README.md` 和 `docs/trigger.md` 可达，Markdown 相对链接有效。
- [ ] `docs/trigger.md` 的服务介绍和 CronJob API 文本已结合专项指南完成润色，能力边界准确且没有大段重复 JSON。
- [ ] `docs/README.md` 能分别导航到 Trigger 服务总览和 CronJob API 场景指南。
- [ ] `git diff --check` 通过，文档中不存在 TBD 或示例敏感信息。

## Out Of Scope

- 不修改 Trigger Proto、调度逻辑、校验规则或生成代码。
- 不新增通用 RRULE `interval` 能力。
- 不编写 Plan/Batch/ExecItem 或 asynq 异步任务的完整使用手册。
- 不重写 `docs/trigger.md` 中与本任务无关的 Plan 状态机、异步任务实现、部署和监控章节。
- 不提供语言 SDK 示例、鉴权配置或部署说明。
