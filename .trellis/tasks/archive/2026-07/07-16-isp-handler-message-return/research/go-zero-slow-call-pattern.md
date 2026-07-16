# Research: go-zero Slow Call Pattern Used in This Project

- **Query**: Search for `slowThreshold`, `slowcall`, `Slowf`, `timex.Since` patterns
- **Scope**: mixed (internal + go-zero library)
- **Date**: 2026-07-16

## Findings

### 1. go-zero zrpc DurationInterceptor (THE canonical pattern)

**File**: `github.com/zeromicro/go-zero@v1.10.0/zrpc/internal/clientinterceptors/durationinterceptor.go`

This is go-zero's **built-in RPC client slow call logger** — the reference pattern.

```
const defaultSlowThreshold = time.Millisecond * 500   // line 16

var slowThreshold = syncx.ForAtomicDuration(defaultSlowThreshold) // line 20

func DurationInterceptor(ctx context.Context, method string, req, reply any,
    cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
    serverName := path.Join(cc.Target(), method)
    start := timex.Now()
    err := invoker(ctx, method, req, reply, cc, opts...)
    if err != nil {
        logger := logx.WithContext(ctx).WithDuration(timex.Since(start))  // context + duration
        logger.Errorf("fail - %s - %s", serverName, err.Error())
    } else {
        elapsed := timex.Since(start)
        if elapsed > slowThreshold.Load() {
            logger := logx.WithContext(ctx).WithDuration(elapsed)         // context + duration
            logger.Slowf("[RPC] ok - slowcall - %s", serverName)          // "slowcall" keyword
        }
    }
    return err
}

func SetSlowThreshold(threshold time.Duration) {   // line 59
    slowThreshold.Set(threshold)
}
```

**Key go-zero RPC pattern**:
| Element | Detail |
|---|---|
| Default threshold | **500ms** |
| Measurement | `timex.Now()` / `timex.Since()` |
| Context | `logx.WithContext(ctx)` — always |
| Duration | `logx.WithDuration(elapsed)` — always |
| Slow log method | `logx.Slowf()` — always |
| Slow keyword | `"[RPC] ok - slowcall"` |
| Threshold type | `syncx.ForAtomicDuration` — atomic, globally settable |
| Config API | `SetSlowThreshold(threshold)` — package-level |

### 2. gormx Logger (project's own go-zero-aligned pattern)

**File**: `common/gormx/logger.go`, lines 88-113 (`Trace` method)

```
elapsed := time.Since(begin)                                   // line 97
// ...
case elapsed > c.cfg.SlowThreshold && c.cfg.SlowThreshold != 0 && c.cfg.LogLevel >= logger.Warn:
    sql, rows := fc()
    logx.WithContext(ctx).WithDuration(elapsed).Slowf(         // context + duration + Slowf
        "[gorm] [rows:%s] [SLOW] %s", formatRows(rows), sql)   // [SLOW] keyword
```

**Key gormx pattern**:
| Element | Detail |
|---|---|
| Default threshold | **200ms** (`common/gormx/config.go:24`, `common/gormx/logger.go:43`) |
| Measurement | `time.Since(begin)` (native time, not timex) |
| Context | `logx.WithContext(ctx)` — always |
| Duration | `logx.WithDuration(elapsed)` — always (via chain) |
| Slow log method | `logx.Slowf()` — always |
| Slow keyword | `"[SLOW]"` |
| Config | YAML `SlowThreshold: 200ms` in DB section |

### 3. Other `timex.Since` Usage in Project (no slow threshold)

These use `timex.Since` for duration logging but **not** for slow detection:

| File | Line | Pattern |
|---|---|---|
| `common/asynqx/asynqTaskServer.go` | 78 | `duration := timex.Since(startTime)` → `logx.WithContext(ctx).WithDuration(duration).Debug/Errorf(...)` |
| `common/mcpx/wrapper.go` | 140,225,228 | `logx.WithContext(ctx).WithDuration(timex.Since(start)).Infof/Errorf(...)` |
| `common/mqttx/dispatcher.go` | 132 | `d.metrics.Add(stat.Task{Duration: timex.Since(startTime)})` |
| `common/wsx/client.go` | 305 | `c.metrics.Add(stat.Task{Duration: timex.Since(start)})` |

None of these implement slow threshold comparison — they only record duration.

### 4. SlowThreshold in YAML Configs (gormx only)

Found in 3 YAML files, all in **DB** (gormx) sections:
- `app/ispagent/etc/ispagent.yaml:19` — `SlowThreshold: 200ms`
- `app/file/etc/file.yaml:58` — `SlowThreshold: 200ms`
- `app/djicloud/etc/djicloud.yaml:35` — `SlowThreshold: 200ms`

**None** of these are RPC SlowThreshold — all are gormx DB SlowThreshold. The go-zero zrpc client has a hardcoded 500ms default with no project-specific override.

### Summary: go-zero Canonical Slow Log Pattern

```
logx.WithContext(ctx).WithDuration(elapsed).Slowf("[PREFIX] <action> - slowcall - <detail>", ...)
```

**Key elements**:
1. `logx.WithContext(ctx)` — always propagate context
2. `logx.WithDuration(elapsed)` — always chain duration (it formats it)
3. `Slowf` — always use slow log method (not Errorf/Infof)
4. `slowcall` — go-zero RPC uses this keyword; gormx uses `[SLOW]`
5. Threshold is configurable (atomic for RPC, struct field for gormx)
