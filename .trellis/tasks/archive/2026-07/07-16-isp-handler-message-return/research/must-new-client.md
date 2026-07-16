# Research: MustNewClient in common/gnetx

- **Query**: `MustNewClient` definition and callers in `common/gnetx/`
- **Scope**: internal
- **Date**: 2026-07-16

## Findings

### Definition

**File**: `common/gnetx/client.go`, lines 44-51

```go
func MustNewClient(address string, opts ...ClientOption) *Client {
    cli, err := NewClient(address, opts...)
    if err != nil {
        panic("gnetx: MustNewClient " + address + ": " + err.Error())
    }
    proc.AddShutdownListener(func() { cli.Close() })
    return cli
}
```

**Key characteristics**:
- Panics on construction failure (constructor pattern, must succeed)
- Registers a shutdown listener via `proc.AddShutdownListener` to call `cli.Close()` on process exit
- Returns `*Client` pointer directly
- Accepts variadic `ClientOption` functional options

### Documentation Reference

**File**: `common/gnetx/doc.go`, lines 42-46

```go
// cli := gnetx.MustNewClient("127.0.0.1:9000",
//     gnetx.WithClientCodec(myCodec),
//     gnetx.WithClientHandler(myHandler),
//     gnetx.WithClientMaxFrameLength(1<<20),
// )
```

### Callers Within gnetx Package

No callers within the gnetx package itself (only the definition and doc example).

### Callers Outside gnetx Package

**NOT directly called outside gnetx by any production code**. The primary gnetx consumer (`app/ispagent`) likely constructs clients differently (via `NewClient` or through a wrapped factory).

### Other MustNewClient Patterns in Project (for comparison)

All other `MustNewClient` patterns in the project follow go-zero's convention (panic on failure + shutdown listener):

| Package | File | Pattern |
|---|---|---|
| `zrpc` (go-zero) | *vendor* | `zrpc.MustNewClient(conf, opts...)` → creates gRPC client, panics on error |
| `common/wsx` | `client.go:48` | `MustNewClient(cfg Config, opts ...ClientOption) Client` |
| `common/mqttx` | `client.go:71` | `MustNewClient(cfg MqttConfig, opts ...ClientOption) Client` |
| `common/iec104` | `client/core.go:93` | `MustNewClient(cfg ClientConfig, opts ...ClientOption) *Client` |
| `common/djisdk` | `client.go:36` | `MustNewClient(cfg Config, opts ...ClientOption) *Client` |
| `common/dockerx` | `dockerx.go:11` | `MustNewClient(ops ...client.Opt) *client.Client` |

### gnetx MustNewClient vs go-zero zrpc.MustNewClient

| Aspect | gnetx.MustNewClient | zrpc.MustNewClient |
|---|---|---|
| Signature | `(address string, opts ...ClientOption) *Client` | `(c RpcClientConf, opts ...ClientOption) Client` |
| Config | Address string + func options | `RpcClientConf` struct |
| Shutdown | `proc.AddShutdownListener(func() { cli.Close() })` | Handled internally |
| Return | Concrete pointer | Interface |
| Slow integration | Built-in via `SlowHandlerThreshold` | Built-in via `DurationInterceptor` with `slowThreshold` |

## Caveats

- `MustNewClient` is the public constructor; `NewClient` is also exported and returns `(*Client, error)` without panicking
- The shutdown listener is added unconditionally — even if the client is later explicitly `Close()`d
- No gnetx client users were found in production code calling `MustNewClient` directly from outside the package (the ispagent app constructs its own TCP client differently)
