# 维护 Trigger CronJob 调度文档

## Goal

基于当前 `app/trigger/internal/cronjob` 实现，重新维护 Trigger 相关项目文档，让使用者从项目首页、文档索引和服务专题都能区分三类调度能力，并正确理解 CronJob 的创建、执行、回执和生命周期语义。

## Confirmed Facts

- Trigger 当前包含三类独立能力：asynq 异步任务、`Plan -> Batch -> ExecItem` 计划任务、基于 `common/crontask` 的 RRULE CronJob。
- CronJob 将业务规则编译为 RFC 5545 RRULE，持久化到 `cron_job` 表，由 `CronJobScheduler` 扫描到期任务。
- CronJob 到点后通过 gRPC `HandleCronJobEvent` 回调 Eventstream；成功回执推进周期，任务不存在回执会请求删除，其他错误由调度器重试。
- `RunCronJob` 是异步人工执行，不改变周期 `next_run` 或启停状态。
- CronJob 使用启用/禁用两态，`next_run = NULL` 表示规则已耗尽；它不使用 Plan/Batch/ExecItem 状态机。

## Requirements

- 将文档能力概览由“两种模式”调整为三类调度能力，并明确各自基础设施、适用场景和边界。
- 新增独立 CronJob 章节，覆盖规则编译、调度链路、回执行为、生命周期 API、核心时间字段和数据模型。
- 同步根目录 `README.md`、`docs/README.md`、`docs/quick-start.md`、`docs/service-ports.md` 与 `docs/trigger.md` 中直接相关的能力概览、导航、依赖和服务说明。
- `docs/architecture.md` 仅保留现有服务结构图，不为没有过时内容的文档制造修改；Plan openGauss 迁移指南继续保持 Plan 专题边界。
- 所有描述必须以当前 proto、`internal/cronjob`、`common/crontask` 和服务装配代码为依据，不承诺未经实现或验证的 Exactly Once、跨节点可靠性等能力。
- 仅修改项目文档和本任务 Trellis 产物，不修改业务代码、协议或生成文件。

## Acceptance Criteria

- [x] `docs/trigger.md` 明确区分异步任务、计划任务和 CronJob 三类能力。
- [x] 项目首页、文档索引、快速开始和端口清单中的 Trigger 描述同步包含 CronJob，且仍保持入口文档的简洁性。
- [x] CronJob 文档准确描述 RRULE/EXDATE、默认有效期、最长三年跨度、`skipTimeFilter`、人工立即执行和 Eventstream 回执语义。
- [x] CronJob API 与 `trigger.proto` 当前暴露的方法一致。
- [x] `cron_job` 核心字段及 `next_run`、`scheduled_time`、`last_run`、`last_scheduled_run` 的用途描述与模型和 Store 一致。
- [x] 文档中的相对链接有效，`git diff --check` 通过，最终差异仅包含 Markdown 文档与 Trellis 任务产物，不包含业务代码变更。

## Out of Scope

- 修改 CronJob 实现、数据库模型、RPC 契约或回调协议。
- 为三类调度提供性能、HA、Exactly Once 或故障转移承诺。
- 重写其他 Trigger 专题文档。

## Open Questions

无。当前仓库能够回答本次文档维护所需的行为与范围问题。
