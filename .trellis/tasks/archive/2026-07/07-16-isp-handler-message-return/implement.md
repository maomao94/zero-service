# Implementation Plan

## Checklist

1. Move ISP base client/server communication into `common/isp`.
   - Add `ClientConfig`, `Client`, `ClientRouter`, `ClientHandler`.
   - Add `ServerConfig`, `Server`, `ServerRouter`, `ServerHandler`.
   - Keep gnetx, codec, rootName codec, registration, heartbeat, request/response and SendSeq/RecvSeq handling inside `common/isp`.
2. Simplify wrapper on top of the migrated communication objects.
   - No exported `WrapOptions` / `ClientContext` pattern.
   - Business code registers protocol instruction handlers only.
   - Wrapper still guarantees nil/error/default response behavior and SendSeq assignment.
3. Refactor `app/ispagent/internal/isp/client.go` into a business extension.
   - Embed `*common/isp.Client`.
   - Keep task/model/robot handlers and report cache loop.
   - Use `ClientRouter.Handle` / `HandlePairs` for handler registration.
4. Refactor `app/ispserver`.
   - Remove private TCP server implementation.
   - Keep only protocol handler registration in `internal/isp`.
   - ServiceContext constructs `common/isp.NewServer` directly.
5. Update configs and specs.
   - Use common ISP config types directly from go-zero service config.
   - Update `.trellis/spec/backend/isp-guidelines.md` to document `common/isp.Client` / `Server` and handler registration boundary.

## Validation Commands

- `gofmt -w common/isp/*.go app/ispagent/internal/config/config.go app/ispagent/internal/isp/client.go app/ispserver/internal/config/config.go app/ispserver/internal/isp/router.go app/ispserver/internal/svc/servicecontext.go`
- `go test ./common/isp ./app/ispagent/internal/isp ./app/ispagent/internal/handler ./app/ispagent/internal/config ./app/ispserver/internal/isp ./app/ispserver/internal/svc ./app/ispserver/internal/handler ./app/ispserver/internal/config`
- `git diff --check`

## Review Gates

- Confirm business packages do not construct gnetx client/server directly for ISP.
- Confirm no `WrapOptions`, `ClientContext`, or `wrapItems` remains.
- Confirm app code registers protocol handlers rather than gnetx handlers.
- Confirm client/server config is passed directly without conversion helpers.

## Known Test Caveat

- `go test ./common/gnetx` may fail at existing timing-sensitive `TestClientOnConnectOnReconnect`; do not treat that as caused by this change unless the new diff touches `common/gnetx`.
