# Implementation Plan

## Phase A: Proto First

- [ ] Change `AckHmsAlert` to return `AckHmsAlertRes` and define the empty platform response message.
- [ ] Rename the six mismatched platform response types to `<RpcName>Res` without changing their fields.
- [ ] Clarify `CommonRes` comments and service section comments so its DJI-only meaning is explicit.
- [ ] Clarify custom fly-region response comments as mixed platform orchestration plus DJI ACK results.
- [ ] Review every RPC declaration for the three categories in `design.md`.
- [ ] Run `app/djicloud/gen.sh` and inspect generated diff.

## Phase B: Logic

- [ ] Update `AckHmsAlertLogic` and server compilation for the new response type.
- [ ] Convert platform DB/OSS/validation failures to extproto errors; cover `AckHmsAlert`, flight-task progress, fly-region queries and writes.
- [ ] Preserve DJI ACK wrapping for standard command Logic and custom fly-region notification failures.
- [ ] Ensure fire-and-forget Logic returns extproto errors for local publish/sequence failures.

## Verification

- `./app/djicloud/gen.sh` or `(cd app/djicloud && ./gen.sh)`
- `go test ./app/djicloud/... ./common/djisdk/... ./common/tool/...`
- `git diff --check`
- Search all `AckHmsAlert`, `CommonRes`, `SubmitCustomFlyRegionRes`, `DeleteCustomFlyRegionRes` references after generation.

## Review Gates

- Stop after Phase A for proto contract review before starting implementation.
- Do not run `task.py start` until the proto review is approved.
