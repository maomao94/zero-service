# aisolo 分布式与持久化部署说明

## 目标

多副本（Kubernetes 多 Pod）部署时，同一 `userId` / `sessionId` 的请求可能落到不同实例。必须保证：

- 会话列表与元数据一致；
- 消息历史可读；
- ADK **Checkpoint**（含 Plan 模式内部状态、中断恢复快照）可跨实例读写。

## 推荐配置组合

| 组件 | 配置项 | 推荐值 |
|------|--------|--------|
| 数据库 | `DB.Enabled` | `true`，并提供 `DataSource` |
| 会话 | `SessionStore.Type` | `gormx`，`BaseDir` 可留空 |
| 消息 | `Memory.Type` | `gormx`（与会话共用 DB） |
| Checkpoint | `Checkpoint.Type` | `gormx`（**Plan 模式必须**） |

单机或小流量可选用 `SessionStore.Type: jsonl` / `Memory.Type: jsonl` / `Checkpoint.Type: jsonl`，**不要在多副本下对会话或 Checkpoint 使用 jsonl**（无跨节点协调）。

## RUNNING 租约

- 进入一轮 Ask/Resume 时，服务会把会话置为 `RUNNING`，并写入 `run_owner`（实例标识）与 `run_lease_until`（默认 `SessionRun.leaseTTL`，见 `aisolo.yaml`）。
- **`run_owner` 默认值**：未配置 `sessionRun.instanceID` 时为 **`主机名` + `:` + 根 yaml 里 `ListenOn` 的 RPC 端口**（与当前 aisolo 进程监听一致，不另读环境变量）。Docker 下每实例一份配置、`ListenOn` 互不冲突即可；极少数同主机名且同端口时再显式设 `sessionRun.instanceID`。
- **启动恢复**（`RecoverRunningSessions`）：  
  - **memory**：仍将所有 `RUNNING` 清为 `IDLE`（单进程开发）。  
  - **gormx / jsonl**：仅当租约已过期，或「无租约且 `updated_at` 早于 `nullLeaseRecoverGrace`」时才清理，避免误伤其它 Pod 上仍在进行的 SSE。

可调配置见根配置 `SessionRun` 与 `SessionStore.nullLeaseRecoverGrace`。

## Plan 模式前置条件

Plan-Execute 依赖 Checkpoint 跨请求持久化。分布式下请使用 **`Checkpoint.Type=gormx`** 且 **`DB.Enabled=true`**，否则多 Pod 间看不到同一计划的中间状态。

Plan 最大迭代次数：`Agent.planMaxIterations`（默认 10）。

## Redis

主数据仍建议落在关系库。Redis 仅作为可选的短时互斥或限流（本仓库未强制接入）；不要把完整 Checkpoint 仅放在无持久化策略的 Redis 中。

## AgentMode 与 Workflow（本版不兼容旧数据）

- `aisolo.proto` 中 `AgentMode` 数值已重排：**Workflow 三枚举固定在 5–7 段末尾**，便于后续追加 `AGENT_MODE_WORKFLOW_*`。
- 若库表 `einox_aisolo_session.mode`（或其它存储）曾按**旧版**整型写入，升级后**不会**自动对应新语义；需自行清空会话表或做离线迁移。**本仓库不提供**旧枚举到新枚举的兼容映射。

## 相关代码

- Solo 网关与 aisolo 共用的 `AgentMode` ↔ HTTP 字符串：`aiapp/aisolo/modeweb`
- 会话存储：`aiapp/aisolo/internal/session`（`memory` / `jsonl` / `gormx`）
- Checkpoint：`common/einox/checkpoint`
- 消息：`common/einox/memory`
- 租约与 Turn：`aiapp/aisolo/internal/turn/executor.go`
