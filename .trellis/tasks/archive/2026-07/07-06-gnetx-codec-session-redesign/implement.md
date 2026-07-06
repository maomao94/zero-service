# Implementation Plan

## Checklist

- Update `Conn` and `session` with `NextSendSeq()` and `sendSeq`.
- Add `SequenceStart` to `ServerOptions` and `ClientOptions`.
- Thread `SequenceStart` through `newSession` calls in server, client, and dialer.
- Remove `CodecConn` and update `Codec`, `Serializer`, `NewFuncCodec`, and `debugSerializer` signatures.
- Update `RawSerializer` and `JSONSerializer` to the body-only interface.
- Update `LengthPrefixCodec`, `DelimiterCodec`, and `FixedLengthCodec` signatures and serializer calls.
- Update encode/decode call sites in `session.go`, `server.go`, `client.go`, and `dialer.go`.
- Pass handler ctx to sync `writeReply` methods.
- Update examples and tests that implement `Serializer` or `Codec`.
- Add/adjust tests for `NextSendSeq`, `SequenceStart`, and `Encode` ctx passthrough.

## Validation

- `go test ./common/gnetx`
- If broader package impact appears, run the smallest affected `go test ./...` subset feasible.

## Risk Points

- Public interface breakage for custom `Codec` and `Serializer` implementations.
- `Encode` call sites must all receive a non-nil context.
- Sequence start must be applied per new session, including reconnects and dialer-created short connections.
- Sync reply path currently drops ctx; missing this would break context-aware reply encoding.

## Rollback Points

- Interface changes are concentrated in `codec.go`, `serializer.go`, and built-in codec files.
- Runtime flow changes are concentrated in `session.go`, `server.go`, `client.go`, and `dialer.go`.
