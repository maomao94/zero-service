# oryxserver relay distributed simplify refactor

## Goal

Rewrite the distributed relay layer of oryxserver to a minimal design: durable desired state in Redis, nodeID-only lease, progress-driven renewal, and Asynq default-retry compensation for transient source outages. Remove all over-engineered machinery (Lua CAS, version, lease token, scanner, per-field status bookkeeping).

## Background

- 定位：简单中继 —— 拉源流推送到 Oryx 录制规则流。不做 HA 流媒体平台；固定可靠拉流由业务侧使用成熟流平台，不在本服务范围。
- 核心难点：源流暂时拉不到时需要重试补拉；显式停止不得复活；同目标全集群只有一个 ffmpeg。
- 当前代码（`internal/relay/state.go`、`distributed.go`、`internal/task/reconcile.go`）实现了 Lua CAS + version + lease_token + 进程状态字段，过复杂，需推倒重写。
- 复用既有设施：`common/asynqx`（队列、日志中间件）、`zrpc.RpcServerConf.Redis`（trigger 同款配置）、`ServiceContext` 中已初始化的 `RelayRedis`/`AsynqClient`/`AsynqServer`/`NodeID`。

## Confirmed Decisions

- 不用 Lua 脚本，全部 Redis 原生命令：抢租约 `SetnxEx`（SET NX EX，原子）、续租 `Get` 校验 + `Expire`、释放 `Del`。
- 不用 version、lease_token；租约值就是 nodeID。
- 续租由 ffmpeg `-progress pipe:1` 输出驱动（每帧 `progress=continue` 续一次），进程死则断租。
- 重试用 asynq 默认（25 次、指数退避、30 分钟超时），不无限重试；入队用 `asynq.TaskID` 幂等去重。
- 不做后台扫描器（V1）：节点崩溃后靠业务重试 Start 时接管（state 在、租约过期 → 本节点启动）。
- `Task.done` 保留（进程完全退出信号，区别于 `Ctx.Done`：替换语义必须等旧进程释放 SRS 连接）。

## Requirements

### Redis state（state.go 重写）

- `oryx:relay:state:{app}:{stream}` = `{"relay_id","source","target"}`，存在 ⟺ 应该运行。
- `RelayState` 只保留 `RelayID`、`Source`、`Target` 三个字段；删除 `DesiredState`、`DesiredVersion`、`OwnerID`、`LeaseToken`、`LeaseExpiry`、`ProcessState`、`LastProgressAt`、`Attempt`、`LastError`、时间戳字段。
- `Store` 提供：`GetState`、`SaveState`、`DeleteState`、`HasLease`、`TryClaim`（SetnxEx）、`Renew`（GET 校验 + Expire，返回是否仍持有）、`Release`（Del）。
- 租约 key：`oryx:relay:lease:{app}:{stream}`，值 = nodeID，TTL 30s（常量）。
- 删除：Lua 脚本、`MakeTargetKey`/`ParseTargetKey`/`RelayTarget`、`ScanRunningStates`。

### 协调器（distributed.go 重写）

- `Start(ctx, source, app, stream, target)`：
  - 读 state：同源且租约有效 → 幂等返回已有 relay_id；
  - 同源但租约失效 → 接管：抢租约 → 本地启动；
  - 异源或无 state → 覆盖写 state → 抢租约 → 本地启动；
  - 抢不到租约 → 返回 relay_id（别的节点在跑）；
  - 本地启动失败 → 释放租约 + 入队重试 + 返回错误。
- `Stop(ctx, app, stream)`：删 state → 删 lease → 停本地进程；返回是否本地命中。
- `Reconcile(ctx, app, stream)`：读 state（无 → ack）→ 抢租约（失败 → ack）→ 复查 state（变了 → 释放 + ack）→ 启动；启动失败 → 释放租约 + 返回 error 交给 asynq 退避重投。
- `EnqueueReconcile(ctx, app, stream)`：payload `{"app","stream"}`，`asynq.TaskID("oryx:relay:"+app+":"+stream)`，队列 `RelayQueue`，其余 asynq 默认。
- progress 回调：`OnProgress(app, stream)` → `Renew` 失败 → 停本地进程。

### ffmpeg 进度续租（ffmpeg.go）

- 构建命令追加 `-progress pipe:1 -stats_period 5`；http/https 源追加 `-reconnect 1 -reconnect_streamed 1 -reconnect_delay_max 10`。
- stdout 管道 reader：解析 `key=value` 行，收到 `progress=continue` 帧调用 Manager 的进度回调。
- `watchPullProcess` 在进程退出后停止 reader 路径（进程死自然无输出）。
- 异常退出（`Ctx.Err() == nil` 且 exit err 非 nil）触发 `OnExit` 回调（入队重试）；主动 cancel 不触发。

### 进程管理微调（manager.go）

- `TargetKey` 分隔符 `\x00` → `:`。
- 新增 `OnExit func(app, stream string)`、`OnProgress func(app, stream string)` 两个可选回调字段。
- `Task.done` 保留（命名不变），注释明确「仅进程完全退出后关闭，区别于 Ctx.Done」。

### 任务层（task/reconcile.go 瘦身）

- `ReconcileHandler` 持有 `*svc.ServiceContext`，`ProcessTask` 解 payload → `svcCtx.DistRelay.Reconcile(ctx, app, stream)`。
- payload 平铺 `{"app","stream"}`；删除本地 `generateToken`。

### 装配（servicecontext.go）

- `StateManager` 字段替换为 `Store`；`DistRelay` 注入 `AsynqClient`。
- 接线：`Manager.OnExit` → 入队；`Manager.OnProgress` → `DistRelay.OnProgress`。
- 删除 `RelayStateManager` 相关旧引用。

### 常量（不进配置文件）

- 租约 TTL 30s、progress 间隔 5s、队列名 `oryx-relay`、任务类型 `oryx:relay:reconcile`、key 前缀 `oryx:relay:`。

## Acceptance Criteria

- [ ] `state.go`/`distributed.go`/`reconcile.go` 编译通过，无 Lua 脚本、无 version/lease_token/DesiredState 等旧字段残留。
- [ ] `Start` 幂等：同源同目标有租约返回原 relay_id；同源无租约本节点接管；异源覆盖。
- [ ] 启动失败与 ffmpeg 异常退出入队重试（TaskID 幂等）；主动 cancel 不入队。
- [ ] 重试任务：state 已删 → ack；租约已被占 → ack；启动成功进入 progress 续租。
- [ ] progress 帧驱动 `Expire` 续租；租约丢失（被 stop/接管）→ 本地进程停止。
- [ ] `Stop`：删 state + lease + 本地停止；cluster 模式未命中走既有 MQTT 广播。
- [ ] standalone 与 cluster 模式 `go build ./app/oryxserver/...`、`go vet`、`go test` 通过，`git diff --check` 干净。
- [ ] 无扫描器 goroutine、无定时器续租、无新增配置项。

## Out of Scope

- 后台扫描器、节点崩溃自动接管（业务重试 Start 自愈即可）。
- 无限重试、自定义退避策略。
- ffmpeg 转码、多目标平台、Oryx 录制规则调整。
- 修改 proto / gRPC 契约。
