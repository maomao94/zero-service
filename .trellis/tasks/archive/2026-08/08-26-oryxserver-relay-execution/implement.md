# Implement — oryxserver distributed relay execution

## Ordered checklist

1. Confirm infrastructure child is implemented and its isolated queue/registration seam is available.
2. Add Redis state model and Lua/CAS operations for desired state, version, lease claim, renewal, release, and conditional process/error updates.
3. Refactor start/stop logic to persist desired state, claim/release leases, preserve idempotency, and fence replacements.
4. Extend FFmpeg command construction with machine-readable progress output and add a parser/watcher whose events conditionally renew Redis.
5. Preserve separate cancellation and process-exited synchronization; wait for `Cmd.Wait()` before replacement or compensation.
6. Enqueue unique reconciliation tasks on failed start and unexpected exit; implement worker re-read, claim, start, and retry/terminal handling.
7. Add startup recovery scan and graceful shutdown lease release.
8. Add unit, integration/fake-process, and race tests for distributed invariants.
9. Run the full validation suite and review Redis keys/task payloads for credential leakage.

## Validation

- `go test ./app/oryxserver/...`
- targeted `go test -race` for relay state and process lifecycle packages
- `go vet ./app/oryxserver/...`
- `go build ./...`
- `git diff --check`

## Risk points

- Every Redis write from a process watcher must be fenced by owner/token/version.
- Asynq handlers must never trust stale payload source/version without re-reading Redis.
- A cancelled process is not an exited process; never start a replacement before `Cmd.Wait()` returns.
