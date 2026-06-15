# Design: djsdk mqttx request/reply replacement

## Architecture

`mqttx` remains the protocol-neutral request/reply layer. `djsdk` owns DJI protocol details: reply topic templates, `ServiceReply` decoding, payload construction, result-code handling, and command-specific error wrapping.

The `djicloud` service constructs `mqttx.Client` with DJI reply routers through `mqttx.WithReplyRouter(...)`, then passes that client to `djisdk.NewClient(...)` as today.

## Data Flow

1. `app/djicloud/internal/svc/servicecontext.go` creates DJI reply routers for `services_reply` and `property/set_reply`.
2. `mqttx.MustNewClient(c.MqttConfig, mqttx.WithReplyRouter(...), ...)` registers those routers before connect/restore-subscribe.
3. `djisdk.SendCommand` builds the existing DJI `ServiceRequest`, publishes to `thing/product/{gateway_sn}/services`, and waits via `mqttx.RequestReply[*ServiceReply]` on `ServicesReplyTopicPattern()`.
4. `djisdk.SetProperty` builds the existing DJI `ServiceRequest`, publishes to `thing/product/{gateway_sn}/property/set`, and waits via `mqttx.RequestReply[*ServiceReply]` on `PropertySetReplyTopicPattern()`.
5. `mqttx` reply router decodes inbound reply payloads, extracts `tid`, and resolves only matching pending calls.

## Boundaries

- Do not add a second request/reply mechanism in `djsdk`.
- Do not move DJI JSON schema, topic names, or DJI result-code mapping into `mqttx`.
- Do not change event/status/request/drc-up handling except to avoid treating reply topics as ordinary handlers.
- Do not add dynamic `mqttx` reply-router registration unless implementation proves construction-time registration cannot satisfy existing usage.

## Compatibility

`SendCommand`, `SetProperty`, `SendCommandFireAndForget`, and high-level DJI command helpers keep their public signatures. `WithPendingTTL` can continue to configure the request/reply TTL, but its internal target changes from `djsdk`'s pool to DJI reply router construction.

Nil MQTT clients in tests remain usable for handler registration and offline-check tests as long as those tests do not call live request/reply paths without a router.

## Trade-Offs

Construction-time router registration keeps `mqttx` unchanged and preserves its current lifecycle model. The trade-off is that callers creating `mqttx.Client` must know to include DJI reply routers before `djisdk.NewClient(...)`.

## Rollback

Rollback is a focused revert of `common/djisdk`, `app/djicloud/internal/svc/servicecontext.go`, and updated tests. No data migration or persistent format change is involved.
