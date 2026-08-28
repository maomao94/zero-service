# 统一 Relay UID 与 Redis Key

## Goal

统一 `app/oryxserver` 中 relay 的唯一身份，避免 Redis state、lease、lock、Sorted Set、内存 `meta`、ffmpeg process ID、Asynq payload 和 stop/reconcile 流程使用不同 key，确保任意入口都能定位并清理同一个 relay。

## Background / Confirmed Facts

- Relay 的业务唯一性是 `ip + port + app + stream`。
- 当前代码存在两套身份：内存 map 与 ffmpeg process 使用 `NormalizeTarget` 产生的 `host:port/app/stream`，Redis key 后缀使用 `CanonicalKey` 产生的 `app/stream`。
- `StopRelayPull` 曾经依赖 `SrsRtmpAddr` 拼接 target，导致 scheme/host 差异时无法读取 state。
- 当前 `RelayState.Target`、`ReconcilePayload.Target`、`StopPayload.Target` 仍叫 target，语义混杂了业务地址和身份。
- `ScanStale` 使用 Sorted Set member 反查 state；因此 member、state key 后缀和可定位的 UID 必须完全一致。
- 现有 Redis 中可能残留 MD5、`rtmp_//host` 和新格式 key；本次以新 UID 格式为唯一新契约，不提供旧 key 双读兼容。

## Requirements

### R1. Canonical UID

- 新增并统一使用 `UID`/`CanonicalUID` 表达 relay 身份。
- UID 的逻辑值为 `ip_port/app/stream`，其中 `:` 不出现在 UID 中，示例：`127.0.0.1_1935/live/drone_xxx`。
- UID 必须同时作为：内存 `meta` map key、ffmpeg process ID、Redis state/lease/lock key 的后缀、relay registry Sorted Set member、Asynq reconcile/stop payload 的身份字段。
- scheme、鉴权 query、source URL 不参与 UID。

### R2. State 业务字段

- `RelayState` 保存 `UID`、`Host/IP`、`Port`、`App`、`Stream`、`Source`、`RelayURL`、deadline 和 retry count。
- `Target` 不再作为 relay 身份字段；需要完整转发地址时使用 `RelayURL`。
- state key 由 UID 唯一生成：`oryx:relay:state:{uid}`；lease、lock 和 Sorted Set 使用同一个 UID。
- state TTL 继续根据 deadline 动态计算，并保留额外兜底时间；不能因固定 TTL 使长于一天的中继提前丢失。

### R3. Start / Stop / Reconcile

- `StartRelay` 从请求的目标地址解析 UID，并以 UID 完成所有本地和 Redis 操作。
- `StopRelayPull` 只使用请求中的 `app`/`stream` 与配置的固定 relay endpoint 解析 UID；不得把完整 RTMP URL 当作唯一 key。
- `StopRelayAndRecording` 复用同一 UID stop 流程；本地进程不存在时仍必须清理 state、lease、Sorted Set member。
- Asynq payload 传 UID，并保留恢复所需的 `Source`、`RelayURL`、`RetryCount` 或等价字段；不再让消费者重新拼接地址推导身份。
- RegistryScanner 取 Sorted Set stale member 后，直接用 UID 读取 state，并在 per-UID 分布式锁内重新读取和判断 lease/pending，再入队。

### R4. Migration / Operational Behavior

- 新代码只生成 UID 格式 key；旧 MD5 和带 scheme key 不做双读兼容。
- 发布前需清理旧的 relay state/lease/registry key，或等待旧 TTL 过期；文档中给出清理方式。
- retry count 仅供运维观察；deadline 到期才停止继续补拉。

## Acceptance Criteria

- [ ] 同一 relay 从 RTMP URL、normalized target 和 `app/stream` stop 输入得到同一个 UID；UID 不包含 `:`、scheme 或 query。
- [ ] `meta` map、ffmpeg Manager、Redis state/lease/lock、Sorted Set member、reconcile/stop payload 使用同一个 UID 值。
- [ ] `StopRelayPull` 不依赖拼接完整 `SrsRtmpAddr` 才能删除 Redis state；即使本地进程不存在，state、lease、Sorted Set member 都被删除或明确返回可重试错误。
- [ ] Scanner 对 stale member 在锁内重新读取 state；不会因旧 state、lease 或 pending 状态造成错误入队。
- [ ] 2 天 deadline 的 state TTL 不早于 deadline，并保留约定的兜底时间；`max_duration_seconds=0` 仍使用默认 1 天。
- [ ] Asynq 任务不会因为 UID 重构重新拼出不同的 target；reconcile 能使用 payload/state 启动相同 relay。
- [ ] 相关单测覆盖 URL/UID 归一化、key 生成、stop 清理、scanner stale 处理和 payload 序列化；`go test ./app/oryxserver/...`、`go vet ./app/oryxserver/...`、`go build ./app/oryxserver/...` 通过。

## Out of Scope

- 不改变 gRPC proto 的业务字段含义，除非现有 payload 无法承载 UID；如需 proto 变更，单独生成并审查兼容影响。
- 不支持多 SRS endpoint 在同一 relay registry 中共享同一 `app/stream` UID；若未来需要，UID 中必须保留 endpoint identity。
- 不为旧 Redis key 增加永久双读、双写或隐式迁移逻辑。

## Open Questions

- None blocking planning. The selected contract is `UID = ip_port/app/stream`, with host port separator `:` replaced by `_`.
