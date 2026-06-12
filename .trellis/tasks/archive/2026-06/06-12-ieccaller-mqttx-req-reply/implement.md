# Implementation Plan

## Checklist

- Read relevant backend messaging and code reuse specs before editing.
- Add an ieccaller MQTT broadcast ack decoder compatible with `mqttx.ReplyDecoder[*types.BroadcastAckBody]`.
- Replace `BroadcastReplyPool` in `ServiceContext` with `BroadcastReplyRouter` and initialize it in cluster mode with a 10 second TTL/name.
- Construct `mqttx.Client` with `mqttx.WithReplyRouter(ctx.BroadcastAckTopic(), ctx.BroadcastReplyRouter)` when broadcast mode is enabled.
- Update `PushPbBroadcastWithAck` to use `ReplyRouter.Do` instead of manual register/await.
- Remove obsolete normal ack handler registration and the service-local ack consumer file if no longer referenced.
- Update shutdown to rely on MQTT client reply-router close or close the router only when no client exists.

## Validation

- `go test ./common/mqttx ./app/ieccaller/...`
- If package tests are too broad or blocked by external dependencies, run the narrowest compiling package set and document the blocker.
- `git diff --check`

## Risk Points

- `mqttx.WithReplyRouter` must be applied before `NewClient` connects so reconnect restore subscriptions includes the ack topic.
- Avoid double-closing the router because `mqttx.Client.Close` closes registered reply handlers.
- Keep ack decode failures visible but do not resolve pending requests with invalid or empty `tId`.
