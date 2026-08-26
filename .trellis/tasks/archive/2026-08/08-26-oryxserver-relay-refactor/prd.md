# Refactor oryxserver relay naming and API

## Goal

Rename oryxserver's relay gRPC API and its internal FFmpeg relay functions to discrete, operation-descriptive names, and replace the random `task_id` business identity with a deterministic `relay_id` derived from the source URL and the fixed configured target. This is a behavior-preserving refactor: it lays the naming foundation before Redis/Asynq-based distributed pull strengthening, which is a follow-up task.

## Background and Confirmed Facts

- Current relay flow: `StreamRelayLogic.StreamRelay` -> `relay.Manager.Start` -> one FFmpeg copy-relay process (`ffmpeg -i <src> -c copy -f flv <target>`).
- Current manager stores tasks in process-local `sync.Map` keyed by a random UUID `task_id`; abnormal FFmpeg exit removes the task and does not retry.
- Target address is assembled from the single configured `RelayConfig.SrsRtmpAddr` + `app` + `stream` (config.go:41-52). There is exactly one fixed Oryx/SRS target service.
- Oryx recording requires a fixed target stream address; the relay publishes a device's fixed source stream to that address so Oryx can record it on task-segmented boundaries.
- The service is disttiuted gRPC with multiple nodes. Follow-up work will let another node continue a relay when its current node stops/fails.
- The relay API must be renamed to `StartRelayPull` and `StopRelayPull`, matching the underlying LAL HTTP API naming (`/api/ctrl/start_relay_pull`, `/api/ctrl/stop_relay_pull`).
- `task_id` as a random business identity is undesirable: the same `url + app + stream` must map to a single relay, so a deterministic identity is needed instead.
- Internal generic `Start` naming must be replaced with names describing the specific operation (pull/stop/stop-all/ffmpeg process lifecycle).
- The implementation must not call the trigger service. This refactor does not introduce it.

## Confirmed repo rules consulted (spec/backend/oryx-guidelines.md)

- FFmpeg process ctx must derive from `context.Background()` and NOT the gRPC request ctx (gRPC context cancels when RPC returns, killing the process).
- `StopAll` is registered via `proc.AddWrapUpListener(...)` and main must `defer` the returned `waitForCalled` so all ffmpeg exit before process exit (wrap-up phase stops producing first, before grpc drains).
- `buildFfmpegCmd` uses `.Silent(true)` to suppress ffmpeg-go's global std `log` prints; logging goes through `logx`.
- SRS rejects a second publisher for the same `(app, stream)` (keeps the old connection, new ffmpeg exits `251`). The single fixed SRS is a backend dedup safety net.
- Generated files must not be hand-edited; regenerate via `app/oryxserver/gen.sh` (goctl rpc protoc + `--client=false`).
- All proto fields carry explicit `[json_name = "..."]`.

## Requirements

### API rename (protobuf surface)

- Rename RPC `StreamRelay` -> `StartRelayPull`, message `StreamRelayReq`/`StreamRelayRes` -> `StartRelayPullReq`/`StartRelayPullRes`.
- Rename RPC `StreamRelayStop` -> `StopRelayPull`, message `StreamRelayStopReq`/`StreamRelayStopRes` -> `StopRelayPullReq`/`StopRelayPullRes`.
- Start message keeps `source_url` (required), `app` (optional, defaults to `RelayConfig.DefaultApp`), `stream` (optional, auto-generated when empty). All `[json_name]` tags preserved/explicit.
- Start response returns a deterministic `relay_id` in place of `task_id`, plus the actual `app` and `stream`.
- Stop message identifies the relay by `app` + `stream` (confirmed contract), dropping a random task identifier.
- Regenerate `oryxserver` pb/go/grpc/validate plus the server wiring via `gen.sh`. Logic/server/svc files must use goctl's standard naming only; no snake_case duplicates.

### Internal relay function rename + local replace semantics

- Replace generic `Manager.Start` with `Manager.StartPull(source, app, stream, target)`; `Stop` with `StopPull(app, stream)`; `TaskIDs` with `PullIDs`.
- Rename the watcher to `watchPullProcess`, keeping the correct `context.Background()` derivation and `.Silent(true)` handling.
- One task per `(app, stream)` key (`TargetKey`); a repeated pull cancels the old process and waits for its exit before starting a new one (replace semantics), serialized by `m.mu`; `watchPullProcess` uses `CompareAndDelete` + `close(done)` to avoid deleting a replacement and to honor `Wait()` (no zombie).
- Keep `task_id` fields internal only where needed, but the business identity surfaced to callers becomes the deterministic `relay_id`.

### Deterministic identity

- `relay_id` is derived deterministically, e.g. `hash(canonical(source_url) + "\x00" + app + "\x00" + stream)`, so the same source and target yields the same identity on any node. It is a return identifier; the map key is `TargetKey(app, stream)`.

### Cluster stop broadcast (ieccaller pattern)

- Payload = `protojson.Marshal(StopRelayPullReq)`; executor decodes to `StopRelayPullReq`, performs local `StopPull`, returns `protojson.Marshal(StopRelayPullRes)`; method key = gRPC-generated constant. Executor must NOT invoke the recursive Logic (broadcast storm); it injects only the minimal `RelayManager`.

## Acceptance Criteria

- [ ] `app/oryxserver/oryxserver.proto` exposes exactly `StartRelayPull` and `StopRelayPull`; `StreamRelay`, `StreamRelayStop`, `StreamRelayReq`, `StreamRelayRes`, `StreamRelayStopReq`, `StreamRelayStopRes` are removed.
- [ ] Generated protobuf/grpc/validate and server wiring are regenerated from the proto; `git diff` shows no hand-edits to generated files other than the regenerated output.
- [ ] `app/oryxserver/internal/logic/*` and the relay package compile with the renamed API and functions; no reference to the old names remains in oryxserver source (except history).
- [ ] Relay behavior: start returns immediately with the `relay_id`; ffmpeg ctx derives from `context.Background()`; `StopAll` is closed via `proc.AddWrapUpListener` + `defer waitForCalled()`.
- [ ] Repeated `StartPull` for the same `(app, stream)` cancels the previous process and waits for exit before starting a new one; two ffmpeg never coexist (SRS-dedup-safe); PullIDs returns the current relay ids.
- [ ] Cluster `StopRelayPull` broadcasts a `StopRelayPullReq` protojson payload and returns `StopRelayPullRes` protojson; executor performs local stop and ack (non-owner returns `ErrSkipAck`).
- [ ] `go build ./...`, `go vet ./app/oryxserver/...`, and `go test ./app/oryxserver/...` pass.
- [ ] The implementation does not call the trigger service.

## Out of Scope (deferred to follow-up task)

- Redis session/lease state, Asynq retry scheduling, cross-node ownership/takeover, and distributed `(app, stream)` uniqueness enforcement.
- The "source temporarily unavailable => retry instead of failing" behavior and "explicit stop prevents retry resurrection" semantics.
- Normal-shutdown ownership release and crashed-node lease takeover behavior.
- Supporting multiple independently configured target platforms; video transcoding; unrelated recording/SRS proxy APIs.

## Notes

- This is a complex task; `design.md` and `implement.md` are required before activation.
- Renaming changes the proto wire contract; confirm all direct callers (oryxgtw path and any consumer) are updated in the same change, or note a migration concern in `design.md`.
