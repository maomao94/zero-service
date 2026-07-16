# Research: Summary — Aligning gnetx Slow Logging with go-zero Pattern

- **Query**: What needs to change to align gnetx slow logging with go-zero's pattern
- **Scope**: internal
- **Date**: 2026-07-16

## Current State

### gnetx slow log (4 locations, same pattern)

```
duration := timex.Since(startTime)
if duration > <threshold> {
    logx.Slowf("[gnetx] <variant> %s id=%s", duration, cn.id)
}
```

**Files**:
- `common/gnetx/server.go:268-270` — sync handler
- `common/gnetx/server.go:301-303` — async handler
- `common/gnetx/client.go:300-302` — sync handler
- `common/gnetx/client.go:328-330` — async handler

### What go-zero Does (canonical pattern)

```
logx.WithContext(ctx).WithDuration(elapsed).Slowf("[PREFIX] <message>", ...)
```

Seen in:
- `zrpc/internal/clientinterceptors/durationinterceptor.go:38-46` — `logx.WithContext(ctx).WithDuration(elapsed).Slowf("[RPC] ok - slowcall - %s", serverName)`
- `common/gormx/logger.go:106-108` — `logx.WithContext(ctx).WithDuration(elapsed).Slowf("[gorm] [rows:%s] [SLOW] %s", formatRows(rows), sql)`

## Gaps to Align

### 1. Missing `logx.WithContext(ctx)`

**Current**: `logx.Slowf("[gnetx] ...", duration, cn.id)`

**Expected**: `logx.WithContext(ctx).Slowf("[gnetx] ...", cn.id)`

The `ctx` variable **already exists** in all four dispatch functions. It already has session fields injected (`ctx = injectSessionLogFields(ctx, cn)` at lines 264/293 of server.go, lines 296/322 of client.go). It's just not being used in the slow log call.

**Impact**: Without `WithContext(ctx)`, the slow log loses trace ID, span ID, and any other context-scoped log fields. This breaks trace correlation in production debugging.

### 2. Missing `WithDuration(elapsed)`

**Current**: Duration passed as `%s` format argument — e.g. `logx.Slowf("...%s...", duration, ...)`

**Expected**: Duration attached via helper — e.g. `logx.WithContext(ctx).WithDuration(elapsed).Slowf("...", cn.id)`

The go-zero logger natively supports `.WithDuration(d)` which adds a `duration=XXXms` field to the log entry. Using it instead of `%s` in the message:
- Keeps the message string clean
- Enables structured log parsing (JSON output mode)
- Follows go-zero convention

### 3. Log Message Format

**Current** gnetx messages:
```
[gnetx] slow handler 50ms id=abc123
[gnetx] client slow handler 50ms id=abc123
[gnetx] async slow handler 50ms id=abc123
[gnetx] client async slow handler 50ms id=abc123
```

**Proposed** go-zero-aligned messages (with `WithDuration` doing the duration formatting):
```
[gnetx] slow handler id=abc123                  (+duration field)
[gnetx] client slow handler id=abc123           (+duration field)
[gnetx] async slow handler id=abc123            (+duration field)
[gnetx] client async slow handler id=abc123     (+duration field)
```

The `%s` format for duration is moved into the `WithDuration` field, so the message format itself no longer needs a duration placeholder.

### 4. No Other Structural Changes Needed

- **Threshold** (50ms default) — already configurable via `SlowHandlerThreshold`, no change needed
- **Measurement** (`timex.Now()` / `timex.Since()`) — already correct, no change needed
- **`Slowf` log level** — already correct, no change needed
- **Scope** (all four locations: server/client × sync/async) — already covers all handler paths

## Change Summary

Each of the 4 slow-log lines needs a 2-part change:

1. **Add context chaining**: `logx.Slowf(...)` → `logx.WithContext(ctx).Slowf(...)`
2. **Add duration chaining**: `.Slowf(...)` → `.WithDuration(duration).Slowf(...)`
3. **Remove duration from format string**: Remove `%s` for duration from the format string since it's now in the struct field

### Before → After (example: server sync)

```go
// BEFORE (server.go:268-270)
duration := timex.Since(startTime)
if duration > s.opts.SlowHandlerThreshold {
    logx.Slowf("[gnetx] slow handler %s id=%s", duration, cn.id)
}

// AFTER
duration := timex.Since(startTime)
if duration > s.opts.SlowHandlerThreshold {
    logx.WithContext(ctx).WithDuration(duration).Slowf("[gnetx] slow handler id=%s", cn.id)
}
```

Same pattern applies to all 4 locations:
- `server.go:268-270` — sync server
- `server.go:301-303` — async server
- `client.go:300-302` — sync client
- `client.go:328-330` — async client

## Files That Need Changes

| File | Lines | Change |
|---|---|---|
| `common/gnetx/server.go` | 270 | Add `WithContext(ctx).WithDuration(duration)` to slow log |
| `common/gnetx/server.go` | 303 | Add `WithContext(ctx).WithDuration(duration)` to slow log |
| `common/gnetx/client.go` | 302 | Add `WithContext(ctx).WithDuration(duration)` to slow log |
| `common/gnetx/client.go` | 330 | Add `WithContext(ctx).WithDuration(duration)` to slow log |

**No other files** need changes. The `options.go` configuration is already complete. The `ctx` variable is already available and pre-populated with session log fields in all four functions.

## Caveats

- The `ctx` in async handlers is captured at dispatch time (before offloading to goroutine pool). This is the same pattern go-zero RPC uses — context is captured at interceptor entry.
- `WithDuration` is a method on `logx.Logger` (returned by `logx.WithContext`), not a package-level function. The chain must be `.WithContext(ctx).WithDuration(d).Slowf(...)`.
- No test files need updating — there are no tests specifically asserting the slow log format string.
