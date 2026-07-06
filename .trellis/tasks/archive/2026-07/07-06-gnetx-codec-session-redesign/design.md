# Design

## Architecture

`gnetx` keeps transport/session concerns in the framework and protocol concerns in codecs.

Framework responsibilities:

- TCP connection lifecycle.
- Session identity, attributes, and close semantics.
- Connection-level send sequence allocation.
- Handler dispatch and reply writing.
- Request/response correlation through `ReplyPool`.

Codec responsibilities:

- Frame parsing and encoding.
- Protocol header parsing and construction.
- Sequence, ack, version, flags, CRC, transaction-id mapping.
- Deciding whether protocol sequence equals request/response TID.

## Public Contracts

`CodecConn` is removed. `Conn` becomes the session-facing contract available to codecs:

```go
type Conn interface {
    ID() string
    NextSendSeq() uint64
    Send(ctx context.Context, msg any) error
    RemoteAddr() net.Addr
    LocalAddr() net.Addr
    CreatedAt() time.Time
    LastActiveAt() time.Time
    SetAttribute(key, val any)
    Attribute(key any) any
    DeleteAttribute(key any)
    Close() error
}
```

`Codec` becomes:

```go
type Codec interface {
    Decode(c gnet.Conn, conn Conn) (any, error)
    Encode(ctx context.Context, msg any, conn Conn) ([]byte, error)
}
```

`Decode` keeps `gnet.Conn` because built-in and custom codecs need event-loop-only methods such as `Peek`, `Discard`, and `InboundBuffered`. `Encode` does not receive `gnet.Conn` because it can run from off-loop goroutines and must not expose read/write buffer primitives.

`Serializer` becomes body-only:

```go
type Serializer interface {
    Decode(raw []byte) (any, error)
    Encode(msg any) ([]byte, error)
}
```

## Sequence State

`session` owns `sendSeq atomic.Uint64`. New sessions store the configured sequence start during construction. `NextSendSeq()` returns the current value and increments atomically using fetch-and-increment semantics:

```go
func (s *session) NextSendSeq() uint64 {
    return s.sendSeq.Add(1) - 1
}
```

`ServerOptions.SequenceStart` and `ClientOptions.SequenceStart` feed `newSession`. Dialer uses `ClientOptions.SequenceStart` because it reuses client options.

## Context Flow

Existing handler contexts are created during dispatch. Reply paths must pass that same context to `Codec.Encode`:

- Sync handler reply: `writeReply(ctx, conn, reply)`.
- Async handler reply: `conn.Send(ctx, reply)` already carries ctx; `session.Send` forwards it to `Codec.Encode`.
- Active send/request: caller-provided ctx is forwarded to `Codec.Encode`.
- Heartbeat: use `context.Background()` because it is not a reply to an inbound request.
- Dialer request: use the request ctx for encoding.

The framework does not define concrete protocol header fields. Protocol codecs can use `context.Context` values and/or session attributes for request-scoped header data. The framework only guarantees that the context supplied to handlers is also supplied to reply encoding.

## Compatibility

This is a breaking change for custom codecs and serializers. Handler, router, message, response, and reply pool APIs remain stable.

Built-in codecs remain frame-focused and ignore context except for passing the new signature. Built-in serializers remain body-only.

## Trade-Offs

Using `Conn` directly removes the extra `CodecConn` abstraction. It exposes more methods to codecs than strictly required, but it avoids maintaining a near-duplicate interface and gives codecs access to session metadata and sequence generation.

Keeping `gnet.Conn` only in `Decode` preserves event-loop safety. Passing it to `Encode` would make unsafe buffer APIs available from off-loop code paths.

Removing connection parameters from `Serializer` narrows its role to payload serialization. Protocol headers and sequence/ack logic stay in `Codec`.
