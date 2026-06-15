# Refactor djsdk request reply

## Goal

Refactor `common/djisdk` request/reply handling so cloud-to-device calls reuse the protocol-neutral request/reply capability already provided by `common/mqttx`, replacing the separate `antsx.ReplyPool` path in `djsdk`.

This should reduce duplicated pending-reply plumbing while preserving DJI protocol behavior for `services_reply` and `property/set_reply`.

## Confirmed Facts

- `common/mqttx` provides `ReplyRouter[T]` plus `RequestReply[T]`, keyed by topic template and request `tid`.
- `mqttx.WithReplyRouter(...)` registers reply routers at MQTT client construction time; dynamic public registration after client creation is not currently available.
- `common/djisdk.Client` currently owns `pending *antsx.ReplyPool[*ServiceReply]` and manually resolves replies in `HandleServicesReply` and `HandlePropertySetReply`.
- `SendCommand` waits on `services_reply`; `SetProperty` waits on `property/set_reply`; both currently call `antsx.RequestReply` directly.
- `SubscribeAll` currently registers `services_reply` and `property/set_reply` as ordinary handlers, not `mqttx` reply routers.
- The only production `djsdk.NewClient` caller is `app/djicloud/internal/svc/servicecontext.go`; `app/djicloud/internal/hooks/register_test.go` has nil-client tests for upstream handlers.
- User confirmed the intent is direct replacement because `mqttx` already provides responsive/request-reply MQTT behavior.

## Requirements

- Reuse `mqttx` built-in request/reply functionality for `djsdk` cloud-to-device request/reply flows where it fits, and prefer that path over the existing `antsx.ReplyPool` implementation.
- Preserve existing public behavior of `SendCommand` and `SetProperty`: return `tid`, publish the same DJI payload shape, wait for matching `tid`, and convert non-zero DJI result codes via existing error handling.
- Preserve fire-and-forget command behavior.
- Preserve non-request/reply inbound handling for events, requests, status, osd/state, and drc/up.
- Keep the change minimal and avoid introducing compatibility shims unless needed by an existing caller.
- Keep DJI protocol-specific decoding, topic names, and result-code interpretation in `djsdk`, not in `mqttx`.
- Update the `app/djicloud` MQTT client construction path to register DJI reply routers before creating `djsdk.Client`.

## Acceptance Criteria

- [ ] `SendCommand` uses `mqttx.RequestReply` or an equivalent `mqttx.ReplyRouter` path for `services_reply` matching by `tid`.
- [ ] `SetProperty` uses `mqttx.RequestReply` or an equivalent `mqttx.ReplyRouter` path for `property/set_reply` matching by `tid`.
- [ ] `djsdk` no longer duplicates a separate pending reply pool for these flows unless a concrete compatibility requirement makes it unavoidable.
- [ ] `services_reply` and `property/set_reply` decoding remains typed as `ServiceReply` and validates non-empty `tid`.
- [ ] `SubscribeAll` and targeted subscribe helpers do not cause duplicate reply-router and ordinary-handler consumption for request/reply topics.
- [ ] Existing `djsdk` tests pass or are updated to cover the new integration seam.
- [ ] Relevant `mqttx` tests continue to pass.

## Notes

- This is likely a lightweight implementation if scope is limited to `services_reply` and `property/set_reply`.
- If the change requires adding dynamic reply-router registration to `mqttx.Client`, treat it as broader cross-package design work and add `design.md` plus `implement.md` before starting.

## Open Questions

- None currently blocking planning.
