# Implement: CronJob per-task MaxDelay + RunCronJob traceId

## 变更清单（按依赖序）

### 1. Config 层: yaml 可配 MaxDelay

**`app/trigger/internal/config/config.go`**:
- 新增 `CronTaskConfig` 结构体含 `MaxDelay time.Duration`（默认 30m）
- `Config` 新增 `CronTask CronTaskConfig`

**`app/trigger/etc/trigger.yaml`**:
- 新增 `CronTask.MaxDelay: 30m`（可选）

**`app/trigger/internal/svc/servicecontext.go:111`**:
- `WithMaxDelay(30*time.Minute)` → `WithMaxDelay(c.CronTask.MaxDelay)`

### 2. crontask 核心: TaskConfig + executeTask

**`common/crontask/config.go`**:
- `TaskConfig` 新增 `MaxDelay time.Duration`（0=使用调度器默认值）

**`common/crontask/crontask.go`**:
- `executeTask` stale 判断改为 per-task 优先
- `RunNow` 签名改为 `(traceID string, err error)`，返回当前 span 的 trace_id

### 3. Model 层

**`app/trigger/model/gormmodel/cron_job.go`**:
- 新增 `MaxDelay int64`（秒，0=使用调度器默认值）

**`app/trigger/model/gormmodel/cron_exec_log.go`**:
- 新增 `TraceId string`（trace_id 列）

### 4. Convert 层: 双向映射

**`app/trigger/internal/cronjob/convert.go`**:
- `fromTaskConfig`: `MaxDelay: cfg.MaxDelay.Milliseconds() / 1000`（duration → 秒）
- `ToTaskConfig`: `MaxDelay: time.Duration(job.MaxDelay) * time.Second`（秒 → duration）
- `ToProto`: `MaxDelay: int64(cfg.MaxDelay.Seconds())`（duration → 秒）

### 5. Handler 层: trace_id 记录

**`app/trigger/internal/cronjob/handler.go`**:
- `NewLoggingEventHandler` 从 ctx 提取 `traceIDFromContext` 写入 `CronExecLog.TraceId`

### 6. Proto 层

**`app/trigger/trigger.proto`**:
- `CreateCronJobReq`: `int64 maxDelay = 14`（秒）
- `CronJobPb`: `int64 maxDelay = 20`（秒）
- `RunCronJobRes`: `string traceId = 1`

### 7. Logic 层

**`app/trigger/internal/logic/createcronjoblogic.go`**:
- `TaskConfig` 构造加入 `MaxDelay: time.Duration(in.MaxDelay) * time.Second`

**`app/trigger/internal/logic/runcronjoblogic.go`**:
- 使用 `RunNow` 返回的 `traceID` 填充 `RunCronJobRes.TraceId`

## 数据库迁移

```sql
ALTER TABLE cron_job ADD COLUMN max_delay BIGINT NOT NULL DEFAULT 0 COMMENT '最大延迟容忍（秒），0 使用调度器默认值' AFTER lock_timeout;
ALTER TABLE cron_exec_log ADD COLUMN trace_id VARCHAR(64) NOT NULL DEFAULT '' COMMENT '追踪 ID' FIRST;
```

## 影响范围

| 层级 | 变更类型 | 向后兼容 |
|------|---------|---------|
| config.go | 新增字段 | 是 |
| crontask.go | 修改 stale+RunNow | 是 |
| cron_job.go | 新增列 | 是 |
| cron_exec_log.go | 新增列 | 是 |
| convert.go | 新增双向映射 | 是 |
| trigger.proto | 新增字段 | 是 |
| create/run logic | 新增字段传递 | 是 |
