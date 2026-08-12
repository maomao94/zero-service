# Unified RRULE Documentation Design

## Documentation Architecture

- `docs/trigger.md` 保持服务总览，补充三类时间输入及 Plan/CronJob 的不同运行模型。
- 将原 CronJob 指南重命名为 `docs/trigger-rrule-api-guide.md`，保留有效的 CronJob 周期示例，并增加统一 Set 语义、Plan 示例和精确时间场景。
- `docs/README.md` 和所有引用旧指南的链接同步更新。
- Trellis backend Specs 保存可执行的跨层契约，不复制面向用户的大量请求示例。

## Content Flow

1. 先解释 RRULE、`specified_times`、`excluded_times`、`exclude_dates` 如何形成最终 Set。
2. 再解释共享格式、时区、范围和优先级规则。
3. 分别展示 Plan 预览/创建和 CronJob 创建/更新/预览。
4. 最后保留周期规则限制、幂等和运维边界。

## Compatibility

- 文档重命名会同步修改仓库内链接，不保留内容重复的旧文件。
- 不修改外部 API 字段或运行行为。
- 旧数据没有精确时间列表时按空列表说明，原调度语义不变。

## Spec Boundaries

- Trigger Spec：公共编译语义、Plan/CronJob 数据流和限制。
- crontask Spec：调度器只消费完整 Set，RDATE/EXDATE 自然参与首次时间、推进和预览。
- contract-generation Spec：snake_case + `json_name`、列表校验和 protojson 边界。
- GORM Spec：CronJob 可空 JSON 列、配置白名单、清空和在途原子拒绝。

## Risks And Rollback

- 最大风险是文档与最终 CronJob 检查修复后的代码不一致，因此本任务必须在该子任务归档后启动并重新核对 diff。
- 若重命名导致链接检查失败，回滚文件名并保留统一内容；不影响运行代码。
