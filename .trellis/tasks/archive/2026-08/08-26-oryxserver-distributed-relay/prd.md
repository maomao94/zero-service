# oryxserver distributed relay and compensation

## Goal

Move FFmpeg relay ownership from process-local memory to a Redis-coordinated distributed model without introducing a separate scheduler. The gRPC node first records the desired relay and immediately tries to start FFmpeg locally; only a failed start or later process failure enters the shared Asynq compensation path. A target must have at most one valid lease and one active FFmpeg process across the cluster, while an explicitly stopped relay must never be recreated.

## Child Tasks

- `08-26-oryxserver-relay-infra`: introduce Redis/Asynq runtime wiring and isolated relay queue/namespace. Must be completed first.
- `08-26-oryxserver-relay-execution`: implement distributed relay state, fencing, direct local start, progress-triggered lease renewal, and compensation. Depends on the infrastructure child.

## Background and Constraints

- Existing local relay management is in `app/oryxserver/internal/relay/manager.go` and `ffmpeg.go`; it allows one process per `(app, stream)` only within one node.
- `relay_id` is deterministic from canonical source URL + app + stream. The durable distributed key is `(app, stream)`; process-local completion signaling is not a distributed state signal.
- `oryxserver` already has optional cluster mode, MQTT broadcast, Nacos instance metadata, go-zero Redis support elsewhere, and Asynq v0.26.0 in `go.mod`.
- Redis is the source of truth for desired state and ownership. A local FFmpeg process is only an execution lease holder. There is no fixed owner-selection algorithm: the gRPC node tries first, then any node that consumes a pending Asynq task may atomically acquire the target lease.
- `StopRelayPull` identifies a relay by `(app, stream)`. Stop must be durable and must prevent retry or failover compensation from recreating it.
- Distributed mode must be optional so existing standalone deployments continue to work without Redis or Asynq services.

## Requirements

### Durable relay state and simple distributed ownership

- Store one durable record per target containing `relay_id`, source URL, target fields, desired state (`running` or `stopped`), owner instance ID, lease version/token, lease expiry, attempt number, last error, and timestamps.
- Use atomic Redis transitions to create/replace a start request and to claim, renew, and release ownership. A node may start FFmpeg only while it owns a valid lease for the current record/version.
- Use a fencing token/version in every ownership transition so a delayed old node cannot delete, renew, or report success for a newer owner.
- Lease renewal must stop when local FFmpeg exits or the node loses Redis connectivity beyond the safety window. Redis errors must fail closed: do not start a new process without a confirmed lease.

### Distributed start, replacement, and stop

- `StartRelayPull` must be idempotent for the same canonical source and target, and replace a different source targeting the same `(app, stream)` through a durable version transition.
- The request durably records desired running state, then immediately tries to start FFmpeg in the current gRPC service. The existing API remains non-blocking and must not claim FFmpeg is connected.
- If local FFmpeg construction or `cmd.Start()` fails, enqueue one durable Asynq reconciliation task for the current relay/version. The task remains pending when no worker is online and is consumed when any node becomes available; worker errors use bounded Asynq retry/backoff.
- If local start succeeds, claim the Redis lease before or as part of starting the process. Ownership coordination must never allow two valid owners for one target.
- `StopRelayPull` must atomically set desired state to `stopped`, invalidate the current lease, cancel the local process if owned locally, and enqueue cleanup/reconciliation as needed. Stale tasks and node recovery must not resurrect it.
- Stop and replacement operations must use a monotonically increasing record version or equivalent fence. An old stop must not stop a newer start and an old start must not overwrite a newer stop.

### Compensation and Asynq

- A failed local start and every unexpected FFmpeg exit enqueue reconciliation work for the current relay/version; no node-to-node RPC is required.
- On service startup, scan durable desired-running records and enqueue reconciliation work for records with no valid lease or stale owner. This is the recovery path when the original gRPC node was offline before it could enqueue a retry.
- Reconcile running records whose lease expires, owner heartbeat becomes stale, or FFmpeg exits unexpectedly. Compensation is bounded by configurable retry/backoff policy and records the last failure.
- Use Asynq for durable reconciliation/retry work with a stable task type and unique task key per relay/version, retention, retry limit, and exponential/backoff delay. Duplicate tasks must be harmless.
- A compensation worker must re-read Redis and perform lease/fence checks before starting FFmpeg. Retryable Redis/process failures return errors; stopped or superseded records are acknowledged without retry loops.
- Recovery must cover normal shutdown (release leases and enqueue reconciliation) and crashed-node takeover after lease expiry. A new node consuming the shared queue must continue a desired running relay with the same source/target.
- Compensation must not create duplicate FFmpeg processes on one target, including when an old process exits slowly. Process-exit completion signaling remains separate from cancellation-request signaling.

