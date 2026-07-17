# Trigger 服务

基于 go-zero 的异步任务调度服务，协议定义见 [`trigger.proto`](../app/trigger/trigger.proto)。

## 两种模式

| 模式 | 基础 | 场景 |
|------|------|------|
| 异步任务调度 | asynq 分布式队列 | 定时/延时回调，一次性任务 |
| 计划任务管理 | 自研数据库扫描引擎 | 周期性巡检，全生命周期管理 |

## 异步任务调度

基于 asynq + Redis。Worker 默认 20 并发，队列权重 `critical:6, default:3, low:1`，支持指数退避自动重试。

### 流程

```
客户端 gRPC --> Redis 队列 --> asynq Worker --> HTTP/gRPC 回调 --> 业务系统
```

`SendTrigger` 发 HTTP POST JSON 回调，`SendProtoTrigger` 发 gRPC Proto 回调。

### API

| 方法 | 说明 |
|------|------|
| `SendTrigger` | 发送 HTTP 回调任务 |
| `SendProtoTrigger` | 发送 gRPC 回调任务 |
| `Queues` | 获取队列列表 |
| `GetQueueInfo` | 获取队列信息 |
| `GetTaskInfo` | 获取任务详情 |
| `ArchiveTask` / `DeleteTask` | 归档/删除任务 |
| `RunTask` | 立即运行任务 |
| `HistoricalStats` | 任务历史统计 |
| `ListActiveTasks` / `ListPendingTasks` / `ListScheduledTasks` / `ListRetryTasks` / `ListArchivedTasks` / `ListCompletedTasks` / `ListAggregatingTasks` | 各状态任务列表 |
| `DeleteAllCompletedTasks` / `DeleteAllArchivedTasks` | 批量删除 |

## 计划任务管理

自研引擎，三级模型：**Plan → Batch → ExecItem**。支持 rrule 表达式定义重复规则，最长跨度 3 年。

```
Plan（计划）
  ├── Batch 1（批次 - 对应一个执行日期）
  │     ├── ExecItem 1（执行单元）
  │     └── ...
  └── ...
```

### 执行流程

1. 创建计划 → 解析 rrule 生成 Batch 和 ExecItem
2. CronService 扫表（有待处理项时 10ms 间隔，空闲时 1-2s 随机）
3. 乐观锁锁定 ExecItem → 通过 gRPC 调用业务系统
4. 业务系统通过 `CallbackPlanExecItem` 回报结果
5. 自动聚合上级状态（ExecItem → Batch → Plan）

### 状态机

```
WAITING(0) ──CronService──> RUNNING(100) ──callback──> COMPLETED(200)
    │                           │
    │                           ├── failed ──> DELAYED(10) ──> RUNNING（重试）
    │                           ├── ongoing ──> RUNNING（保持）
    │                           └── terminated ──> TERMINATED(300)
    │
    └── Pause ──> PAUSED(150) ──> Resume ──> RUNNING
```

| 状态 | 值 | 说明 |
|------|-----|------|
| WAITING | 0 | 等待扫表触发 |
| DELAYED | 10 | 失败延期，等待重试 |
| RUNNING | 100 | 已下发，等待回调 |
| PAUSED | 150 | 人工暂停 |
| COMPLETED | 200 | 终态 |
| TERMINATED | 300 | 终止（超限或人工） |

### 回调结果

| execResult | 状态转移 |
|------------|----------|
| `completed` | RUNNING → COMPLETED |
| `failed` | RUNNING → DELAYED（重试，25 次上限后 → TERMINATED） |
| `delayed` | RUNNING → DELAYED（按 `nextTriggerTime` 重新调度） |
| `ongoing` | RUNNING → RUNNING（等待后续回调） |
| `terminated` | RUNNING → TERMINATED |

重试策略：首次 10s，指数退避最高 30min，Redis 分布式锁防并发。

### API

