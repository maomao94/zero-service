# Implement — oryxserver relay naming refactor

## Ordered checklist

1. Update `app/oryxserver/oryxserver.proto`:
   - Rename `StreamRelayReq/Res` -> `StartRelayPullReq/Res` (field `task_id` -> `relay_id`), `StreamRelayStopReq/Res` -> `StopRelayPullReq/Res`.
   - Add `[json_name]` to every field (camelCase).
   - Update comments to reference `/api/ctrl/start_relay_pull` and `/api/ctrl/stop_relay_pull`.
2. Run `app/oryxserver/gen.sh` to regenerate pb/grpc/validate/server/svc; verify `git status` only generated output changed.
3. Rename internal relay functions in `internal/relay/manager.go`, `ffmpeg.go`:
   - `Start` -> `StartPull`; `Stop` -> `StopPull`; `TaskIDs` -> `PullIDs`; keep `StopAll`.
   - `watch` -> `watchPullProcess`.
   - Deterministic `relay_id` generation from canonical `source_url`+`app`+`stream`.
   - Preserve `context.Background()` derivation and `.Silent(true)` in `buildFfmpegCmd`.
4. Rename `internal/relay/broadcast.go` constant `MethodStreamRelayStop` -> `MethodStopRelayPull`.
5. Update `mqtt/broadcast.go` to register `relay.MethodStopRelayPull` and rename `stopRelay` payload handling (identity).
6. Rename logic files/functions:
   - `streamrelaylogic.go` -> `startrelaypulllogic.go`, `func StreamRelay` -> `StartRelayPull`.
   - `streamrelaystoplogic.go` -> `stoprelaypulllogic.go`, `func StreamRelayStop` -> `StopRelayPull`.
   - Use `relay_id` (identity) in Stop path; keep cluster MQTT broadcast semantics; standalone clear error.
7. Update `internal/server/oryxserverserver.go` methods/imports (regenerated).
8. Check `app/oryxgtw` for any direct reference to the renamed RPCs (should be none besides internal hooks); update if present.
9. Update `main`/`oryxserver.go` if rename touches `RelayManager.Start`/`StopAll` callers.

## Validation commands

- `cd app/oryxserver && ./gen.sh` (regenerate)
- `go build ./app/oryxserver/...`
- `go test ./app/oryxserver/...`
- `git diff --check`
- `grep -rn "StreamRelay\|task_id" app/oryxserver --include=*.go` (should only match history/external comment; no residual symbol usage)

## Risky files / rollback points

- `oryxserver.proto` + generated `oryxserver/` dir (highest risk — contract changes; rollback = revert + regen).
- `internal/relay/manager.go` / `ffmpeg.go` / `broadcast.go` (behavior preserved by rename; ensure `StartAll`/waler semantics intact).
- `mqtt/broadcast.go` (executor registration key rename; rollback = restore constant name).
- `internal/logic/*.go` (rename only; no URL string change in target-building logic — keep `buildAuthQuery`).

## Follow-up checks before `task.py start`

- [ ] proto comments/field json_name complete.
- [ ] regression: start non-blocking, ffmpeg ctx `context.Background()`, `StopAll` + `defer waitForCalled()` intact.
- [ ] no hand-edits to generated files.
