# Implement — oryxserver relay Redis/Asynq infrastructure

## Ordered checklist

1. Read backend lifecycle, concurrency, trigger, and Oryx guidelines before editing.
2. Add relay-specific Redis and distributed configuration with safe standalone defaults and validation.
3. Add optional Redis/Asynq dependencies to `ServiceContext`, using existing `common/asynqx` constructors and timeout conventions.
4. Add a private oryxserver relay task package with queue/type/key constants, mux registration, and a narrow handler registration interface for the execution child.
5. Wire the relay worker into service startup/shutdown without changing trigger task registration.
6. Add config example/comments and startup logs showing mode, Redis DB, queue, and task prefix without secrets.
7. Add tests for disabled mode, enabled mode, isolated task registration, and invalid configuration.

## Validation

- `go test ./app/oryxserver/...`
- `go vet ./app/oryxserver/...`
- `go build ./...`
- `git diff --check`
- Verify no `defer:*`, `scheduler:*`, `critical`, `default`, or `low` relay registration is introduced.

## Risk points

- `ServiceContext` initialization must not make standalone startup require Redis.
- Asynq server queue configuration must be restricted to `oryx-relay`.
- Service-group stop ordering must not leave FFmpeg processes running.
