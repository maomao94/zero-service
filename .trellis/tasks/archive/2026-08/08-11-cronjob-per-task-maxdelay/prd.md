# PRD: CronJob per-task MaxDelay

## 背景

当前 `CronJobScheduler` 的 `MaxDelay` 通过 `commoncrontask.WithMaxDelay(30*time.Minute)` 硬编码在 `servicecontext.go:111`，对所有周期任务统一生效。当任务延迟超过 30 分钟时跳过本次执行，直接计算下次时间。

缺失能力：
1. 调度器级 `MaxDelay` 不可通过 yaml 配置调整
2. 不支持按任务粒度覆盖最大延迟（某些高频任务期望更短超时，某些低频任务期望更长容忍）

## 需求

### R1: 调度器级 MaxDelay 支持 yaml 配置

- `Config` 新增 `CronTask` 嵌套配置 struct
- `CronTask.MaxDelay` 默认 `30m`
- `NewServiceContext` 读取配置替代硬编码

### R2: TaskConfig 支持 per-task MaxDelay

- `crontask.TaskConfig` 新增 `MaxDelay time.Duration`
- 零值表示使用调度器默认值（保持向后兼容）
- `executeTask` 中若 `task.MaxDelay > 0`，用任务级值覆盖调度器默认值

### R3: CronJob 持久化 MaxDelay

- `gormmodel.CronJob` 新增 `MaxDelay int64`（毫秒），与 `LockTimeout` 对齐
- `convert.go` 双向映射：`fromTaskConfig` / `ToTaskConfig` / `ToProto`
- `CreateCronJobReq` proto 新增 `maxDelay` 字段（毫秒）
- `CronJobPb` proto 新增 `maxDelay` 字段

## 验收标准

### AC1: 调度器级配置生效
- yaml 配置 `MaxDelay: 10m` 后，所有任务延迟超 10 分钟触发 stale skip
- 不配置时默认 30m，行为不变

### AC2: 任务级覆盖生效
- 任务 A: maxDelay=5m，延迟超 5 分钟 skip
- 任务 B: 未设置（0），走调度器默认 30m
- 两任务行为互不影响

### AC3: 创建 & 查询回路完整
- `CreateCronJob` 带 `maxDelay: 300000`（5min）后，`GetCronJob` 返回 `maxDelay: 300000`
- `ListCronJobs` 返回 `maxDelay` 字段

### AC4: 存量兼容
- 未配置 maxDelay 的存量任务行为不变
- 已存在的 `cron_job` 行 `max_delay` 列为 0/NULL，等效于使用调度器默认值

## 非目标
- 不支持 `maxDelay=0` 的显式"无限容忍"语义（保持 0=默认值的约定）
- 不修改 `RunNow`（手动立即执行不判断延迟）
