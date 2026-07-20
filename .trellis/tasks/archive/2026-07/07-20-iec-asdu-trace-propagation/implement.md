# Implementation Plan

- [x] Add `common/iec104/trace.go`: `NewClientContext`, `StartRecvSpan`, `StartForwardSpan`, `TraceIdFromContext`, `TraceHeaders`, `ExtractTraceHeaders`
- [x] Extend `ASDUCall` interface with `context.Context` parameter on all callbacks
- [x] Update `ClientHandler` in `handle.go`: inject `NewClientContext` before calling `ASDUCall` methods
- [x] Implement `iecLogContext` in `ClientCall` with unified fields (host, port, stationId, traceId, iecType, typeId, coa, cot, cotCause, isNegative)
- [x] Start `StartRecvSpan` in `OnASDU` goroutine (covers Kafka/MQTT push)
- [x] `copyMetaData` ensures `ClientConfig.MetaData` thread-safety; no traceId in business metadata
- [x] `types.MsgBody` has `Headers` + `TraceId` fields; `PushASDU` sets via `TraceHeaders(ctx)` before marshal
- [x] MQTT uses `Publish` (not `PublishWithTrace`), trace via JSON `headers` + `traceId`
- [x] Add `streamevent.MsgBody.headers` (map<string, string>) + `traceId` fields to proto, populate in chunk callbacks
- [x] Chunk callbacks use independent span (ieccaller: `StartForwardSpan(context.Background())`, iecstash: `context.Background()`)
- [x] iecstash `asdu.go` no trace injection — Kafka value already has headers+traceId from ieccaller
- [x] Remove redundant log fields from `onCommandAck` and `onEndOfInitialization`
- [x] Update docs (`iec104-message.md`, `design.md`, `implement.md`)
- [x] Run focused tests/builds
