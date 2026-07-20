# IEC ASDU trace propagation

## Goal

Preserve OpenTelemetry trace context for IEC104 ASDU messages from `ieccaller` ingress through Kafka/MQTT and batched gRPC forwarding, without mixing trace transport headers into business metadata.

## Requirements

- Start an OTel trace span when `ieccaller` receives an ASDU packet.
- Generate a business-level ASDU correlation ID for logs and ASDU payloads.
- Keep business `metaData` separate from trace transport headers.
- Persist trace carrier headers with each ASDU message before asynchronous chunk buffering.
- Push MQTT with the existing trace-aware MQTT path so consumers can restore trace context from message headers.
- Push Kafka with the traced context so go-queue can inject Kafka headers.
- In `iecstash`, preserve Kafka-consumer trace context when writing to the async chunk pusher and forward gRPC calls with recovered trace context.
- Handle batched gRPC forwarding correctly when one batch contains messages from different trace contexts.

## Acceptance Criteria

- [ ] `OnASDU` logs and downstream push logs include a stable `asduId` for one received ASDU packet.
- [ ] ASDU business `metaData` remains reserved for business data and does not receive `traceparent` / `tracestate` transport headers.
- [ ] MQTT ASDU publishing uses the trace-aware publish path.
- [ ] Kafka ASDU publishing receives a context containing the ASDU span so go-queue can inject trace headers.
- [ ] `iecstash` Kafka consumption carries trace context into its async batch buffer and gRPC push.
- [ ] Batched gRPC push groups messages by trace carrier before invoking `PushChunkAsdu`.
- [ ] Focused Go tests or builds for the changed packages pass.

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