| 方法 | 说明 |
|------|------|
| `CalcPlanTaskDate` | 预计算执行日期 |
| `CreatePlanTask` | 创建计划任务 |
| `PausePlan` / `ResumePlan` / `TerminatePlan` | 计划级控制 |
| `PausePlanBatch` / `ResumePlanBatch` / `TerminatePlanBatch` | 批次级控制 |
| `PausePlanExecItem` / `ResumePlanExecItem` / `RunPlanExecItem` / `TerminatePlanExecItem` | 执行项级控制 |
| `CallbackPlanExecItem` | 回调执行结果 |
| `GetPlan` / `ListPlans` | 计划查询 |
| `GetPlanBatch` / `ListPlanBatches` | 批次查询 |
| `GetPlanExecItem` / `ListPlanExecItems` | 执行项查询 |
| `GetPlanExecLog` / `ListPlanExecLogs` | 日志查询 |
| `GetExecItemDashboard` | 仪表板统计 |
| `NextId` | 生成自增 ID |

## 配置

```yaml
Name: trigger.rpc
ListenOn: 0.0.0.0:21006
Timeout: 120000

NacosConfig:
  IsRegister: false
  Host: 127.0.0.1
  Port: 8848

Redis:
  Host: 127.0.0.1:6379
  Type: node
  Pass: ""

DB:
  DataSource: "postgres://user:pass@127.0.0.1:5432/dbname?sslmode=disable"

StreamEventConf:          # 回调业务系统的 gRPC 目标
  Endpoints:
    - 127.0.0.1:21009
  NonBlock: true
  Timeout: 120000
```

## 数据模型

### plan

| 字段 | 类型 | 说明 |
|------|------|------|
| plan_id | VARCHAR(64) UNIQUE | 业务唯一 ID |
| plan_name | VARCHAR(128) | 计划名称 |
| type | VARCHAR(64) | 任务类型 |
| recurrence_rule | JSONB | rrule 重复规则 |
| start_time / end_time | TIMESTAMP | 生效时间范围 |
| status | SMALLINT | 1=启用，2=暂停，3=终止 |
| scan_flg | SMALLINT | 扫表完成标记 |

### plan_batch

| 字段 | 类型 | 说明 |
|------|------|------|
| plan_pk | BIGINT | 关联计划 |
| batch_id | VARCHAR(64) UNIQUE | 批次 ID |
| batch_num | VARCHAR(128) UNIQUE | 批次序号 |
| status | SMALLINT | 1=启用，2=暂停，3=终止 |
| plan_trigger_time | TIMESTAMP | 计划触发时间 |

### plan_exec_item

| 字段 | 类型 | 说明 |
|------|------|------|
| exec_id | VARCHAR(64) UNIQUE | 全局执行 ID |
| item_id | VARCHAR(64) | 业务执行项 ID |
| payload | TEXT | 业务负载 |
| next_trigger_time | TIMESTAMP | 下次触发时间（扫表核心字段） |
| trigger_count | INT | 触发次数 |
| status | SMALLINT | 0/10/100/150/200/300 |

> 核心索引：`(is_deleted, next_trigger_time, status)`

### plan_exec_log

| 字段 | 类型 | 说明 |
|------|------|------|
| exec_id / item_id | VARCHAR(64) | 关联执行项 |
| trigger_time | TIMESTAMP | 触发时间 |
| trace_id | VARCHAR(64) | 分布式追踪 ID |
| exec_result | VARCHAR(256) | completed/failed/delayed/ongoing/terminated |
| message / reason | VARCHAR/TEXT | 执行信息 |

## 部署

单机直接运行，集群多节点通过 Redis 共享队列，CronService 乐观锁保证不重复执行。

```bash
cd app/trigger && go run trigger.go -f etc/trigger.yaml
```

## 监控

- OpenTelemetry 分布式追踪（每次执行携带 TraceID）
- `GetExecItemDashboard` 仪表板统计
- 完整 `PlanExecLog` 执行日志

## 参考

- [`trigger.proto`](../app/trigger/trigger.proto) — RPC 接口定义
- [`streamevent.proto`](../facade/streamevent/streamevent.proto) — 业务回调协议

<div align="center">
  <img src="images/trigger-flow.png" alt="Trigger 异步任务回调流程" style="max-width: 80%; height: auto;" />
</div>
