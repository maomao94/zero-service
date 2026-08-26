# oryxserver Redis Asynq relay infrastructure

## Goal

Introduce the Redis and Asynq runtime foundation required by distributed relay compensation, using the existing trigger service patterns while keeping relay tasks isolated from trigger tasks.

## Confirmed Constraints

- Trigger uses `common/asynqx`, Redis-backed Asynq, task types `defer:*` and `scheduler:*`, and queues `critical/default/low`.
- `oryxserver` currently has `DeployMode`, optional MQTT, and a local `RelayManager`, but no Redis or Asynq service context.
- This task must not implement distributed relay ownership or FFmpeg compensation behavior; that is the dependent execution task.

## Requirements

- Add optional oryxserver Redis configuration including address/password/database and connection timeouts using repository conventions.
- Add optional distributed relay configuration including enablement, node/instance ID, lease and progress timing, scan interval, retry limit, and backoff bounds.
- Add Asynq client, inspector, server, and task mux lifecycle to oryxserver, following trigger's `ServiceContext`, `serviceGroup`, and `common/asynqx` patterns.
- Isolate relay work from trigger through all of:
  - configurable or dedicated Redis DB, defaulting to a relay-specific DB when distributed mode is enabled;
  - dedicated queue name `oryx-relay` (not `critical`, `default`, or `low`);
  - task type prefix `oryx:relay:`;
  - distinct Redis key prefix `oryx:relay:`.
- Register only relay task handlers in the oryxserver mux. Existing trigger handlers and scheduler tasks must remain unchanged.
- Start and stop the Asynq worker cleanly with the oryxserver service group. Standalone mode must not require Redis/Asynq initialization.
- Provide a way for the dependent execution task to enqueue unique relay/version reconciliation tasks and to register its worker handler without duplicating client/server setup.
- Add config examples and startup validation that make queue/DB/prefix isolation observable in logs.

## Acceptance Criteria

- [ ] Distributed mode starts an oryxserver Asynq client, inspector, server, and worker; standalone mode preserves current startup without Redis.
- [ ] Relay tasks use queue `oryx-relay`, type prefix `oryx:relay:`, and key prefix `oryx:relay:`; no relay handler is registered for trigger task types and no trigger queue is consumed.
- [ ] Two services configured against the same Redis endpoint but different DB or queue namespaces cannot consume each other's tasks.
- [ ] Asynq server/client options follow existing 5-second Redis timeout and pool conventions, with relay-specific configuration documented.
- [ ] Graceful shutdown stops the relay worker without preventing local relay cleanup or leaving the service group blocked.
- [ ] Unit/configuration tests cover disabled mode, enabled mode, namespace isolation, task registration, and invalid configuration.
- [ ] `go test ./app/oryxserver/...`, `go vet ./app/oryxserver/...`, and `go build ./...` pass.

## Out of Scope

- Redis relay state schema, fencing, lease renewal, FFmpeg progress parsing, and compensation decisions.
- Changes to trigger's task types, queues, Redis DB, or worker behavior.
- Public protobuf/API changes.
