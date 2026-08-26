# Design — oryxserver relay distributed simplify refactor

## 数据模型

```
oryx:relay:state:{app}:{stream} = {"relay_id":"sha256...","source":"...","target":"rtmp://..."}
  ├─ 存在 ⟺ 应该运行
  ├─ StopRelayPull → DEL
  └─ StartRelayPull → 覆盖写（同源幂等）

oryx:relay:lease:{app}:{stream} = "<nodeID>"   TTL 30s
  ├─ 抢：SETNXEX 30s（原子）
  ├─ 续：GET == nodeID ? EXPIRE 30s（两步；竞态无害，见下）
  └─ 放：DEL
```

续租两步竞态分析（替代 Lua 的理由）：
- GET 后租约被 B 抢走 → 我的 EXPIRE 只是给活着的 B 续了几秒，无害；
- GET 后租约被 stop 删除 → EXPIRE 返回 0 / key 不存在 → 我停止本地进程，自愈。
不存在「旧节点写坏新 owner 状态」的路径，因为续租失败的唯一动作是杀自己。

## 状态机

```
Start(source, app, stream, target):
  st = GetState(app, stream)
  if st != nil && st.Source == source && st.Target == target:
      if HasLease: return st.RelayID                    // 幂等
      // else 接管：状态在、没人跑
  else:
      SaveState({RelayID, source, target})              // 新写或替换
  ok = TryClaim(app, stream, nodeID)                    // SETNXEX
  if !ok: return RelayID                                // 别的节点在跑
  if err := Manager.StartPull(...); err != nil:
      Release(app, stream)                              // DEL
      EnqueueReconcile(app, stream)                     // 立即入队
      return "", err

Progress 帧（每 5s，ffmpeg 输出驱动）:
  ok = Renew(app, stream, nodeID)                       // GET 校验 + EXPIRE
  if !ok: Manager.StopPull(app, stream)                 // 失联，杀自己

ffmpeg 异常退出（watcher，非 cancel）:
  Release(app, stream)                                  // DEL
  EnqueueReconcile(app, stream)

Stop(app, stream):
  DeleteState; Release; Manager.StopPull
  未命中本地 && cluster → 既有 MQTT 广播

Reconcile(app, stream)（Asynq 消费，默认重试 25 次指数退避）:
  st = GetState; if nil → ack                            // 已停止/被替换，不复活
  if !TryClaim → ack                                     // 别人在跑
  st2 = GetState; if nil → Release + ack                 // 复查：抢租约期间被 stop
  Manager.StartPull 失败 → Release + return err          // asynq 退避重投
```

## 入队去重

`asynq.TaskID("oryx:relay:"+app+":"+stream)`：同目标重复入队不产生重复任务；
任务归档（重试用尽）后再次入队视为新任务。用尽后不自动恢复，业务重新 Start 拉起。

## 节点崩溃恢复（无扫描器）

节点崩溃 → ffmpeg 随进程死、租约 30s 过期、state 留在 Redis。
业务侧重试 `StartRelayPull` → 读到「同源 + 无租约」→ 本节点接管启动。满足简单中继定位。

## 分层与边界

```
internal/logic/startrelaypulllogic.go ─┐
internal/logic/stoprelaypulllogic.go ──┤ 调 DistRelay.Start/Stop
internal/task/reconcile.go ────────────┘ 调 DistRelay.Reconcile（薄壳）
internal/relay/distributed.go   协调器：状态流转 + 入队 + progress 回调
internal/relay/state.go         Redis Store：原生命令封装
internal/relay/manager.go       本地进程生命周期（done/exited 信号、替换语义）
internal/relay/ffmpeg.go        命令构建 + progress reader + 退出 watcher
```

依赖方向单向：task → svc → relay；relay 不反向依赖。

## 常量

```go
const (
    RelayQueue        = "oryx-relay"
    RelayTaskPrefix   = "oryx:relay:"
    RelayReconcileTask = RelayTaskPrefix + "reconcile"
    leaseTTL          = 30 * time.Second
    progressInterval  = 5 * time.Second   // -stats_period 5
)
```

## 回滚

改动集中在 relay/task/svc 三个内部包，不涉及 proto 与契约；
回滚 = git revert 对应提交即可，Redis key 无迁移负担（旧 key 过期自然消失）。
