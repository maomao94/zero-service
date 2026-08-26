# Design — oryxserver distributed relay execution

## Dependency

Depends on `08-26-oryxserver-relay-infra`. The infrastructure child supplies the isolated Redis/Asynq clients, worker lifecycle, queue, and handler registration seam.

## Redis state

Use a target-scoped hash/key namespace:

```text
oryx:relay:state:{escaped-target-key}
oryx:relay:lease:{escaped-target-key}
```

The state record contains `relay_id`, source/target, `desired_state`, `desired_version`, owner ID, lease token, lease expiry, process state, progress timestamp, attempt, and last error. Source credentials never enter logs or Asynq payloads.

All mutations that affect ownership use Lua/CAS conditions on owner ID, lease token, desired state, and desired version. A stale owner may not renew, release, mark failed, or enqueue a replacement for a newer version.

## Start and replace

1. Logic writes desired running state. Same canonical source/target is idempotent; a different source increments `desired_version`.
2. The gRPC node atomically claims the target lease for the current version.
3. The lease holder starts FFmpeg locally and returns the existing API response after `cmd.Start()` succeeds.
4. If `cmd.Start()` fails, conditionally record the error/release the lease and enqueue `oryx:relay:reconcile` with target and version.
5. A replacement fences the previous version before waiting for the old local process to exit. No new process starts until the old local process has returned from `Cmd.Wait()` when both are on the same node.

## Progress-driven renewal

Build FFmpeg with `-progress pipe:1` and a bounded progress interval. A watcher parses machine-readable progress events. Each valid event conditionally renews the Redis lease using owner/token/version and updates `last_progress_at`.

`context.CancelFunc` requests process termination. A separate internal process-exited signal is closed only after `Cmd.Wait()` returns. Progress events after cancellation or process exit are ignored. Renewal failures caused by fencing stop the local process; prolonged Redis uncertainty fails closed.

## Compensation

- Failed start and unexpected exit enqueue a unique target/version reconciliation task.
- Startup recovery scans desired-running state and enqueues only missing/stale owners.
- Worker re-reads Redis, acknowledges stopped/superseded records, acquires the current lease atomically, starts locally, and returns retryable errors for Redis/process failures.
- Asynq retry exhaustion records the terminal error and leaves desired state visible for later operator/manual reconciliation; it does not silently mark an explicit stop as running.

## Stop, failover, and shutdown

Stop atomically writes `desired_state=stopped`, increments/fences version, cancels a matching local process, waits for `Cmd.Wait()`, and makes stale tasks terminal. A crashed node leaves the desired state and lease TTL; another worker takes over after expiry. Normal shutdown releases only matching local ownership while preserving desired running state for recovery.

## Testing and rollback

Use miniredis/fakes for Redis CAS and an injectable process runner for FFmpeg. Test concurrent claims, fencing, stale progress, process exit versus cancellation, offline pending Asynq work, failover, and explicit-stop non-resurrection. Disable distributed mode to roll back operationally without changing the public API.
