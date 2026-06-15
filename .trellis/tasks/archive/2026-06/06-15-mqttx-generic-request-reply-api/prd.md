# Improve mqttx generic request reply API

## Goal

Improve `common/mqttx` request/reply API ergonomics so callers can receive typed reply values without `any` return values and manual type assertions, while preserving MQTT routing behavior and Go generic constraints.

## Confirmed Facts

- Go 1.26 still does not allow method-specific type parameters on non-generic types, so `func (c *Client) RequestReply[T any](...) (T, error)` is invalid Go.
- Current `Client.RequestReply(ctx, topicTemplate, tid, send, ttl...) (any, error)` uses a private `replyCaller` interface to erase `ReplyRouter[T]` and returns `any`.
- `ReplyRouter[T]` is already typed and internally returns `T` through private `do`.
- Current only production caller is `app/ieccaller/internal/svc`, which type-asserts `raw.(*types.BroadcastAckBody)` after `Client.RequestReply`.
- `common/mqttx` tests cover reply routing, dispatcher behavior, and current `Client.RequestReply` behavior.
- Existing spec already documents the `any` return and type-erasure decision; this task will supersede that decision if implemented.

## Requirements

- Add a typed request/reply API for `mqttx` callers that returns `T` directly.
- Remove the current public `Client.RequestReply(...)(any, error)` API; compatibility is not required for this cleanup.
- Keep `Client` as the owner of MQTT connection, subscriptions, dispatch, and registered reply routers.
- Keep `ReplyRouter.do`, `resolve`, `reject`, and `has` package-private; callers should not keep or invoke routers directly for request/reply.
- Avoid exposing protocol-specific payload fields or error semantics in `common/mqttx`.
- Preserve current topic-template based routing, `tid` matching, timeout/TTL behavior, and `WithReplyRouter` registration semantics.
- Update `ieccaller` to use the typed API and remove manual `any` type assertion.
- Update tests and messaging code-spec to reflect the final typed API.

## Acceptance Criteria

- [ ] `mqttx` exposes only a typed public request/reply API that returns `T` without caller-side type assertion.
- [ ] `app/ieccaller` no longer calls `Client.RequestReply(...)(any, error)` or type-asserts request/reply results from `any`.
- [ ] Existing `WithReplyRouter`, dispatcher priority, unmatched reply behavior, and subscription restore behavior remain unchanged.
- [ ] `go test ./common/mqttx ./app/ieccaller/...` passes.
- [ ] `.trellis/spec/backend/messaging-guidelines.md` documents the final API signatures, error behavior, and examples.

## Out Of Scope

- Changing MQTT payload schemas, topic names, or trace envelope behavior.
- Reworking dispatcher wildcard matching or handler registration semantics.
- Reintroducing public `ReplyRouter.Do/Resolve/Reject/Has` methods.

## Open Questions

- Resolved: compatibility is not required. Implement the typed requester/wrapper interface and remove the public `Client.RequestReply(...)(any, error)` method.
