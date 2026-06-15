# Design

## Constraint

Go does not support method-level type parameters on non-generic types. This invalid shape cannot be implemented:

```go
func (c *Client) RequestReply[T any](ctx context.Context, topicTemplate string, tid string, send func() error, ttl ...time.Duration) (T, error)
```

Therefore the typed API must move the type parameter to either a package-level generic function or a generic wrapper type.

## Recommended Architecture

- Keep `Client` non-generic and responsible for MQTT connection, dispatch, and reply handler lookup.
- Add a generic typed client abstraction in `common/mqttx`, for example:

```go
type RequestReplyer[T any] interface {
    RequestReply(ctx context.Context, tid string, send func() error, ttl ...time.Duration) (T, error)
}

type RequestReplyClient[T any] struct {
    *Client
    RequestReplyer[T]
}

func NewRequestReplyClient[T any](cfg MqttConfig, topicTemplate string, decode ReplyDecoder[T], opts ...RequestReplyClientOption) (*RequestReplyClient[T], error)
```

- `RequestReplyClient[T]` embeds `*Client` plus `RequestReplyer[T]`; callers receive one client object with both MQTT client methods and typed request/reply methods.
- `ReplyRouter[T]` implements `RequestReplyer[T]` and is registered internally while creating the typed client.
- Remove `Client.RequestReply(ctx, topicTemplate, tid, send, ttl...) (any, error)`. Compatibility is not required; typed requester becomes the only public request/reply API.

## Data Flow

1. Protocol package creates `RequestReplyClient[T]` with a protocol-owned decoder and reply topic template.
2. Constructor creates/registers `ReplyRouter[T]` and creates the underlying `Client`.
3. Caller invokes `client.RequestReply(ctx, tid, send)`, receiving `T` directly.
5. MQTT dispatch resolves replies through the same `ReplyRouter[T].Consume` path as today.

## Trade-Offs

- A package-level `mqttx.RequestReply[T](ctx, client, topicTemplate, tid, send)` is shorter, but repeats `topicTemplate` at every call site.
- A standalone typed requester adds another object callers must explicitly create, which is not desired.
- `RequestReplyClient[T]` keeps request/reply embedded in the client object while preserving Go's non-generic `Client` constraint.
- Removing current `Client.RequestReply(any)` is a breaking change, but leaves `mqttx` with one safer public request/reply path and no caller-side type assertion.

## Compatibility

- No MQTT wire behavior changes.
- No topic or payload changes.
- `ReplyRouter[T]` remains public for lower-level construction/registration, but `RequestReplyClient[T]` is the preferred high-level API when callers need typed request/reply.

## Risks

- `RequestReplyClient[T]` only supports one typed reply route. Multiple typed reply routes still require explicit lower-level router/client composition.
