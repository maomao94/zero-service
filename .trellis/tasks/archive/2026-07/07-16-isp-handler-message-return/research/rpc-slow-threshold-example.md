# Research: RPC Proxy slowThreshold Usage in Project

- **Query**: How RPC proxies in this project use `slowThreshold` — find an example
- **Scope**: internal + go-zero library
- **Date**: 2026-07-16

## Findings

### The short answer

This project does **not** explicitly configure `slowThreshold` for RPC proxies. The go-zero zrpc framework has a built-in `DurationInterceptor` with a hardcoded 500ms default, and **no project file calls `SetSlowThreshold`** to override it.

### go-zero RPC Built-in Slow Call Detection

**File**: `github.com/zeromicro/go-zero@v1.10.0/zrpc/internal/clientinterceptors/durationinterceptor.go`

The `DurationInterceptor` is **automatically applied** to every gRPC unary client call. It:

1. Measures call duration via `timex.Now()` / `timex.Since()`
2. Checks `elapsed > slowThreshold.Load()` (default: **500ms**)
3. On slow calls: `logx.WithContext(ctx).WithDuration(elapsed).Slowf("[RPC] ok - slowcall - %s", serverName)`
4. Has a package-level `SetSlowThreshold(threshold time.Duration)` API — **never called in this project**

### Example RPC Client Usage in This Project

**File**: `gtw/internal/svc/servicecontext.go`, lines 59-62

```go
ZeroRpcCli: zerorpc.NewZerorpcClient(zrpc.MustNewClient(c.ZeroRpcConf,
    zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor)).Conn()),
FileRpcCLi: file.NewFileRpcClient(zrpc.MustNewClient(c.FileRpcConf,
    zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor)).Conn()),
```

All 16+ usages of `zrpc.MustNewClient` in this project follow the same pattern:
1. Pass a `RpcClientConf` struct (with `Endpoints` or `Etcd` config)
2. Optionally add interceptors
3. Get `.Conn()` and pass to generated client constructor

### RPC Config Example

**File**: go-zero `zrpc/config.go`

```go
type RpcClientConf struct {
    Etcd          discov.EtcdConf `json:",optional,inherit"`
    Endpoints     []string        `json:",optional"`
    Target        string          `json:",optional"`
    App           string          `json:",optional"`
    Token         string          `json:",optional"`
    NonBlock      bool            `json:",default=true"`
    Timeout       int64           `json:",default=2000"`
    KeepaliveTime time.Duration   `json:",optional"`
    Middlewares   ClientMiddlewaresConf
    BalancerName  string `json:",default=p2c_ewma"`
}
```

There is **no `SlowThreshold` field** in `RpcClientConf`. The slow threshold is managed by the interceptor layer, not the config layer.

### How the Slow Call Log Looks (go-zero RPC)

When a gRPC call exceeds 500ms, the output is:

```
[RPC] ok - slowcall - direct://127.0.0.1:21003/zerorpc.Zerorpc/SomeMethod
```

Or with request/reply content:

```
[RPC] ok - slowcall - direct://127.0.0.1:21003/zerorpc.Zerorpc/SomeMethod - request_body - reply_body
```

### Comparison: gnetx vs go-zero RPC Slow Log

| Aspect | gnetx (current) | go-zero RPC (canonical) |
|---|---|---|
| Default threshold | **50ms** | **500ms** |
| Context in log | ❌ `logx.Slowf(...)` | ✅ `logx.WithContext(ctx).WithDuration(elapsed).Slowf(...)` |
| Duration in log | ✅ as `%s` format arg | ✅ via `.WithDuration()` chain |
| Key word | `"slow handler"` | `"slowcall"` |
| Prefix | `[gnetx]` | `[RPC]` |
| Dynamic threshold | ❌ fixed per instance | ✅ `SetSlowThreshold()` atomic global |
| Throws away ctx | Yes (ctx available but unused) | No |

### YAML `SlowThreshold` Fields (NOT RPC)

The only `SlowThreshold` found in YAML configs is for **gormx DB**, not RPC:
- `app/ispagent/etc/ispagent.yaml:19` — `SlowThreshold: 200ms` (DB section)
- `app/file/etc/file.yaml:58` — `SlowThreshold: 200ms` (DB section)
- `app/djicloud/etc/djicloud.yaml:35` — `SlowThreshold: 200ms` (DB section)

## Caveats

- There is no `SetSlowThreshold` call anywhere in this project — all RPC slow detection uses the 500ms default
- The project's `zrpc.MustNewClient` calls never pass a `SlowThreshold` option because `RpcClientConf` doesn't have one
- gnetx and go-zero RPC use different slow detection patterns — they're independent systems
