# Optimize ieccaller MQTT req/reply with mqttx

## Goal

Use the shared `common/mqttx` request/reply router for `app/ieccaller` cluster-mode MQTT broadcast acknowledgements, removing the service-local reply pool plumbing while preserving existing broadcast protocol behavior.

## Confirmed Facts

- `common/mqttx` provides `ReplyRouter[T]` with `Do`, `Consume`, `Resolve`, `Reject`, TTL, and close behavior backed by `antsx.ReplyPool`.
- `app/ieccaller` currently keeps `BroadcastReplyPool *antsx.ReplyPool[*types.BroadcastAckBody]` in `ServiceContext` and manually resolves replies from `mqtt/broadcast_ack.go`.
- Cluster broadcast publishes requests to `iec/broadcast` and expects replies on per-instance `iec/broadcast-ack/<instanceId>`.
- `types.BroadcastBody` and `types.BroadcastAckBody` already contain the correlation id field `tId` used for request/reply matching.
- `PushPbBroadcastWithAck` maps remote `ErrorKind` values back to local errors and unmarshals `ResponseBody` into the caller-provided response.

## Requirements

- Replace ieccaller's direct `antsx.ReplyPool` usage for MQTT broadcast ack wait/resolve with `mqttx.ReplyRouter[*types.BroadcastAckBody]`.
- Keep the existing MQTT topics, payload JSON fields, `tId` correlation behavior, ack success/error semantics, and 10 second wait timeout.
- Decode ieccaller broadcast ack payloads in an ieccaller-owned decoder because `mqttx` must remain protocol-neutral.
- Register the ack reply route through `mqttx.WithReplyRouter` instead of a normal `AddHandlerFunc` reply consumer.
- Preserve existing broadcast request handling and ack publishing behavior for remote instances.
- Keep changes focused to `app/ieccaller` unless a small `common/mqttx` adjustment is required by an observed bug.

## Acceptance Criteria

- [ ] `PushPbBroadcastWithAck` uses `mqttx.ReplyRouter.Do` to publish the broadcast request and await the matching ack.
- [ ] Broadcast ack messages on `iec/broadcast-ack/<instanceId>` are decoded and routed by `mqttx.ReplyRouter.Consume`.
- [ ] Empty or invalid ack payloads are logged/returned consistently without panics, and empty `tId` does not resolve a request.
- [ ] Existing remote ack publishing keeps `ErrorKind` values `timeout`, `duplicate`, `iec_rejected`, and `unknown` unchanged.
- [ ] Service shutdown closes the MQTT client/router path without double-closing obsolete service-local reply pools.
- [ ] Relevant Go tests or package-level test/build commands pass, or any inability to run them is documented.

## Out Of Scope

- Changing MQTT topic names, broadcast payload schema, or cluster deployment configuration.
- Reworking the large broadcast method switch or IEC command execution semantics.
- Adding new public `mqttx` APIs unless the existing router cannot support this migration.

## Open Questions

- None blocking after code inspection. Recommended implementation is the minimal migration to `mqttx.ReplyRouter` while preserving all protocol-level behavior.
