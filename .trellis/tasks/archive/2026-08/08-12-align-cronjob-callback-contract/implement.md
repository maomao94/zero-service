# Implementation Plan

1. 修改 Trigger Proto 的 Create/Submit/List TaskCode 最大长度为 128，并调整 CronJob GORM 模型列宽（已完成并提交）。
2. 覆盖 StreamEvent `HandleCronJobEventReq` 为扁平关键业务字段，删除 `extra`，字段号从 1 连续对齐；不导入 `validate/validate.proto`，不引入嵌套模型。
3. 执行 `app/trigger/gen.sh` 和 `facade/streamevent/gen.sh`，审查生成 diff，确认没有无关插件噪声。
4. 增加 Trigger 请求边界（128/129 task code）与模型 schema（size:128、唯一索引）测试，强化 handler 回调字段断言。
5. 运行 Trigger、StreamEvent 相关测试与 vet，最后运行 `git diff --check`。

## Validation

- `go test ./app/trigger/trigger ./app/trigger/internal/cronjob ./app/trigger/model/gormmodel -count=1`
- `go test ./facade/streamevent/... -count=1`
- `go vet ./app/trigger/internal/cronjob ./app/trigger/model/gormmodel ./facade/streamevent/...`
- `git diff --check`

## Risky Files

- `facade/streamevent/streamevent.proto`: 字段号顺延对齐后必须重新生成，确认生成物与源 Proto 一致。
- `app/trigger/model/gormmodel/cron_job.go`: 模型声明不会自动替代生产 DDL。

## Pre-Start Checks

- 确认两个 `gen.sh` 在当前工具链可运行。
- 确认生成文件只来自源 Proto。
- 回执枚举和 RPC 方法不变；请求 PB 与历史 `extra` 字段允许直接覆盖，不保留字段号兼容。
