# Implement — oryxserver distributed relay parent integration

## Ordered Checklist

1. Activate and implement `08-26-oryxserver-relay-infra`.
2. Run its quality check and verify trigger queue/task behavior is unchanged.
3. Activate and implement `08-26-oryxserver-relay-execution` against the infrastructure seam.
4. Run its race, process lifecycle, Redis state, and compensation checks.
5. Perform parent integration review across API, standalone mode, shutdown, Redis fencing, Asynq isolation, and credential handling.
6. Update Oryx/concurrency/service lifecycle specs with stable project-level rules learned during implementation.
7. Run full build/test/vet/diff validation and commit only after both children pass.

## Validation

- `go test ./app/oryxserver/...`
- targeted `go test -race` for relay packages
- `go vet ./app/oryxserver/...`
- `go build ./...`
- `git diff --check`
- verify trigger tests and task registration remain unchanged

## Rollback Points

- Disable `DistributedRelay` configuration after infrastructure changes.
- Revert execution child independently if Redis state/fencing tests fail.
- Do not alter trigger task names, queues, or Redis DB during rollback.
