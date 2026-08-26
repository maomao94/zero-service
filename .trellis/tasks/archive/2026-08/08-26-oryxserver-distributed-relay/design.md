# Design — oryxserver distributed relay parent integration

## Child Ordering

1. `08-26-oryxserver-relay-infra` adds the optional Redis/Asynq runtime and strict relay namespace isolation.
2. `08-26-oryxserver-relay-execution` uses that runtime to implement Redis state, fencing, direct local start, progress-driven renewal, and compensation.
3. Parent integration review verifies the public API, standalone behavior, trigger isolation, and cross-node invariants together.

The parent is not an implementation target while the two children are active. It owns the shared requirements and final integration review.

## Shared Contract

```text
desired state: Redis, target scoped, running/stopped
ownership:     Redis lease + owner/token/version fencing
work queue:    Asynq queue=oryx-relay
task type:     oryx:relay:reconcile
key prefix:    oryx:relay:
direct path:   gRPC node writes state, claims lease, starts local FFmpeg
fallback:      start/exit/recovery -> isolated Asynq task -> any node claims lease
liveness:      FFmpeg -progress pipe:1 event triggers conditional Redis renewal
stop:          durable stopped state fences old workers and prevents resurrection
```

## Integration Gates

- Infrastructure child must pass its namespace/lifecycle tests before execution starts.
- Execution child must prove Redis fencing and process lifecycle tests before parent integration review.
- Final review must verify trigger remains unchanged, standalone startup remains Redis-free, and no credential enters Redis logs or Asynq payloads.

## Rollback

Disable distributed relay mode first. This leaves the existing local manager/API available while the new Redis/Asynq wiring and execution behavior can be reverted independently.
