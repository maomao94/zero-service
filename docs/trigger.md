# Trigger 服务架构

## 服务简介

Trigger 是基于 go-zero 的异步任务调度服务，提供两种核心业务模式：

| 模式 | 实现基础 | 适用场景 |
|------|----------|----------|
| **异步任务调度** | asynq 分布式任务队列 | 定时/延时回调，一次性任务 |
| **计划任务管理** | 自研数据库扫描引擎 | 周期性巡检任务，全生命周期管理 |

协议定义：[`trigger.proto`](../app/trigger/trigger.proto)

## 异步任务调度（基于 asynq）

### 架构

<div align="center">
  <img src="images/trigger-flow.png" alt="Trigger 异步任务回调流程" style="max-width: 80%; height: auto;" />
</div>

### 流程

1. 客户端通过 gRPC 接口发送任务请求（`SendTrigger` / `SendProtoTrigger`）
2. 服务将任务存储到 Redis 队列
3. asynq Worker 从队列取出任务执行
4. 根据任务类型发起回调：
   - `SendTrigger` -- HTTP POST JSON 回调
   - `SendProtoTrigger` -- gRPC Proto 字节码回调
5. 任务状态更新到 Redis，支持查询和管理

### 任务类型

| 内部类型 | 说明 |
|----------|------|
| `defer:triggerTask` | 延迟 HTTP 触发 |
| `defer:triggerProtoTask` | 延迟 gRPC Proto 触发 |
| `scheduler:defer:task` | 定时调度任务 |

### 任务管理 API

| 方法 | 说明 |
|------|------|
| `SendTrigger` | 发送 HTTP POST JSON 回调任务 |
| `SendProtoTrigger` | 发送 gRPC 回调任务 |
| `Queues` | 获取队列列表 |
| `GetQueueInfo` | 获取队列信息 |
| `GetTaskInfo` | 获取任务详情 |
| `ArchiveTask` | 归档任务 |
| `DeleteTask` | 删除任务 |
| `RunTask` | 立即运行任务 |
| `HistoricalStats` | 获取任务历史统计 |
| `ListActiveTasks` | 活跃任务列表 |
| `ListPendingTasks` | 待处理任务列表 |
| `ListScheduledTasks` | 预定任务列表 |
| `ListRetryTasks` | 重试任务列表 |
| `ListArchivedTasks` | 已归档任务列表 |
| `ListCompletedTasks` | 已完成任务列表 |
| `ListAggregatingTasks` | 聚合任务列表 |
| `DeleteAllCompletedTasks` | 删除所有已完成任务 |
| `DeleteAllArchivedTasks` | 删除所有已归档任务 |

### 技术细节

- **并发控制**：asynq Worker 默认 20 并发
- **队列权重**：`critical:6, default:3, low:1`
- **自动重试**：支持指数退避策略
- **存储**：Redis，支持多节点部署

## 计划任务管理（自研引擎）

### 数据模型

采用三级模型：**Plan -> Batch -> ExecItem**

```
Plan（计划）
  ├── Batch 1（批次 - 对应一个执行日期）
  │     ├── ExecItem 1（执行项 - 具体任务单元）
  │     ├── ExecItem 2
  │     └── ...
  ├── Batch 2
  │     ├── ExecItem 1
  │     └── ...
  └── ...
```

- **Plan**：计划任务定义，包含执行规则（rrule 表达式）、时间范围和基本信息
- **Batch**：执行批次，对应一个具体的执行日期
- **ExecItem**：具体的任务执行单元，包含业务负载（payload）和执行状态
- **PlanExecLog**：执行日志，记录每次触发的详细信息

> 限制：计划时间跨度最长 3 年。

### 执行流程

1. **创建计划**：定义执行规则和执行项，系统解析 rrule 计算所有触发日期，生成 Batch 和 ExecItem
2. **CronService 扫表**：定时扫描 `next_trigger_time <= NOW()` 且状态为可执行的 ExecItem
3. **锁定执行项**：乐观锁更新状态为 RUNNING，`next_trigger_time` 后移 5 分钟（防重兜底）
4. **调用业务系统**：通过 StreamEvent gRPC 接口执行具体任务
5. **等待回调**：业务系统处理完成后调用 `CallbackPlanExecItem` 回报结果
6. **状态更新**：根据回调结果更新 ExecItem 状态
7. **状态聚合**：检查批次和计划整体完成度，自动更新上级状态
8. **记录日志**：每次触发写入 PlanExecLog

