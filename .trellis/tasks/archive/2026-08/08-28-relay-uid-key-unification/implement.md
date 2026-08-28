# Implementation Checklist

## Ordered Work

- [ ] Add `CanonicalUID(target string) (string, error)` and focused tests for full URL, normalized endpoint, UID, query stripping, invalid path, and forbidden `:` in the returned UID.
- [ ] Replace `hashKey` and all Redis key helpers with UID-based helpers; update comments and key constants to document exact values.
- [ ] Change `RelayState`, `ReconcilePayload`, and `StopPayload` to carry UID plus explicit state address fields required to start the process.
- [ ] Update `RelayRegistry.startLocalRelay`, `StartRelay`, `stopLocalRelay`, `HasTarget`, `handleOutput`, `handleExit`, `StopRelayByAppStream`, and `StopAll` so `meta` and ffmpeg Manager always use UID.
- [ ] Update `SaveState`/`GetState`/lease/lock/registry methods to consume UID consistently and preserve dynamic deadline TTL.
- [ ] Update start, stop, reconcile, scanner, MQTT broadcast, and task handlers so no caller reconstructs identity by concatenating `SrsRtmpAddr`.
- [ ] Make stop cleanup explicit: state/lease/registry deletion happens under the UID lock even when no local process metadata exists; distinguish lock failure from process-not-found.
- [ ] Update scanner to use stale UID members, lock each UID, reread state, then enqueue UID payload before setting pending state; verify enqueue/save failure semantics.
- [ ] Update existing relay tests and add integration-level Redis contract tests where the project test infrastructure permits.
- [ ] Update `.trellis/spec/backend/oryx-guidelines.md` with the UID contract, field schema, validation matrix, migration behavior, and wrong/correct examples.

## Verification Commands

```bash
gofmt -w app/oryxserver/internal/relay app/oryxserver/internal/logic app/oryxserver/internal/cron app/oryxserver/internal/task app/oryxserver/mqtt common/tool
go test ./app/oryxserver/...
go vet ./app/oryxserver/...
go build ./app/oryxserver/...
git diff --check
```

## Risk Checks

- Confirm every `processes.Start`, `processes.Stop`, `processes.Has`, `meta[...]`, `StateStore`, and queue payload call uses the same UID.
- Confirm `ParseTarget` remains available for extracting app/stream and does not silently become the identity function.
- Confirm `StopRelayAndRecording` cannot return success while leaving state/lease/registry keys because the local process was absent.
- Confirm old Redis keys are intentionally incompatible and the release note/operator cleanup command is documented.
