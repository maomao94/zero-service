# Implementation Plan

## Checklist

- Add typed `RequestReplyClient[T]` with embedded `*Client` and `RequestReplyer[T]` in `common/mqttx`.
- Create/register `ReplyRouter[T]` inside the typed client constructor.
- Remove public `Client.RequestReply(any)` and route all request/reply callers through typed requester.
- Update `app/ieccaller/internal/svc` to create/use `RequestReplyClient[*types.BroadcastAckBody]` and keep `MqttClient` from the embedded client.
- Add/update `common/mqttx` tests for typed success and send failure cleanup.
- Update `app/ieccaller` tests if needed.
- Update `.trellis/spec/backend/messaging-guidelines.md` signatures, matrix, examples, and design decision.

## Validation

- `go test ./common/mqttx ./app/ieccaller/...`
- `git diff --check`

## Review Points

- Confirm no public `ReplyRouter.Do/Resolve/Reject/Has` methods are reintroduced.
- Confirm no call site uses `Client.RequestReply(any)`, `NewRequestReplyer`, or type-asserts request/reply results from `any`.
- Confirm topicTemplate is still the exact subscription template, not actual topic.
