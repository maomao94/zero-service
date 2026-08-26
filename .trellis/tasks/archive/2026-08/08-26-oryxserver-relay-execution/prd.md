# oryxserver distributed relay execution

## Goal

Implement distributed relay state, lease fencing, direct local FFmpeg start, and Asynq compensation on top of the Redis/Asynq infrastructure from `08-26-oryxserver-relay-infra`.

## Dependency

The infrastructure child must be implemented and verified first. It provides the isolated `oryx-relay` queue, `oryx:relay:` task namespace, Redis client, Asynq lifecycle, and relay worker registration seam. This child must not create a second Asynq server or consume trigger queues.

## Requirements

- Persist one Redis record per `(app, stream)` with source/target, deterministic `relay_id`, desired state, monotonically increasing desired version, owner ID, lease token/version, lease expiry, process state, progress timestamp, attempt count, and last error.
- On `StartRelayPull`, durably write desired running state, acquire the target lease atomically, and immediately start FFmpeg in the gRPC node when it owns the lease.
- Same source and target is idempotent. A different source for the same target creates a newer fenced version; old owner/process cannot renew, release, or overwrite the new version.
- If local FFmpeg `Start` fails, enqueue one isolated reconciliation task. If a running process exits unexpectedly, enqueue the current version for compensation. Pending tasks must survive an offline worker and be consumed after service recovery.
- Asynq worker must re-read Redis, ignore stopped/superseded tasks, atomically acquire a valid lease, and start FFmpeg only after fencing checks. Redis errors are retryable and must fail closed.
- Use FFmpeg machine-readable `-progress pipe:1` output. Progress watcher events must conditionally renew Redis using current owner ID, lease token, desired version, and desired running state. A timer must not renew a dead process independently.
- Stop progress renewal on process exit, context cancellation, lease fencing failure, stale progress threshold, or explicit stop. Cancel and wait for `Cmd.Wait()` before replacement or compensation to prevent duplicate FFmpeg processes.
- `StopRelayPull` durably sets stopped state, invalidates the current fence, cancels the local process when owned, and makes old Asynq tasks terminal without resurrection.
- Startup recovery scans desired-running Redis records and enqueues reconciliation for missing/stale owners. Normal shutdown releases only matching local leases and leaves desired running records recoverable.
- Preserve standalone behavior when distributed mode is disabled and preserve existing public gRPC contract.

## Acceptance Criteria

- [ ] Concurrent starts across multiple nodes result in one valid lease and one FFmpeg process for a target.
- [ ] Direct local start is attempted from the gRPC node; failed starts and unexpected exits enqueue durable isolated compensation tasks.
- [ ] Offline workers do not lose tasks; a later worker resumes them without requiring the original gRPC node to be online.
- [ ] Process progress events renew only the current owner/version lease; stale or fenced processes cannot renew Redis.
- [ ] Explicit stop prevents delayed Asynq tasks, startup scans, stale workers, and lease takeover from restarting the relay.
- [ ] Node crash, lease expiry, source failure, and retry exhaustion are covered by tests; replacement never overlaps a slow old process.
- [ ] `go test ./app/oryxserver/...`, targeted race tests, `go vet ./app/oryxserver/...`, and `go build ./...` pass.

## Out of Scope

- New public protobuf methods or changes to existing API names.
- Trigger service changes or shared trigger queue consumption.
- FFmpeg transcoding and proof of remote SRS delivery beyond process/progress health.
