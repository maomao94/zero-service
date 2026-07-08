# ISP Agent Implementation Plan

## Preconditions

- Task status remains `planning` until these artifacts are reviewed.
- Implementation should start only after `task.py start .trellis/tasks/07-08-sip-client`.
- Code generation must follow go-zero proto-first workflow.

## Implementation Steps

1. Create protocol package `common/isp`.
   - Add constants for confirmed message IDs.
   - Add `EncodeMessageID(type, command)` and decode helper.
   - Add `Item` as `map[string]string` wrapper.
   - Add `Message` implementing `gnetx.Identifiable`, `gnetx.Correlatable`, and `gnetx.Response`.

2. Implement XML parsing/building.
   - Support root names `PatrolHost` and `PatrolDevice`.
   - Parse `SendCode`, `ReceiveCode`, `Type`, `Code`, `Command`, `Time`, and `Items`.
   - Encode Items as XML attributes under `<Items>`.
   - Preserve raw XML on decoded messages for diagnostics.

3. Implement gnetx Serializer/Codec constructor.
   - Serializer decodes `[TransmitSeq][ReceiveSeq][SessionSource][XMLLength][XML]`.
   - Serializer encodes reserved length field and XML body.
   - Constructor returns LengthPrefixCodec with leading/trailing `0xEB90`, offset `19`, adjust `2`, strip `2`.
   - Add unit tests for encode/decode round trip and Java demo frame compatibility.

4. Create `app/ispagent` service skeleton.
   - Add `ispagent.proto`.
   - Add `gen.sh` following adjacent services.
   - Run generation.
   - Add `ispagent.go`, `etc/ispagent.yaml`, `internal/config`, `internal/svc`, `internal/server`, `internal/logic`.

5. Implement `internal/ispclient.Manager`.
   - Initialize `gnetx.Client` with codec, handler/router, reconnect interval, request timeout, and max frame length.
   - On ready/reconnect, send register `251-1`.
   - Send heartbeat `251-2` on configured interval.
   - Provide `Execute(ctx, type, command, code, items)`.
   - Fill configured `SendCode` and `RootName` on outbound messages.
   - Use `RegisterReceiveCode` only for the initial register packet if needed.
   - Store the learned remote code from register response and use it as `ReceiveCode` for subsequent commands.

6. Implement gRPC logic.
   - `ExecuteCommand`: generic pass-through.
   - `SendPatrolDeviceStatusData`: fixed Type `1`, Command `0`.
   - `SendPatrolDeviceRunData`: fixed Type `2`, Command `0`.
   - `SendPatrolDeviceCoordinates`: fixed Type `3`, Command `0`.
   - Convert parsed protocol response into `CommandRes`.

7. Add tests.
   - `common/isp`: message ID helpers, XML parse/build, frame encode/decode round trip.
   - `app/ispagent/internal/ispclient`: use a lightweight TCP mock when feasible, or test request construction separately.

8. Validate.
   - `go test ./common/isp/...`
   - `go test ./app/ispagent/...`
   - `go build ./app/ispagent/...`
   - Broaden to `go test ./...` only if local dependency state allows reasonable runtime.

## Review Gates

- Generated code diff must be inspected after running `gen.sh`.
- No logic should directly instantiate protocol clients outside `ServiceContext`.
- No secrets or real production endpoints in yaml.
- Root element must remain configurable, not hard-coded.
- Normal command `ReceiveCode` must come from register response, not fixed config.
- Response must include parsed Items and raw XML.

## Rollback Points

- If `LengthPrefixCodec` cannot encode the reserved length field cleanly, replace with a small custom `gnetx.Codec` in `common/isp` while preserving the same `Message` and `Serializer` surface.
- If goctl generation is unavailable locally, stop after proto/service skeleton and report the missing generator instead of hand-writing generated files.
