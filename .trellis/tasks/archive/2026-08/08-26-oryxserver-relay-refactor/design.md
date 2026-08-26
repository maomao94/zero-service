# Design — oryxserver relay naming refactor

## Scope

Rename the relay gRPC API and internal functions to operation-descriptive names, replace random `task_id` with deterministic `relay_id`, and introduce local replace semantics for repeated pulls. Distributed Redis/Asynq strengthening and cross-node ownership/takeover are a separate follow-up task, out of scope here.

## Boundary / Components

```
oryxserver.proto  →  RPC: StartRelayPull / StopRelayPull
                         Msg: StartRelayPullReq{source_url,app,stream}
                              StartRelayPullRes{relay_id,app,stream}
                              StopRelayPullReq{app,stream}
                              StopRelayPullRes{}
internal/relay/  →  Manager.StartPull(source, app, stream, target)(relayID, error)
                    Manager.StopPull(app, stream)
                    Manager.PullIDs()
                    Manager.StopAll()
                    watchPullProcess(...)         (defined in ffmpeg.go)
internal/logic/  →  StartRelayPullLogic / StopRelayPullLogic
mqtt/broadcast.go→  executor keyed by oryxserver.OryxServer_StopRelayPull_FullMethodName
main (oryxserver.go) →  proc.AddWrapUpListener(ctx.RelayManager.StopAll); defer waitForCalled()
```

## Data Flow

StartRelayPull:
1. validate `source_url` non-empty; fill `app` default = `RelayConfig.DefaultApp`; generate `stream` UUID if empty.
2. compute `relay_id = SHA256(canonical(source_url) + "\x00" + app + "\x00" + stream)` (return identifier, not a map key).
3. `Manager.StartPull(source, app, stream, target)`: under `m.mu`, if a task exists at `TargetKey(app, stream)` → `stopTaskLocked` (CAS-delete old entry + `Cancel()` + wait `done` closed) then start new process; `cmd.Start()` non-blocking; return `relay_id`.

StopRelayPull:
1. `Manager.StopPull(app, stream)` under `m.mu`: load by `TargetKey`; if found → `stopTaskLocked` and return true.
2. not found + cluster → MQTT broadcast `OryxServer_StopRelayPull_FullMethodName` with `protojson.Marshal(StopRelayPullReq)`; executor decodes, calls `StopPull`, returns `protojson.Marshal(StopRelayPullRes)`; standalone → clear "任务不在当前节点" error.

StopAll: under lock, cancel all; then `wg.Wait()`; registered with `proc.AddWrapUpListener`, main defers the wait func.

## Key Design Decisions

- **Deterministic `relay_id`**: derived from source URL + app + stream; same source+target → same id on any node. Used as a return identifier and for logging only (not the map key).
- **One task per `(app, stream)` (replace semantics)**: map key is `TargetKey(app, stream)`; a repeated `StartPull` for a target cancels the old process and waits for its exit before starting a new one, so two ffmpeg never coexist and get rejected by SRS `existing or busy`.
- **Concurrency**: `m.mu` serializes the check-stop-start critical sections; `watchPullProcess` uses `CompareAndDelete(key, task)` and `close(task.done)` after `Cmd.Wait()` so replacement never deletes a successor and `Wait()` is always honored (no zombie).
- **Stop by `(app, stream)`**: confirmed contract; single fixed target ⇒ `(app, stream)` uniquely identifies the one relay; stable across nodes/restarts.
- **Cluster broadcast = ieccaller protojson pattern**: payload is gRPC request protojson; executor performs the local action (via injected `RelayManager`, NOT via `StopRelayPullLogic`) and returns gRPC response protojson. Calling the Logic inside the executor would recurse (未命中再广播) into a broadcast storm.
- **Service exit = `AddWrapUpListener`**: killing relays is "stop producing", run it at wrap-up (first signal) with the full `GracePeriod` budget, before grpc drains (grpc `GracefulStop` runs in the shutdown phase). main defers `waitForCalled`.
- **Contract migration**: renaming RPCs is a wire-breaking change; no Go client in this repo calls `oryxserver.StreamRelay` (oryxgtw uses only internal hooks); the LAL `StartRelayPull` is a separate service.

## Trade-offs

- Replace semantics means a repeated pull restarts the process even if the same source returned; acceptable — a pull request maps to "this target must be fed by this source now".
- Different source → same `(app, stream)` now stops the old and starts the new (previous behavior blocked nothing locally and let SRS reject the newcomer).
- Renaming RPCs breaks external consumers expecting `StreamRelay`; note for Java/Nacos callers.

## Operational / Rollback

- Regenerate via `app/oryxserver/gen.sh`; do not hand-edit generated files.
- Rollback: revert proto + logic rename (git) — internal logic behavior is locally equivalent; only wire names and local replace semantics change.
- Validation: `go build ./...`, `go vet ./app/oryxserver/...`, `go test ./app/oryxserver/...`, `git diff --check`.
