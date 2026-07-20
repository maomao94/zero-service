# Design

## Data Flow

`ieccaller` receives an IEC104 frame from the slave, generates a `traceId` via `NewClientContext`, and passes a context to `ClientCall`. `OnASDU` starts a `StartRecvSpan` inside the async goroutine so the span covers Kafka/MQTT push latencies. `PushASDU` sets `Headers` and `TraceId` on the MsgBody struct via `TraceHeaders(ctx)`, then marshals once:

| Path | Mechanism |
|------|-----------|
| Kafka | `PushWithKey(spanCtx, ...)` — go-queue injects OTel `traceparent` into Kafka message headers. JSON payload also carries top-level `headers`. |
| MQTT | `Publish(spanCtx, ...)` — preserve raw ASDU JSON; trace via top-level `headers` in payload. |
| gRPC (batch) | `StartForwardSpan(context.Background())` — independent root Producer span per chunk batch. `streamevent.MsgBody.headers` carries per-message trace for server-side restoration. |

`iecstash` consumes Kafka (value already has `headers` injected by ieccaller) and writes to chunk buffer. The chunk callback uses `context.Background()` — no fake parent from a random message.

## Boundaries

- Kafka: native message headers via go-queue + JSON `headers` for fallback.
- MQTT: JSON `headers` is the only trace transport.
- gRPC: `streamevent.MsgBody.headers` (map<string, string>) carries per-message trace; chunk batch span is independent.
- Trace headers and `traceId` are NOT injected into business `metaData`.

## Span Lifecycle

```
Slave → handle.go (NewClientContext: traceId, no span)
  → OnASDU → goroutine: StartRecvSpan (Consumer)
    → onSinglePoint/... → pushASDU → PushASDU
      → marshal → byteData has headers + traceId fields
        ├─ Kafka:  PushWithKey(span ctx, key, byteData)  // native Kafka headers + JSON headers
        ├─ MQTT:   Publish(span ctx, topic, byteData)     // JSON headers
        └─ Chunk:  Write(byteData)                        // JSON headers in async queue

ChunkAsduPusher callback (ieccaller):
  → StartForwardSpan(context.Background()) (independent root Producer)
    → gjsonHeadersMap → MsgBody.headers
    → PushChunkAsdu(ctx)

iecstash chunk callback:
  → context.Background()
    → gjsonHeadersMap → MsgBody.headers
    → PushChunkAsdu(ctx)
```