**扫表频率**：有待处理项时每 10ms 扫描一次，无数据时随机 1-2 秒间隔。

### 状态机

```
                ┌──────────── ResumePlanExecItem ────────────┐
                │                                            │
                v                                            │
WAITING(0) ──CronService──> RUNNING(100) ──callback──> COMPLETED(200)
    │                           │                            
    │                           ├── callback(failed) ──> DELAYED(10) ──CronService──> RUNNING
    │                           │                        （重试，超限→TERMINATED）
    │                           │
    │                           ├── callback(ongoing) ──> RUNNING（保持，等待后续回调）
    │                           │
    │                           ├── callback(terminated) ──> TERMINATED(300)
    │                           │
    │                           └── PausePlanExecItem ──> PAUSED(150)
    │                                                        │
    │                                                  ResumePlanExecItem
    │                                                        │
    └─────────── PausePlanExecItem ──> PAUSED(150) ──────────┘
```

**状态说明**：

| 状态 | 值 | 说明 |
|------|-----|------|
| WAITING | 0 | 初始状态，等待 CronService 扫表触发 |
| DELAYED | 10 | 失败后延期，等待重新触发 |
| RUNNING | 100 | 已下发至业务系统，等待回调 |
| PAUSED | 150 | 人工暂停，不扫表、不触发 |
| COMPLETED | 200 | 执行完成（终态） |
| TERMINATED | 300 | 终止（终态），人工终止或超过重试上限 |

### 回调结果处理

业务系统通过 `CallbackPlanExecItem` 回报执行结果：

| execResult | 状态转移 | 说明 |
|------------|----------|------|
| `completed` | RUNNING -> COMPLETED | 执行成功，终态 |
| `failed` | RUNNING -> DELAYED / TERMINATED | 失败重试，超过上限自动终止 |
| `delayed` | RUNNING -> DELAYED | 业务延期，按 `delayConfig.nextTriggerTime` 重新调度 |
| `ongoing` | RUNNING -> RUNNING | 部分异步，保持 RUNNING 等待后续回调 |
| `terminated` | RUNNING -> TERMINATED | 人工终止，终态 |

### 重试机制

- 首次失败后 10 秒重试
- 指数退避：1s, 2s, 4s, 8s, 16s ... 最高 30 分钟
- 默认最多 25 次重试，超限自动转为 TERMINATED
- 使用 Redis 分布式锁防止并发回调冲突

### 计划任务 API

| 方法 | 说明 |
|------|------|
| `CalcPlanTaskDate` | 预计算计划任务日期 |
| `CreatePlanTask` | 创建计划任务 |
| `PausePlan` / `ResumePlan` / `TerminatePlan` | 计划级别控制 |
| `PausePlanBatch` / `ResumePlanBatch` / `TerminatePlanBatch` | 批次级别控制 |
| `PausePlanExecItem` / `ResumePlanExecItem` / `RunPlanExecItem` / `TerminatePlanExecItem` | 执行项级别控制 |
| `CallbackPlanExecItem` | 回调执行项结果（ongoing 回执） |
| `GetPlan` / `ListPlans` | 计划查询 |
| `GetPlanBatch` / `ListPlanBatches` | 批次查询 |
| `GetPlanExecItem` / `ListPlanExecItems` | 执行项查询 |
| `GetPlanExecLog` / `ListPlanExecLogs` | 执行日志查询 |
| `GetExecItemDashboard` | 执行项仪表板统计 |
| `NextId` | 生成自增 ID |

## 数据库表结构

### plan（计划）

