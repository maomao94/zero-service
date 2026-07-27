# CronJob 管理接口实施计划

## Execution Order

- [x] 更新 `app/trigger/trigger.proto`，追加 `RunCronJob`、`GetCronJob`、`ListCronJobs`、`CronJobPb` 及请求响应消息。
- [x] 执行 `app/trigger/gen.sh`，检查 pb、validate、gRPC server、Logic 模板、descriptor 和 Swagger 生成差异。
- [x] 在 `internal/cronjob` 增加统一的模型、`TaskConfig`、Proto 转换，并补 DBStore 测试。
- [x] 实现三个 Logic，接入 Scheduler/Store，统一 NotFound 与查询错误映射。
- [x] 扩展 CronJob Logic 生命周期测试，覆盖立即执行、详情、列表、空时间、禁用/耗尽和软删除。
- [x] 执行质量检查并根据结果修正；更新 crontask spec 中的管理 RPC 契约。
- [x] 按评审意见在公共 `TaskStore` 增加 `GetByID`，合并 Trigger 重复查询，并让详情/列表统一消费 `TaskConfig`。
- [x] 保留 `CronJobPb.createTime/updateTime`，通过 `TaskConfig` 传递审计时间；列表分页筛选下沉到 Logic 后回归 common、ISP 与 Trigger。
- [x] 统一 Trigger 分页参数范围，修复 Asynq 零值 option，并用 `gormx.PageParams` 消除 Offset 溢出和提前 `int` 转换。

## Validation

```bash
/usr/bin/env bash app/trigger/gen.sh
go test -count=1 ./app/trigger/internal/cronjob ./app/trigger/internal/logic ./app/trigger/internal/server
go test -race -count=1 ./common/crontask ./app/ispagent/internal/crontask ./app/trigger/internal/cronjob ./app/trigger/internal/logic
go test -count=1 ./app/ispagent/...
go test -count=1 ./app/trigger/...
go vet ./common/crontask ./app/ispagent/... ./app/trigger/...
go build ./app/ispagent/... ./app/trigger/...
git diff --check
```

## Risk And Rollback Points

- 先检查 proto codegen diff；若工具版本造成无关大范围改写，暂停并定位生成器差异。
- 分页必须在同一筛选 query 上 count 和 find，且排序必须包含唯一 `id` 作为次序键。
- 立即执行不得走周期 `Complete` 路径，也不得写 `status/next_run`。
- 持久化 JSON 转换错误必须返回，不能跳过坏记录导致 `total` 与列表长度无从解释。
- 公共 `TaskStore` 增加方法后必须同步所有实现和测试替身，并扩大验证到 common/crontask 与 ISP crontask。
- 公共分页参数保持 `int64` 到 `gormx` 边界；不得在 Logic 提前转 `int`，极大页码必须返回空页而不是第一页。

## Validation Results

- `go test -count=10 ./app/trigger/internal/cronjob ./app/trigger/internal/logic` 通过。
- `go test -race -count=1 ./app/trigger/internal/cronjob ./app/trigger/internal/logic` 通过。
- `go test -count=1 ./app/trigger/...` 通过；既有 invoke 测试需要在允许监听本地回环端口的环境运行。
- `go test -count=1 ./common/crontask ./app/ispagent/internal/crontask ./app/trigger/internal/cronjob ./app/trigger/internal/logic ./app/trigger/internal/server` 通过。
- `go test -count=1 ./app/ispagent/...` 通过。
- `go test -race -count=1 ./common/crontask ./app/ispagent/internal/crontask ./app/trigger/internal/cronjob ./app/trigger/internal/logic` 通过。
- `go vet ./common/crontask ./app/ispagent/... ./app/trigger/...` 通过。
- `go build ./app/ispagent/... ./app/trigger/...` 通过。
- `git diff --check` 通过。
- `go test -count=1 ./common/gormx` 与 `go test -race -count=1 ./common/gormx` 通过。
- `go test -count=1 ./app/djicloud/... ./app/bridgemodbus/... ./app/ieccaller/...` 通过。
- `go build ./...` 通过；File 与 Trigger 中需要回环端口的测试在允许监听的环境通过。
- 生成器版本与仓库现有生成文件头一致；大范围生成 diff 来自上一提交调整 Proto 消息顺序后未同步生成文件，本次已按当前契约源重新生成。
