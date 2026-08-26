# Design — oryxserver relay Redis/Asynq infrastructure

## Boundaries

This child owns configuration, client/server construction, isolated queue registration, and lifecycle wiring. It does not own relay state transitions or FFmpeg behavior.

Trigger remains unchanged and continues using `common/asynqx` with its existing task types and queues. Oryxserver uses the same low-level wrappers but creates a relay-specific Asynq server configuration:

```text
Redis DB:         configurable, relay-specific default
Queue:            oryx-relay
Task prefix:     oryx:relay:
Redis key prefix: oryx:relay:
```

The dedicated DB is the strongest isolation. Queue and type/key prefixes remain mandatory defense-in-depth when operators point both services at one Redis DB.

## Components

- `app/oryxserver/internal/config/config.go`: distributed relay and Redis settings.
- `app/oryxserver/internal/svc/servicecontext.go`: optional relay Redis client, Asynq client/inspector/server, and dependency bundle.
- `app/oryxserver/internal/task/`: relay mux and worker registration seam, private to oryxserver.
- `app/oryxserver/oryxserver.go`: service-group lifecycle and wrap-up ordering.
- `common/asynqx`: extend only if an existing wrapper lacks a needed option; do not alter trigger defaults.

## Lifecycle

1. Load and validate configuration.
2. If distributed mode is disabled, do not require Redis and do not construct the relay worker.
3. If enabled, construct Redis and Asynq using the configured relay DB and queue weights.
4. Register only `oryx:relay:*` handlers on the relay mux.
5. Add the worker to the same service group as gRPC so it starts and stops with the service.
6. Let the execution child register the reconciliation handler through the provided narrow seam.
7. Stop the worker before final service shutdown while allowing the relay manager to cancel local FFmpeg and wait for process exit.

## Asynq isolation

Use `asynq.Queue("oryx-relay")` when enqueuing tasks and configure the server with only that queue. Use a stable task type such as `oryx:relay:reconcile`. Do not register `defer:*` or `scheduler:*` handlers, and do not use trigger queue names.

Asynq payloads contain only target/version identifiers. Full source URLs and credentials are retrieved from Redis by the execution child.

## Compatibility and rollback

- Standalone remains the default and has no Redis/Asynq startup dependency.
- Existing trigger configuration and workers are untouched.
- Rollback is configuration-disable first, then removal of oryxserver worker wiring; public protobufs are unchanged.