| 字段 | 类型 | 说明 |
|------|------|------|
| plan_id | VARCHAR(64) UNIQUE | 业务唯一 ID |
| plan_name | VARCHAR(128) | 计划名称 |
| type | VARCHAR(64) | 任务类型 |
| group_id | VARCHAR(64) | 分组 ID |
| recurrence_rule | JSONB | rrule 重复规则 |
| start_time / end_time | TIMESTAMP | 规则生效时间范围 |
| status | SMALLINT | 1=启用，2=暂停，3=终止 |
| scan_flg | SMALLINT | 0=未扫表完成，1=已扫表完成 |
| finished_time | TIMESTAMP | 完成时间 |
| ext_1 ~ ext_5 | VARCHAR(256) | 扩展字段 |

### plan_batch（批次）

| 字段 | 类型 | 说明 |
|------|------|------|
| plan_pk | BIGINT | 关联计划主键 |
| batch_id | VARCHAR(64) UNIQUE | 批次唯一 ID |
| batch_num | VARCHAR(128) UNIQUE | 批次序号 |
| status | SMALLINT | 1=启用，2=暂停，3=终止 |
| plan_trigger_time | TIMESTAMP | 计划触发时间 |
| finished_time | TIMESTAMP | 批次完成时间 |

### plan_exec_item（执行项）

| 字段 | 类型 | 说明 |
|------|------|------|
| exec_id | VARCHAR(64) UNIQUE | 全局唯一执行 ID |
| item_id | VARCHAR(64) | 业务执行项 ID |
| item_type / item_name | VARCHAR | 执行项类型和名称 |
| payload | TEXT | 业务负载 |
| request_timeout | INT | 请求超时（毫秒） |
| plan_trigger_time | TIMESTAMP | 计划触发时间 |
| next_trigger_time | TIMESTAMP | 下次触发时间（扫表核心字段） |
| trigger_count | INT | 触发次数 |
| status | SMALLINT | 0/10/100/150/200/300 |
| last_result / last_message / last_reason | VARCHAR/TEXT | 最近一次执行结果 |

**核心扫表索引**：`(del_state, next_trigger_time, status)`

### plan_exec_log（执行日志）

| 字段 | 类型 | 说明 |
|------|------|------|
| exec_id / item_id | VARCHAR(64) | 关联执行项 |
| trigger_time | TIMESTAMP | 触发时间 |
| trace_id | VARCHAR(64) | 分布式追踪 ID |
| exec_result | VARCHAR(256) | completed/failed/delayed/ongoing |
| message / reason | VARCHAR/TEXT | 执行消息和原因 |

## 配置

```yaml
Name: trigger.rpc
ListenOn: 0.0.0.0:21006
Timeout: 120000

# Nacos 服务注册（可选）
NacosConfig:
  IsRegister: false
  Host: 127.0.0.1
  Port: 8848
  Username: nacos
  PassWord: nacos
  NamespaceId: public
  ServiceName: trigger

# Redis 配置（asynq 任务队列 + 分布式锁）
Redis:
  Host: 127.0.0.1:6379
  Type: node
  Pass: ""
RedisDB: 0

# 数据库配置（计划任务存储）
DB:
  DataSource: "postgres://user:pass@127.0.0.1:5432/dbname?sslmode=disable"

# StreamEvent 服务（回调业务系统）
StreamEventConf:
  Endpoints:
    - 127.0.0.1:21009
  NonBlock: true
  Timeout: 120000
```

## 部署

- **单机**：直接运行 `go run trigger.go -f etc/trigger.yaml`
- **Docker**：使用 `app/trigger/Dockerfile` 构建镜像
- **集群**：多节点运行，通过 Redis 共享任务队列，CronService 扫表通过乐观锁 + 分布式锁保证不重复执行

## 监控

- **集成 OpenTelemetry**：每次任务触发携带 TraceID，支持分布式追踪
- **仪表板统计**：`GetExecItemDashboard` 接口提供各计划类型的执行项分布（total/finished/pending）
- **执行日志**：完整的 `PlanExecLog` 记录，便于问题排查

## 相关文档

- [`trigger.proto`](../app/trigger/trigger.proto) -- 完整 RPC 接口定义
- [`streamevent.proto`](../facade/streamevent/streamevent.proto) -- 业务回调协议
