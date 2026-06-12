# Design: mqttx reply review

## Boundaries

- `common/mqttx` owns MQTT client lifecycle, topic subscription tracking, handler dispatch, trace wrapping, and protocol-neutral reply routing.
- `ReplyRouter[T]` owns decoder delegation, `tid` validation, pending request resolution, and `ConsumeHandler` integration.
- `common/antsx.ReplyPool[T]` owns pending request storage, timeout, resolve/reject, and close semantics.
- Protocol packages own concrete MQTT topics, payload schemas, device identifiers, business result codes, and domain errors.

## Public API

- `ReplyMessage[T]` contains `Tid string` and `Value T`.
- `ReplyDecoder[T]` is an interface: `Decode(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[T], error)`.
- `ReplyDecoderFunc[T]` adapts function decoders to the interface.
- `NewReplyRouter[T](decoder ReplyDecoder[T], opts ...ReplyRouterOption) *ReplyRouter[T]` creates a router with default TTL/name options.
- `WithReplyRouter[T](topic string, router *ReplyRouter[T]) Option` registers reply routing during client construction.
- `WithReplyHandler` should be removed from the public API because it accepts arbitrary `ConsumeHandler`.

## Reply Router Contract

- Nil decoder is allowed at construction but returns `ErrNilDecoder` when handling a reply.
- Decoder errors are returned directly.
- Empty `Tid` returns `ErrEmptyReplyID` unless renamed to `ErrEmptyReplyTid` during implementation for consistency.
- Decoded but unmatched replies return `ErrReplyNotMatched` from `Consume`.
- `Do(ctx, tid, send, ttl...)` registers pending state before `send`, cleans up when `send` fails through `antsx.RequestReply`, and waits for a matching reply.
- `Close()` rejects all pending requests through `ReplyPool.Close()` and is invoked by `Client.Close()` for registered reply routers.

## Message Dispatch Model

- Store regular handlers and reply routers separately; registration path defines semantics.
- Keep regular handlers as `map[string][]ConsumeHandler`; append preserves execution order within the same topic filter.
- Keep reply routers as `map[string]ConsumeHandler`; one reply router per topic filter.
- Dispatch means runtime message routing after a MQTT message is received; it does not merge or deduplicate subscription topics.
- For each incoming MQTT message, use the callback `topicTemplate` as the route key; the MQTT client is responsible for wildcard subscription matching.
- Run all matching reply routers first, then all matching regular handlers.
- Suppress logs for `ErrReplyNotMatched`; log real reply decode/handler errors once at dispatch boundary.
- Call `onNoHandler` only when no reply router and no regular handler match the actual topic.

## Topic Matching Contract

- Topic matching is delegated to the MQTT client subscription mechanism.
- Dispatcher does not run an additional wildcard matcher across all registered filters.
- `topicTemplate` is the subscription filter whose callback fired, and is used as the exact lookup key.

## Subscription Topics

- `getAllTopicTemplates()` returns a de-duplicated set of registered regular handler topic templates and reply router topic templates.
- Topic merge/de-duplication belongs only to subscription restoration, not message dispatch.
- Topic list order is not contractual and tests must not depend on map iteration order.
- `restoreSubscriptions()` may append configured `SubscribeTopics` and de-duplicate before subscribing.

## Logging

- `tid` is the canonical unique request/reply message ID name, aligned with `antsx`.
- `mqttx` must not treat `tid` as DJI-only business data or add DJI-specific log fields.
- Avoid per-message success logs on hot MQTT consume paths.

## Compatibility

- This task may make a narrow breaking change from `WithReplyHandler` to `WithReplyRouter` because the current API is too broad and newly introduced.
- Ordinary consumers continue using `AddHandler` / `AddHandlerFunc`.
- No migration of existing DJI SDK code is included.
