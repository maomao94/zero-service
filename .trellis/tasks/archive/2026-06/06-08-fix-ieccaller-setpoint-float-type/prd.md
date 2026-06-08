# Fix ieccaller setpoint float protocol type

## Goal

Align the `ieccaller` setpoint-float API contract with IEC 60870-5-104 protocol semantics by defining `SendSetpointFloat` values as single-precision floats instead of doubles.

## Requirements

- Scope is limited to the `ieccaller` business/protocol surface for `SendSetpointFloat`.
- `SendSetpointFloatReq.value` must be represented as protobuf `float` because IEC 104 `C_SE_NC_1` / `C_SE_TC_1` short floating-point setpoints are IEEE 754 single precision.
- `SendSetpointFloatRes.value` must be represented as protobuf `float` because it is the substation ACK echo of the same single-precision protocol value.
- Generated Go code must be refreshed so the request/response value fields become `float32`.
- Server and MQTT broadcast code must compile against the regenerated `float32` API without unnecessary precision-widening in the protocol path.
- IEC 104 documentation must describe the value as protobuf `float` / IEEE 754 single precision and mention expected single-precision decimal display effects.
- Existing compatibility is not a constraint because these interfaces have not been used by external callers yet.

## Acceptance Criteria

- [ ] `app/ieccaller/ieccaller.proto` defines both `SendSetpointFloatReq.value` and `SendSetpointFloatRes.value` as `float`.
- [ ] Regenerated `app/ieccaller/ieccaller/ieccaller.pb.go` exposes these fields as Go `float32`.
- [ ] `app/ieccaller/internal/logic/sendsetpointfloatlogic.go` and `app/ieccaller/mqtt/broadcast.go` build without stale `float64` assumptions for setpoint float values.
- [ ] `docs/iec104-protocol.md` documents `SendSetpointFloat.value` as single-precision and clarifies precision/display behavior.
- [ ] Relevant tests or package builds for `app/ieccaller` and `common/iec104/client` pass.
- [ ] No proto float/double changes are made outside the `ieccaller` setpoint-float protocol surface.

## Out of Scope

- Changing GIS, DJI Cloud, AI, podengine, bridgedump, file, xfusionmock, or streamevent proto floating-point fields.
- Adding backward-compatible v2 fields or deprecation paths for `SendSetpointFloat`; this task assumes no external compatibility requirement.
- Changing IEC 104 wire behavior beyond making the public proto type match the existing single-precision protocol value.

## Notes

- User confirmed the API is not in use yet, so direct proto type changes are acceptable.
- This is a lightweight task; PRD-only planning is sufficient before activation.
