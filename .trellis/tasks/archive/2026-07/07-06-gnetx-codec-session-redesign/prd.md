# Redesign gnetx codec session interfaces

## Goal

Redesign `common/gnetx` codec and session interfaces so the framework remains protocol-agnostic while supporting connection-level send sequence generation and request-scoped protocol header context for replies.

## Requirements

- `Conn` exposes a connection-level `NextSendSeq() uint64` API backed by per-session atomic state.
- `ServerOptions` and `ClientOptions` support `SequenceStart uint64`; the first `NextSendSeq()` call returns that configured value.
- The public `CodecConn` interface is removed. `Codec` implementations receive the public `Conn` interface for session metadata and sequence access.
- `Codec.Encode` receives `context.Context` so protocols can read request-scoped packet/header context when encoding replies.
- `Codec.Decode` continues to receive `gnet.Conn` for event-loop-only `Peek`/`Discard` I/O and also receives `Conn` for session metadata.
- `Serializer` is simplified to body-only serialization and no longer receives connection/session parameters.
- `Handler`, `Router`, `Correlatable`, `Response`, and `ReplyPool` behavior remain unchanged.
- Request/response correlation remains based on `Correlatable.TID()` and `Response.ResponseTID()` strings; sequence values are protocol-owned and mapped inside codecs.
- Built-in codecs, serializers, examples, and tests are updated to the new interfaces.
- Reply encoding paths pass the handler/request context through to `Codec.Encode`.
- Active sends, heartbeat sends, and dialer sends call `Codec.Encode` with an appropriate context.

## Acceptance Criteria

- [x] `CodecConn` no longer exists in production or test code.
- [x] `Conn.NextSendSeq()` is available on server, client, and dialer sessions.
- [x] `SequenceStart` is applied to new server/client/dialer sessions and verified by tests.
- [x] `Codec.Encode(ctx, msg, conn)` is used consistently by `Send`, sync replies, async replies, heartbeats, and dialer requests.
- [x] `Codec.Decode(gnet.Conn, Conn)` continues to support the existing built-in framing behavior.
- [x] `Serializer.Decode(raw)` and `Serializer.Encode(msg)` are body-only and all built-in serializers compile under the new contract.
- [x] Handler-facing APIs remain source-compatible except for code that directly implements `Codec` or `Serializer`.
- [x] Existing gnetx tests pass after updating them to the new public interfaces.
- [x] At least one test demonstrates context-aware reply encoding or verifies the ctx is passed into `Encode`.

## Notes

- This is a breaking interface change for custom `Codec` and `Serializer` implementations.
- `gnet.Conn` is intentionally kept out of `Encode` because encoding can run off the gnet event loop.
