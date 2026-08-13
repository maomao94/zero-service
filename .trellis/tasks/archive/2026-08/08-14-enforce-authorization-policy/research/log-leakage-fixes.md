# Research: Log-leakage Fix Points (L1/L2/L3)

- **Query**: Exact locations of the three confirmed full-token logs, what to redact/replace, and whether any test locks the current log text
- **Scope**: internal
- **Date**: 2026-08-14

## Findings

### L1 — `facade/streamevent/internal/logic/upsocketmessagelogic.go:30-31` (Info)

```go
token := authctx.GetAuthorization(l.ctx)   // line 30
l.Logger.Infof("token: %s", token)          // line 31
```

- **Level**: Info (production-visible by default when level ≤ info).
- **Content**: full raw Socket.IO JWT / raw Authorization propagated via gRPC metadata from socketgtw (P3).
- **Redact shape**: replace with a presence boolean / masked value, e.g. `Infof("token_present=%t, len=%d", token != "", len(token))` — or, per audit §7.2, avoid even length unless approved (length can fingerprint issuer); safest is `token_present=%t` and log `authctx.GetUserId(l.ctx)` (claims, not raw token) if identity is needed.
- **Test lock**: none — `facade/streamevent` has no test files (`find` returned none); no test asserts the `"token: %s"` string.
- **Independently deployable**: yes; this is a single isolated change in one logic file, unrelated to interceptor/policy changes.

### L2 — `aiapp/mcpserver/internal/tools/echo.go:25-28` (Debug)

```go
auth := authctx.GetAuthorization(ctx)       // line 26
username := authctx.GetUserName(ctx)        // line 27
logx.Debugf("token: %s,username: %s", auth, username)  // line 28
```

- **Level**: Debug (only emitted when service log level ≤ debug; `aiapp/mcpserver/etc/mcpserver.yaml:14` currently sets `Level: debug`).
- **Content**: full raw Authorization restored from `_meta` (echo tool registered with `WithExtractUserCtx`, line 44).
- **Redact shape**: drop the `auth` value entirely; keep `username` (claims, information attribute per audit §6). E.g. `Debugf("username: %s", username)` or `Debugf("auth_present=%t, username: %s", auth != "", username)`.
- **Test lock**: none — no test files exist under `aiapp/mcpserver/internal/tools/`; grep for `token: %s` in `_test.go` across the repo returned zero.
- **Independently deployable**: yes; isolated file change.

### L3 — `common/mcpx/auth.go:45-65` (Debug)

```go
extra[authctx.CtxAuthorizationKey] = token   // line 61 — raw token stored in Extra
...
logx.WithContext(ctx).Debugf("[mcpx-auth] jwt verified, userId=%s, extra=%v", info.UserID, extra)  // line 65 — prints ENTIRE map including raw token
```

- **Level**: Debug; `aiapp/mcpserver/etc/mcpserver.yaml:14` sets `Level: debug`, so this is emitted in the current dev config.
- **Content**: line 61 puts the full raw JWT into `extra` map; line 65 formats the whole `extra` map with `%v`, printing the raw token.
- **Redact shape**: line 65 must not format the raw-token-containing map. Two independent aspects:
  1. *Log line only*: change to `Debugf("[mcpx-auth] jwt verified, userId=%s, extraKeys=%v", info.UserID, mapKeys(extra))` or explicitly format only claim keys without `CtxAuthorizationKey` (and without `exp` if considered sensitive).
  2. *Extra map content*: whether to remove `extra[authctx.CtxAuthorizationKey] = token` (line 61) is a **policy decision**, not purely a log fix — `TokenInfo.Extra` is the SDK boundary feeding tool context (echo). The audit L3 entry is specifically about the **log** printing the map; removing raw token from `Extra` changes the P5/P6 trust boundary and must be handled by the meta policy task, not silently as a "log fix".
- **Test lock**: YES — `common/mcpx/auth_test.go:46-48` (`TestDualTokenVerifierJWTPropagationContract`) asserts `info.Extra[authctx.CtxAuthorizationKey] == token`. **Removing line 61 breaks this test.** Changing only line 65's log format does not break it. No test asserts the log text itself.
- **Independently deployable**: the log-format change (line 65) is independent and safe; the Extra-content change (line 61) is entangled with the `_meta`/tool-context contract (auth_test.go + echo behavior) and should be deferred to the MCP meta policy work.

### Deployment note (audit U10)

All three are source-confirmed full-value log calls. Actual production exposure depends on per-deployment log level, sink ACL, and retention. Current dev YAMLs set `Level: debug` for aigtw and mcpserver; streamevent.yaml was not inspected for level — L1 is Info level regardless.

## Caveats / Not Found

- No test anywhere locks the `"token: %s"` / `"extra=%v"` log text, so log-format fixes are test-safe.
- L3 has a functional test lock on `Extra[CtxAuthorizationKey]` (auth_test.go:46-48) that constrains any change to line 61.
- Progress-log `token` fields (`wrapper.go`, `client.go:831-840`) are MCP progress correlation IDs, not Authorization — not part of L1-L3.
