# Trigger RRULE Cron Job Implementation

## Execution Order

- [x] 更新 `common/crontask`：删除 Version，引入 TaskClaim/Complete/UpdateLastRun，增加 ErrDeleteTask，修正 Scheduler 与 RunNow。
- [x] 同步 MemoryStore、ISP DBStore、转换代码和相关单测，先通过 crontask 与 ISP 定向验证。
- [x] 在 Trigger 新增 CronJob/CronJobExtra 模型、转换、RRULE 规则编译和 DBStore，并覆盖 SQL NULL、claim CAS、业务 Extra 与时间计算测试。
- [x] 更新 Trigger Proto，执行 `app/trigger/gen.sh`，实现创建、启用、禁用、删除 Logic，并接入 ServiceContext 与服务生命周期。
- [x] 更新 Eventstream Proto，执行 `facade/streamevent/gen.sh`，实现最小 SUCCESS 回执 Logic。
- [x] 实现 Trigger 调度 Handler：构造稳定 scheduledTime 请求，映射 SUCCESS、UNKNOWN/error 和 TASK_NOT_FOUND。
- [x] 更新开发/测试 AutoMigrate，检查生成文件、Swagger 和服务注册 diff。
- [x] 增加任务级 `LockTimeout`，同步 Trigger 创建、ISP 查询、两侧 GORM 模型和 Store claim 行为；ISP `101-1` 不包含该字段且更新时保留原值，未配置时回退 Scheduler 默认值。
- [x] 运行质量验证，按发现的问题迭代；更新 crontask Trellis spec 后提交和收尾。

## Validation

```bash
go test -count=1 ./common/crontask ./app/ispagent/internal/crontask
go test -race -count=1 ./common/crontask ./app/ispagent/internal/crontask
go test -count=1 ./app/trigger/... ./facade/streamevent/...
go test -race -count=1 ./app/trigger/internal/cronjob
go vet ./common/crontask ./app/ispagent/internal/crontask ./app/trigger/...
go build ./...
git diff --check
```

## Risk And Review Gates

- 公共 TaskStore 接口修改后立即运行 common/ISP 测试，避免把接口回归带入 Trigger 开发。
- Proto 只从源文件生成；若 codegen 产生无关大范围变更，停止并检查工具版本，不手工修生成文件。
- claim 和 complete SQL 必须检查 `RowsAffected`；测试覆盖双实例竞争和配置/状态并发。
- 业务字段以平铺列为真源；转换测试必须防止 caller Extra、Rule 和 ExcludeDates 丢失。
- 最终不以单测通过替代 race、vet 和全仓 build。

## Validation Results

- 任务级锁超时覆盖 Scheduler 默认值、0 值回退默认值、ISP 配置更新保留已有值、两侧 GORM 往返和 Trigger 创建映射测试通过。
- Trigger、ISP、Eventstream 和 `common/crontask` 相关模块单测通过；需要监听端口的 Trigger 既有测试已在沙箱外通过。
- `common/crontask`、ISP DBStore、Trigger CronJob DBStore 的 race 通过。
- 相关包 `go vet`、`go build ./...`、两套 Proto `gen.sh` 和 `git diff --check` 通过。
- `go vet ./...` 仍有两处任务外既有告警：`app/djicloud/model/gormmodel/dji_fly_region.go` 的 struct tag，以及 `app/iecagent/internal/iec/iechandler.go` 的 unkeyed struct literal。
