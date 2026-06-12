# Design

## Architecture

- `common/mqttx.ReplyRouter[*types.BroadcastAckBody]` becomes the owner of pending MQTT broadcast ack requests, managed internally by `mqttx.Client`.
- `ReplyRouter.do` is private; `Client.Reply` is the sole public request/reply entry point. Go does not support method type parameters, so `Client.Reply` returns `(any, error)` and callers type-assert to the expected type.
- `app/ieccaller/internal/svc` owns the `BroadcastAckBody` decoder because topic and payload semantics belong to the IEC caller protocol, not `mqttx`.
- `ServiceContext` creates the router as a local variable in cluster broadcast mode, passes it to `mqttx.MustNewClient` through `mqttx.WithReplyRouter`, and does not store any router reference.

## Data Flow

1. `PushPbBroadcastWithAck` generates `tId` and calls `svc.MqttClient.Reply(ctx, ackTopic, tId, send)`.
2. The `Reply` method looks up the registered reply router by topic template, registers the pending `tId`, and calls the send function which publishes via `pushBroadcast`.
3. Remote instances consume `iec/broadcast`, execute the requested command, and publish `BroadcastAckBody` to the caller's `ackTopic` unchanged.
4. Local MQTT client receives `iec/broadcast-ack/<instanceId>`, dispatches it to `ReplyRouter.Consume`, and resolves the pending `tId`.
5. `PushPbBroadcastWithAck` type-asserts the `any` result to `*types.BroadcastAckBody`, then maps error kinds and unmarshals response bodies as before.

## Compatibility

- No topic, config, proto, or JSON schema changes.
- The 10 second timeout remains the default by configuring the router TTL to 10 seconds.
- `mqttx` still unwraps trace envelopes before dispatch, so decoder receives the original `BroadcastAckBody` JSON.

## Trade-Offs

- `Client.Reply` returns `any` because Go methods cannot have type parameters; callers trade compile-time type safety for a clean public API without package-level generic helpers.
- Removing `BroadcastAck` as a normal handler reduces service-local duplicate reply routing, but places ack decode errors on the shared dispatcher error path.

## Rollback

- Reintroduce `BroadcastReplyPool`, normal ack handler registration, and manual `Register/Await/Resolve` if router integration causes unexpected dispatch behavior.
