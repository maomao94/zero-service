# Implementation Plan

## Checklist

- Add DJI reply-router construction helpers in `common/djisdk` for `services_reply` and `property/set_reply`, using `mqttx.NewReplyRouter[*ServiceReply]` and a shared DJI reply decoder.
- Replace `Client.pending` usage in `common/djisdk.Client` with the configured `mqttx.Client` request/reply path.
- Update `SendCommand` to call `mqttx.RequestReply[*ServiceReply]` against `ServicesReplyTopicPattern()` and keep existing publish/error/result behavior.
- Update `SetProperty` to call `mqttx.RequestReply[*ServiceReply]` against `PropertySetReplyTopicPattern()` and keep existing publish/error/result behavior.
- Remove or de-emphasize ordinary reply handlers from `SubscribeAll`, `SubscribeServicesReply`, and `SubscribePropertySetReply` so reply topics are handled by `mqttx` reply routers.
- Update `app/djicloud/internal/svc/servicecontext.go` to create `mqttx.Client` with DJI reply-router options before `djisdk.NewClient(...)`.
- Update focused tests/fakes for the new request/reply seam and nil-client handler tests.

## Validation Commands

- `go test ./common/mqttx ./common/djisdk ./app/djicloud/internal/hooks ./app/djicloud/internal/svc`
- If package dependencies make the svc test too broad or flaky, run the narrower affected packages and report the skipped reason.

## Risk Points

- `mqttx.Client` has an unexported `getReplyHandler` method, so `djsdk` test fakes cannot trivially implement it from another package. Prefer testing router helpers and behavior through real `mqttx` tests or package-local fakes where needed.
- `SubscribeAll` must not register reply topics as ordinary handlers after routers are installed, otherwise replies may be decoded twice or log misleading handler errors.
- `WithPendingTTL` naming may become less precise after replacement; keep it only if needed for existing caller compatibility.

## Review Gate

- User reviews `prd.md`, `design.md`, and `implement.md` before `task.py start` moves the task to implementation.
