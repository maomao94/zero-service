# 统一 ISP handler 返回 Message

## Goal

Unify ISP handler wrapping at the base protocol layer so both `ispserver` and `ispagent` handlers return `*isp.Message` plus `error`, and gnetx `any` results always carry a protocol message object.

## Requirements

- Provide reusable ISP base wrapper and response assembly methods in `common/isp`.
- Reuse the base wrapper for both server-side and client-side ISP inbound handlers where practical.
- Preserve current protocol behavior for success without items, success with items, and error responses.
- Avoid `wrapItems`-style adapter as the target design; migrate client business handlers to return `*isp.Message` directly.
- Keep fallback handling returning an `*isp.Message` response for unmatched ISP messages.
- Avoid broad unrelated refactors outside `common/isp`, `app/ispserver`, and `app/ispagent` unless required by compilation.

## Acceptance Criteria

- [x] `common/isp` exposes a reusable wrapper/response contract that can serve both ISP server and ISP agent directions.
- [x] `app/ispserver` keeps existing behavior while using the common wrapper API.
- [x] `app/ispagent` client inbound business handlers return `*isp.Message` directly, without `wrapItems` as the final shape.
- [x] Existing agent response semantics remain unchanged: errors map through `isp.ResponseCode`, empty item responses use generic response without items, non-empty item responses use generic response with items.
- [x] The affected packages compile and targeted tests pass.

## Notes

- User pointed at `common/isp/`, `app/ispserver/`, `app/ispagent/`, and `app/ispagent/internal/isp/client.go`.
- Direction update: do not keep the interim `wrapItems` adapter as the target solution; plan a common protocol wrapper and then optimize each client business handler to return `*isp.Message`.