### FFmpeg liveness signal

- FFmpeg does not provide a business-level relay heartbeat that proves the remote SRS/Oryx publisher is healthy. Treat `Cmd.Wait()`/process liveness as the hard stop signal.
- Add a machine-readable FFmpeg progress stream using `-progress pipe:1` with a bounded report interval. A watcher in the same service reads progress output from the running FFmpeg process and uses each valid progress event to renew the Redis lease key, guarded by the current owner instance and fencing version. The FFmpeg process itself does not connect to Redis.
- The Redis key must be renewed by process-triggered progress events, not by an unrelated timer that could keep a dead process alive. The watcher may use a small bounded write timeout, but it must not renew after `Cmd.Wait()` returns or after the local context is cancelled.
- Record the last progress timestamp locally and, while the lease is valid, in Redis. This detects a hung process or no input/output progress but is advisory, not proof of remote delivery.
- Do not use human-readable stderr `-stats` output as the distributed heartbeat. The heartbeat is valid only while the process has not exited and the node still owns the current lease/version.
- If progress renewal stops beyond the configured threshold, the owner must cancel and wait for the process to exit before enqueueing compensation. If the Redis lease expires first, the old process must be cancelled locally as soon as ownership loss is detected and must never renew again. A replacement node must acquire a new fence before starting.

### Observability and operations

- Log and expose owner, lease version, desired state, process state, attempts, last error, and compensation reason without logging source credentials.
- Add configuration for distributed enablement, Redis connection/database/timeouts, node identity, lease duration/renew interval, reconciliation interval, retry limit, and backoff bounds. Defaults preserve standalone behavior when distributed mode is off.
- Startup reconciliation must claim leases through the same fencing path rather than blindly starting every record. Shutdown stops local FFmpeg, releases only matching leases, and leaves desired running records eligible for takeover.

## Acceptance Criteria

- [ ] Two or more instances sharing Redis cannot both hold a valid lease or run FFmpeg for the same `(app, stream)`; concurrency tests cover repeated start/replace/claim races.
- [ ] Start persists desired running state and returns deterministic `relay_id`; repeating it is idempotent, while a different source creates a newer fenced version and replaces the previous relay.
- [ ] Stop persists desired stopped state and no delayed retry, stale worker, restart, or lease takeover starts it again.
- [ ] Failed local start or owner process exit enqueues a durable Asynq task; when all workers are offline the task remains pending, and a later node consumes it without losing the desired relay.
- [ ] Stale progress, lease expiry, or simulated node crash is compensated by another node after the safety interval without duplicate relay processes.
- [ ] Redis loss or uncertain lease renewal prevents new FFmpeg starts and an old owner cannot continue after fencing loss.
- [ ] Asynq tasks are deduplicated by relay/version, retries are bounded with backoff, and terminal stopped/superseded records do not loop.
- [ ] Normal shutdown releases only the local matching lease and leaves desired running records recoverable by another node.
- [ ] Standalone mode and the current `StartRelayPull`/`StopRelayPull` API remain usable without Redis/Asynq enabled.
- [ ] Tests cover state transitions, fencing, duplicate claims, stale task suppression, explicit-stop non-resurrection, process exit versus cancel signaling, process-triggered Redis renewal, stale progress, offline pending tasks, failover, and retry exhaustion.
- [ ] `go test ./app/oryxserver/...`, targeted race tests, `go vet ./app/oryxserver/...`, and `go build ./...` pass.

## Out of Scope

- Changing public protobuf names or adding a second public relay API.
- Cross-region Redis replication, multi-Redis consensus, or a new orchestration service.
- FFmpeg transcoding, source protocol conversion, or changing Oryx/SRS recording semantics.
- Automatic credential rotation or storing plaintext credentials in Redis, logs, or Asynq payloads.

## Technical Defaults

- FFmpeg progress interval: `5s`.
- Progress stale threshold: `15s`.
- Redis lease duration: `30s`.
- Lease renewal write timeout: `2s`.
- These defaults are configurable and may be adjusted during implementation only if tests demonstrate a concrete timing problem.
