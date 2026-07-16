# Research: Current gnetx Slow Logging Code

- **Query**: Search `common/gnetx/` for existing slow call / slow threshold / slow logging code
- **Scope**: internal
- **Date**: 2026-07-16

## Findings

### Slow Handler Threshold Definition

**File**: `common/gnetx/options.go`

| Location | Detail |
|---|---|
| Line 34-36 | `ServerOptions.SlowHandlerThreshold` — "on-loop 同步 handler 慢处理告警阈值，超过打 logx 日志。0 用默认 50ms。async handler 不计入（已 offload）。" |
| Line 157 | `const defaultSlowHandlerThreshold = 50 * time.Millisecond` |
| Line 190-192 | `ServerOptions.applyDefaults()` — zero-value fallback to `defaultSlowHandlerThreshold` |
| Line 91-93 | `WithSlowHandlerThreshold(d time.Duration) ServerOption` — public option setter |
| Line 213-214 | `ClientOptions.SlowHandlerThreshold` — same field for client |
| Line 267-269 | `WithClientSlowHandlerThreshold(d time.Duration) ClientOption` — client option setter |
| Line 352-354 | `ClientOptions.applyDefaults()` — zero-value fallback to same `defaultSlowHandlerThreshold` |

### Server-Side Slow Logging (sync handler)

**File**: `common/gnetx/server.go`, lines 256-285 (`dispatchSync`)

```
startTime := timex.Now()              // line 257
ctx, span := startServerSpan(...)     // line 258
// ... context injection ...
reply, hErr := h.Handle(ctx, cn, msg) // line 266
duration := timex.Since(startTime)    // line 268
if duration > s.opts.SlowHandlerThreshold {   // line 269
    logx.Slowf("[gnetx] slow handler %s id=%s", duration, cn.id)  // line 270
}
```

### Server-Side Slow Logging (async handler)

**File**: `common/gnetx/server.go`, lines 287-322 (`dispatchAsync`)

```
// Inside goroutine pool submit:
startTime := timex.Now()              // line 299
reply, hErr := h.Handle(ctx, cn, msg) // line 300
duration := timex.Since(startTime)    // line 301
if duration > s.opts.SlowHandlerThreshold {   // line 302
    logx.Slowf("[gnetx] async slow handler %s id=%s", duration, cn.id)  // line 303
}
```

### Client-Side Slow Logging (sync handler)

**File**: `common/gnetx/client.go`, lines 288-314 (`dispatchSync`)

Exactly the same pattern as server:
```
startTime := timex.Now()              // line 289
// ... context injection ...
reply, hErr := h.Handle(ctx, cn, msg) // line 298
duration := timex.Since(startTime)    // line 300
if duration > c.opts.SlowHandlerThreshold {   // line 301
    logx.Slowf("[gnetx] client slow handler %s id=%s", duration, cn.id)  // line 302
}
```

### Client-Side Slow Logging (async handler)

**File**: `common/gnetx/client.go`, lines 316-347 (`dispatchAsync`)

```
// Inside goroutine pool submit:
startTime := timex.Now()              // line 326
reply, hErr := h.Handle(ctx, cn, msg) // line 327
duration := timex.Since(startTime)    // line 328
if duration > c.opts.SlowHandlerThreshold {   // line 329
    logx.Slowf("[gnetx] client async slow handler %s id=%s", duration, cn.id)  // line 330
}
```

## Current gnetx Slow Logging Pattern

```
duration := timex.Since(startTime)
if duration > threshold {
    logx.Slowf("[gnetx] <variant> %s id=%s", duration, cn.id)
}
```

**Key characteristics**:
1. Uses `timex.Now()` / `timex.Since()` for duration measurement
2. Default threshold: **50ms** (`defaultSlowHandlerThreshold`)
3. Threshold is configurable via `ServerOptions.SlowHandlerThreshold` / `ClientOptions.SlowHandlerThreshold`
4. **Does NOT use `logx.WithContext(ctx)`** — no context propagation into slow log
5. **Does NOT use `WithDuration(elapsed)`** on the logger
6. Logs duration as a `%s` format argument (string representation of time.Duration)
7. Logs session id only (`cn.id`)
8. No RPC-like `[RPC] ok - slowcall` keyword format

## Caveats

- The `ctx` variable **is available** in all four dispatch functions (sync/async × server/client)
- The `ctx` already has session log fields injected (`ctx = injectSessionLogFields(ctx, cn)`)
- No context is passed into `logx.Slowf` — it's always called as a package-level function
- No `WithDuration` helper is used, unlike gormx and go-zero RPC patterns
