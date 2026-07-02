# Research: gnet (panjf2000) Design & Extension Points

- **Query**: Produce a thorough design-level understanding of gnet's event-driven model and its extension points, so we can build an "out-of-the-box" TCP framework (gnetx) on top of it.
- **Scope**: external (GitHub source `panjf2000/gnet` dev branch, `gnet-io/gnet-examples` v2 branch, pkg.go.dev v2.9.8)
- **Date**: 2026-07-02
- **Versions referenced**: module `github.com/panjf2000/gnet/v2` v2.9.8 (pkg.go.dev, published 2026-05-25); latest GitHub *release* with notes is v2.9.0 "Millennium Actress" (2025-06-11). Source fetched from `dev` branch.
- **Go version**: `go.mod` declares `go 1.20`; README badge `>=1.20`. Safe on Go 1.26 (see §11).

---

## 0. Repo layout (what lives where)

Root package `gnet` (file `gnet.go`) exposes the entire public API. Platform split via build tags:

| File | Role |
|---|---|
| `gnet.go` | Public types: `Action`, `Engine`, `EventHandler`, `BuiltinEventEngine`, `Conn`, `Reader`, `Writer`, `Socket`, `EventLoop`, `Client`, `AsyncCallback`, `Runnable`, `RegisteredResult`; `Run`/`Rotate`/`Stop`; `MaxStreamBufferCap`. |
| `options.go` | `Options` struct + all `With*` option funcs. |
| `load_balancer.go` | `LoadBalancing` enum (RoundRobin / LeastConnections / SourceAddrHash) + internal `loadBalancer` impls. |
| `acceptor_*.go`, `listener_*.go`, `engine_*.go`, `eventloop_*.go`, `connection_*.go`, `client_*.go` | Platform split: `*_unix.go` (epoll/kqueue), `connection_bsd.go`, `connection_linux.go`, `*_windows.go`. |
| `context.go` | `NewContext`/`FromContext`/`NewNetConnContext`/`FromNetConnContext`/`NewNetAddrContext`/`FromNetAddrContext` — carry a value into a `Conn` via `context.Context`. |
| `pkg/` | **No `codec` sub-package.** Sub-packages: `bs` (byte-string helpers), `buffer` (ring/linked-list/elastic buffers), `errors`, `io` (writev), `logging` (zap-backed), `math` (power-of-two rounding), `netpoll` (epoll/kqueue poller), `pool` (goroutine pool via `ants`), `queue` (lock-free task queue), `socket` (syscalls). |
| `internal/gfd` | "gnet file descriptor" encoding — packs (event-loop index, fd) into a uint64. |

**Examples** live in a separate repo `github.com/gnet-io/gnet-examples` (branch `v2`): `echo_tcp`, `echo_udp`, `echo_uds`, `http`, `push`, `simple_protocol`, `websocket`.

---

## 1. gnet core event model (reactor threads, event loop, epoll/kqueue)

gnet is a **reactor pattern**: a fixed set of event-loop goroutines, each driving one epoll/kqueue poller. It is *not* the Go `net` model (one goroutine per connection blocking on read/write).

### 1.1 Topology

```
                 ┌──────────── acceptor (main goroutine) ────────────┐
   Listener fd ─►│ unix.Accept loop → new conn fd → lb.next(addr)    │
                 └────────────────────────┬─────────────────────────┘
                                          │ assigns conn to ONE event-loop
            ┌─────────────────────────────┼─────────────────────────────┐
            ▼                             ▼                              ▼
   ┌── event-loop 0 ──┐         ┌── event-loop 1 ──┐          ┌── event-loop N ──┐
   │ poller (epoll)   │         │ poller (epoll)   │          │ poller (epoll)   │
   │ conns (connMatrix)│        │ conns            │          │ conns            │
   │ read/write/close │         │ read/write/close │          │ read/write/close │
   │ ticker goroutine │         │ ticker goroutine │          │ ticker goroutine │
   └──────────────────┘         └──────────────────┘          └──────────────────┘
```

- `determineEventLoops` (`gnet.go`): if `Multicore` → `runtime.NumCPU()`; `NumEventLoop` (if >0) overrides; capped at `gfd.EventLoopIndexMax`.
- `eventloop` struct (`eventloop_unix.go`): `listeners`, `idx`, `engine`, `poller *netpoll.Poller`, `buffer []byte` (read scratch, default 64KB), `connections connMatrix`, `eventHandler EventHandler`.
- **Load balancing** (`load_balancer.go`) picks which event-loop owns a *newly accepted* connection:
  - `RoundRobin` (default) — `eventLoops[nextIndex % size]`.
  - `LeastConnections` — min `countConn()`.
  - `SourceAddrHash` — `crc32(remoteAddr) % size` → **same client always lands on the same loop** (useful for session affinity / avoiding cross-loop state).

### 1.2 Edge-triggered vs level-triggered

- Default: **level-triggered (LT)**. On readable event, `read()` reads once into `el.buffer`, calls `OnTraffic`, then appends leftovers to `c.inboundBuffer` and loops again only while `c.isEOF || (isET && recv < chunk)`.
- `WithEdgeTriggeredIO(true)` / `WithEdgeTriggeredIOChunk(n)`: switches to ET, registers `AddReadWrite`, and drains in chunks of `1<<20` (1MB default, rounded to power of two). To avoid starving other fds, ET read/write re-arms via `poller.Trigger(queue.Low/HighPriority, el.read0/write0, c)` when the buffer fills. **ET is stream-protocol only; auto-disabled for UDP.**

### 1.3 The read path (`eventloop_unix.go`, `eventloop.read`)

```go
n, err := unix.Read(c.fd, el.buffer)      // single syscall per LT event
...
c.buffer = el.buffer[:n]
action := el.eventHandler.OnTraffic(c)    // synchronous, on the loop goroutine
...
_, _ = c.inboundBuffer.Write(c.buffer)    // unread bytes spill into the conn's inbound ring buffer
c.buffer = c.buffer[:0]
```

Key implication: **`OnTraffic` runs on the event-loop goroutine, synchronously with the read.** Whatever the user does in `OnTraffic` blocks every other connection on the same loop.

### 1.4 The write path (`eventloop.write`)

- `c.Write(buf)` (synchronous, on-loop) appends to `c.outboundBuffer` and, under LT, calls `poller.ModReadWrite` so the loop gets writability events and `write()` flushes via `gio.Writev`/`unix.Write`.
- `c.Flush()` forces a flush; `c.OutboundBuffered()` reports pending bytes.
- When the outbound buffer drains under LT, `poller.ModRead` removes writability monitoring.
- `AsyncWrite(buf, cb)` / `AsyncWritev(bs, cb)` are the **concurrency-safe** way to write from another goroutine: they enqueue the write onto the owning event-loop's poller queue; the actual `unix.Write` happens on-loop, and `cb` is invoked on-loop afterwards.

### 1.5 Ticker

- Each event-loop starts its own `ticker(ctx)` goroutine (`eventloop_unix.go`) that calls `el.eventHandler.OnTick()` in a loop, sleeping `delay` between calls via `time.Timer`.
- **Gotcha**: with N event-loops, `OnTick` is invoked **N times per interval** (once per loop, concurrently). Enable with `WithTicker(true)`. `OnTick` returns `(delay time.Duration, action Action)`; returning `Shutdown` signals the engine to stop.

### 1.6 Accept path & SO_REUSEPORT

- On Linux/DragonFly/FreeBSD, `WithReusePort(true)` + multiple loops lets each loop own a listener fd and the kernel load-balances accepts. On macOS/*BSD (no LB-capable SO_REUSEPORT) gnet **forces `ReusePort=false`** when `Multicore || NumEventLoop>1` (see `createListeners` in `gnet.go`) — so on Darwin multi-loop still goes through a single acceptor that hands fds to loops.
- UDP **requires** `ReusePort=true` and disables ET.

---

## 2. EventHandler interface — every callback, signature, when fired, thread context

From `gnet.go` (v2.9.8, `EventHandler` at pkg.go.dev line ~474):

```go
type EventHandler interface {
    OnBoot(eng Engine) (action Action)
    OnShutdown(eng Engine)
    OnOpen(c Conn) (out []byte, action Action)
    OnClose(c Conn, err error) (action Action)
    OnTraffic(c Conn) (action Action)
    OnTick() (delay time.Duration, action Action)
}
```

`BuiltinEventEngine` is an empty embedding stub you embed so you only override what you need:

```go
type BuiltinEventEngine struct{}
func (*BuiltinEventEngine) OnBoot(_ Engine) (action Action)         { return }
func (*BuiltinEventEngine) OnShutdown(_ Engine)                     {}
func (*BuiltinEventEngine) OnOpen(_ Conn) (out []byte, action Action) { return }
func (*BuiltinEventEngine) OnClose(_ Conn, _ error) (action Action) { return }
func (*BuiltinEventEngine) OnTraffic(_ Conn) (action Action)        { return }
func (*BuiltinEventEngine) OnTick() (delay time.Duration, action Action) { return }
```

| Callback | Signature | When fired | Thread context | Return `Action` |
|---|---|---|---|---|
| `OnBoot` | `(Engine) Action` | Engine ready, before accepting. | **Main goroutine** (the one calling `gnet.Run`). | `Shutdown` aborts startup. |
| `OnShutdown` | `(Engine)` | After all event-loops and connections are closed. | Main goroutine. | (none) |
| `OnOpen` | `(Conn) ([]byte, Action)` | New conn accepted/enrolled, before reading. `out` is sent to peer immediately (avoid large writes here). | **Owning event-loop goroutine.** | `Close`/`Shutdown`. |
| `OnClose` | `(Conn, error) Action` | Conn closed; `err` is the last known error (e.g. `io.EOF` on clean peer close, `*os.SyscallError` on read/write failure). | Owning event-loop goroutine. | `Close` is no-op; `Shutdown` stops engine. |
| `OnTraffic` | `(Conn) Action` | Socket has readable data. **This is where you consume the inbound buffer** via `Reader` methods. | Owning event-loop goroutine. | `Close`/`Shutdown`. |
| `OnTick` | `() (time.Duration, Action)` | Immediately after engine start, then every `delay`. | **Per event-loop ticker goroutine** (one per loop → called N× per tick under multicore). | `delay` controls next fire; `Shutdown` stops. |

`Action` enum (`gnet.go`): `None` (iota), `Close` (close the conn), `Shutdown` (stop the engine). Returned from callbacks to manage conn/engine state.

**There is no `OnDetach`/`React` callback.** Detach-style fd migration is not exposed. The only "move a conn between loops" affordance is `Engine.Register`/`EventLoop.Enroll` (for *external* net.Conn → gnet.Conn adoption, see §6).

---

## 3. gnet.Conn API — full method list grouped by category

`Conn` is an interface composed of three sub-interfaces plus its own methods (`gnet.go`):

```go
type Conn interface {
    Reader   // NOT concurrency-safe — call only inside EventHandler callbacks
    Writer   // mixed: sync methods NOT safe; Async* ARE safe
    Socket   // all concurrency-safe
    Context() any                  // NOT safe — on-loop only
    EventLoop() EventLoop         // safe
    SetContext(ctx any)           // NOT safe — on-loop only
    LocalAddr() net.Addr          // NOT safe — on-loop only
    RemoteAddr() net.Addr         // NOT safe — on-loop only
    Wake(callback AsyncCallback) error               // safe
    CloseWithCallback(callback AsyncCallback) error  // safe
    Close() error                                     // safe (implements net.Conn)
    SetDeadline(time.Time) error                      // safe
    SetReadDeadline(time.Time) error                  // safe
    SetWriteDeadline(time.Time) error                 // safe
}
```

### 3.1 Reader (inbound buffer; **all on-loop only**)

```go
type Reader interface {
    io.Reader        // Read(p []byte) (n int, err error) — copies out, advances
    io.WriterTo      // WriteTo(w io.Writer) (n int64, err error)
    Next(n int) (buf []byte, err error)      // advance+return n bytes; n=-1 ⇒ all; buf invalid across goroutines
    Peek(n int) (buf []byte, err error)      // return n bytes WITHOUT advancing; valid until Discard
    Discard(n int) (discarded int, err error)
    InboundBuffered() int                    // bytes currently available
}
```
- `Peek`/`Next` return `(0, io.ErrShortBuffer)` when fewer than `n` bytes are available — **this is the primary primitive for packet framing** (see §4 & §9).
- Returned slices point into gnet's internal ring buffer; **do not retain them across goroutines or past the next `Discard`/`Read`** — copy if you need to keep.

### 3.2 Writer (outbound; mixed safety)

```go
type Writer interface {
    io.Writer        // Write(p) (n int, err error) — NOT safe, on-loop only
    io.ReaderFrom    // ReadFrom(r) (n int64, err error) — NOT safe
    SendTo(buf []byte, addr net.Addr) (n int, err error) // UDP only, NOT safe
    Writev(bs [][]byte) (n int, err error)   // NOT safe, on-loop only
    Flush() error                            // NOT safe, on-loop only
    OutboundBuffered() int                   // NOT safe, on-loop only
    AsyncWrite(buf []byte, callback AsyncCallback) error     // SAFE — post to owning loop
    AsyncWritev(bs [][]byte, callback AsyncCallback) error  // SAFE — post to owning loop
}
```
- `AsyncCallback = func(c Conn, err error) error` — runs **on the event-loop** after the async op; must not block. For UDP the `Conn` may already be released, so don't touch it.
- Synchronous `Write`/`Writev` buffer data and (under LT) re-arm writability; actual `unix.Write` happens later in `eventloop.write`. `Flush` forces it now.

### 3.3 Socket (fd control; **all concurrency-safe**)

```go
type Socket interface {
    Fd() int
    Dup() (int, error)
    SetReadBuffer(size int) error
    SetWriteBuffer(size int) error
    SetLinger(secs int) error
    SetKeepAlivePeriod(d time.Duration) error
    SetKeepAlive(enabled bool, idle, intvl time.Duration, cnt int) error
    SetNoDelay(noDelay bool) error
}
```

### 3.4 Control / lifecycle (on Conn itself)

- `Wake(cb)` — **safe**; triggers a synthetic `OnTraffic` for this conn on its owning loop (used when you left data in the inbound buffer and need another read cycle — see §9 server example).
- `Close()` / `CloseWithCallback(cb)` — **safe** from any goroutine.
- `SetDeadline`/`SetReadDeadline`/`SetWriteDeadline` — **safe** (net.Conn-compatible).
- `Context()`/`SetContext()` — **on-loop only**; the canonical place to stash per-conn state (codec instance, session, last-active time, pending-request map).
- `EventLoop()` — returns the owning `EventLoop` (safe to hold the reference).

---

## 4. Codec / packet framing — what gnet provides natively vs what you build

### 4.1 What gnet provides

**gnet provides NO built-in codec.** There is no `pkg/codec`, no `WithCodec` option, no length-prefix / delimiter / fixed-length framer. The connection hands you a raw byte stream via the `Reader` interface (`Peek`/`Next`/`Discard`/`Read`). Framing is 100% the application's responsibility.

The only framing-adjacent primitives:
- `Peek(n)` / `Discard(n)` — non-consuming look-ahead + advance, the building blocks for any length-prefix codec.
- `InboundBuffered()` — how many bytes are available right now.
- `Next(-1)` — "give me everything currently buffered" (used by the echo example).
- `MaxStreamBufferCap` (64KB default; configurable via `WithReadBufferCap`/`WithWriteBufferCap`, rounded to a power of two ≥ `ring.DefaultBufferSize`) — bounds the per-conn ring buffer.

### 4.2 What you build (the canonical pattern from `simple_protocol`)

The `simple_protocol/protocol/proto.go` example defines a `SimpleCodec` with `Encode`/`Decode`/`Unpack`. The wire format is `[2B magic 1314][4B body len][body]`. `Decode` uses **only `c.Peek` + `c.Discard`**:

```go
func (codec SimpleCodec) Decode(c gnet.Conn) ([]byte, error) {
    bodyOffset := magicNumberSize + bodySize // 6
    buf, err := c.Peek(bodyOffset)
    if err != nil {
        if errors.Is(err, io.ErrShortBuffer) { return nil, ErrIncompletePacket }
        return nil, err
    }
    if !bytes.Equal(magicNumberBytes, buf[:magicNumberSize]) { return nil, errors.New("invalid magic number") }
    bodyLen := binary.BigEndian.Uint32(buf[magicNumberSize:bodyOffset])
    msgLen := bodyOffset + int(bodyLen)
    buf, err = c.Peek(msgLen)            // 2nd Peek for the whole frame
    if err != nil {
        if errors.Is(err, io.ErrShortBuffer) { return nil, ErrIncompletePacket }
        return nil, err
    }
    body := make([]byte, bodyLen)
    copy(body, buf[bodyOffset:msgLen])
    _, _ = c.Discard(msgLen)             // advance past the consumed frame
    return body, nil
}
```

The codec instance is stored on the conn via `c.SetContext(new(SimpleCodec))` in `OnOpen`, and retrieved with `c.Context().(*SimpleCodec)` in `OnTraffic` (see §9). This `Peek → check len → Peek full → copy → Discard` dance is **the** idiomatic gnet framing pattern and is exactly what gnetx must package into reusable codecs (length-prefix, delimiter, fixed-length).

### 4.3 Partial-frame handling

gnet fires `OnTraffic` once per readable event; unread bytes remain in `c.inboundBuffer`. If a frame spans two read events, the codec's `Peek` returns `io.ErrShortBuffer` and you simply `return gnet.None` — the leftover stays buffered and the next readable event re-enters `OnTraffic` with the accumulated bytes. If you consumed `batchRead` frames and there's still data, call `c.Wake(nil)` to re-trigger `OnTraffic` immediately (see §9).

---

## 5. Server bootstrap API (`gnet.Run`, `gnet.Rotate`, options)

### 5.1 Entry points (`gnet.go`)

```go
func Run(eventHandler EventHandler, protoAddr string, opts ...Option) error
func Rotate(eventHandler EventHandler, addrs []string, opts ...Option) error  // multi-address (v2.5.0+)
func Stop(ctx context.Context, protoAddr string) error                       // DEPRECATED — use Engine.Stop
```

- `protoAddr` scheme: `tcp` / `tcp4` / `tcp6` / `udp` / `udp4` / `udp6` / `unix`. `tcp` assumed when omitted. Example: `"tcp://:9000"`, `"unix:///tmp/socket"`.
- `Run`/`Rotate` **block** until the engine stops (error or `Shutdown` action). They build listeners (`createListeners`), then call internal `run(...)`.
- There is **no `NewServer` exported**. `Run`/`Rotate` are the server bootstrap. The "server" handle is the `Engine` you capture in `OnBoot` (see §5.3).

### 5.2 Options (`options.go`) — full `Options` struct

```go
type Options struct {
    LB LoadBalancing            // RoundRobin | LeastConnections | SourceAddrHash (server-only)
    ReuseAddr bool              // SO_REUSEADDR (server-only)
    ReusePort bool              // SO_REUSEPORT (server-only; auto-disabled on macOS/*BSD multi-loop)
    MulticastInterfaceIndex int // UDP multicast iface (server-only)
    BindToDevice string         // SO_BINDTODEVICE, Linux only (server-only)
    Multicore bool              // true ⇒ NumCPU event-loops
    NumEventLoop int            // overrides Multicore; capped by gfd.EventLoopIndexMax
    ReadBufferCap int           // per-conn read ring buffer; default 64KB; rounded to pow2
    WriteBufferCap int          // per-conn outbound static buffer; default 64KB; rounded to pow2
    LockOSThread bool           // pin each loop goroutine to an OS thread (max 10000 loops)
    Ticker bool                 // enable OnTick
    TCPKeepAlive time.Duration  // SO_KEEPALIVE + TCP_KEEPIDLE
    TCPKeepInterval time.Duration // TCP_KEEPINTVL (v2.9.0+)
    TCPKeepCount int              // TCP_KEEPCNT   (v2.9.0+)
    TCPNoDelay TCPSocketOpt       // TCPNoDelay (default) | TCPDelay
    SocketRecvBuffer int          // SO_RCVBUF
    SocketSendBuffer int          // SO_SNDBUF
    LogPath string                // file path for default zap logger
    LogLevel logging.Level
    Logger logging.Logger         // override default zap logger
    EdgeTriggeredIO bool          // ET mode (stream only)
    EdgeTriggeredIOChunk int      // ET read/write chunk (default 1MB, pow2)
}
```

With* setters: `WithMulticore`, `WithNumEventLoop`, `WithLoadBalancing`, `WithReusePort`, `WithReuseAddr`, `WithTCPKeepAlive`/`WithTCPKeepInterval`/`WithTCPKeepCount`, `WithTCPNoDelay`, `WithSocketRecvBuffer`/`WithSocketSendBuffer`, `WithReadBufferCap`/`WithWriteBufferCap`, `WithLockOSThread`, `WithTicker`, `WithLogPath`/`WithLogLevel`/`WithLogger`, `WithMulticastInterfaceIndex`, `WithBindToDevice`, `WithEdgeTriggeredIO`/`WithEdgeTriggeredIOChunk`, `WithOptions`.

### 5.3 Engine handle (captured in `OnBoot`)

```go
type Engine struct{ eng *engine }   // opaque
func (e Engine) Validate() error
func (e Engine) CountConnections() int
func (e Engine) Stop(ctx context.Context) error              // graceful; polls every 500ms
func (e Engine) Register(ctx context.Context) (<-chan RegisteredResult, error)  // adopt external conn (v2.8.0+)
func (e Engine) Dup() (fd int, err error)                    // dup listener fd (single-listener only)
func (e Engine) DupListener(network, addr string) (int, error) // v2.8.0+
```

Typical server: store `eng` in `OnBoot`, call `eng.Stop(ctx)` later (signal handler) for graceful shutdown. `Stop` waits for all loops/conns to close before returning.

### 5.4 Idiomatic server skeleton (echo example, abridged)

```go
type echoServer struct {
    gnet.BuiltinEventEngine
    eng gnet.Engine
    // ...app fields...
}
func (es *echoServer) OnBoot(eng gnet.Engine) gnet.Action { es.eng = eng; return gnet.None }
func (es *echoServer) OnTraffic(c gnet.Conn) gnet.Action {
    buf, _ := c.Next(-1)   // all currently buffered bytes
    c.Write(buf)           // echo back synchronously
    return gnet.None
}
// gnet.Run(echo, "tcp://:9000", gnet.WithMulticore(true))
```

---

## 6. Client bootstrap API (`gnet.NewClient`, `Client.Dial`/`Enroll`)

Yes — gnet has a real active-client. As of **v2.9.0** the client also runs over multiple event-loops (previously single-loop).

```go
type Client struct{ /* unexported */ }
func NewClient(eh EventHandler, opts ...Option) (*Client, error)        // opts: most server opts apply (LB etc.)
func (cli *Client) Start() error                                        // start the client event-loop(s)
func (cli *Client) Stop() error                                         // stop event-loop(s)
func (cli *Client) Dial(network, address string) (Conn, error)          // like net.Dial; blocks until connected
func (cli *Client) DialContext(network, address string, ctx any) (Conn, error) // ctx any → retrievable via Conn.Context (v2.4.0+)
func (cli *Client) Enroll(c net.Conn) (Conn, error)                     // adopt an established net.Conn into gnet (v2.1.0+)
func (cli *Client) EnrollContext(c net.Conn, ctx any) (Conn, error)     // v2.4.0+
```

Lifecycle: `NewClient` → `Start` → `Dial`/`Enroll` (returns a `gnet.Conn` that fires `OnOpen`/`OnTraffic`/`OnClose` on the client's event-loop) → `Stop`. The same `EventHandler` interface is used; `OnBoot`/`OnShutdown` fire on the client engine. `DialContext`'s `ctx any` (note: **not** `context.Context` — it's an opaque value) is later returned by `Conn.Context()` — useful to tag a conn with a request/correlation ID at dial time.

`EventLoop.Register(ctx, addr)` / `EventLoop.Enroll(ctx, c)` (v2.8.0+) are the per-loop variants that pin the new conn to a *specific* event-loop (used for affinity / accepting an external conn onto a chosen loop). `Engine.Register(ctx)` picks the loop via `LB` and is the public entry.

Internally (`eventloop_unix.go`), `enroll` does the `net.Dial` on the **ants goroutine pool** (`goroutine.DefaultWorkerPool.Submit`), dups the fd, then `poller.Trigger`s `el.register` onto the owning loop — so `Dial` is safe to call concurrently and does not block the loop.

---

## 7. Request/response correlation — gnet has none; build it yourself

gnet is **purely event-driven**. There is no request/response, no future/promise, no "send-and-await" primitive. `OnTraffic` just tells you "bytes arrived". To implement request/response you layer it on top:

### 7.1 Server side (simplest: synchronous in `OnTraffic`)

```go
func (h *handler) OnTraffic(c gnet.Conn) gnet.Action {
    req, _ := codec.Decode(c)          // one framed request
    resp := handle(req)                // MUST be fast & non-blocking
    c.Write(codec.Encode(resp))        // synchronous reply on-loop
    return gnet.None
}
```
If `handle` is expensive (DB, downstream RPC), offload to a worker pool and reply via `c.AsyncWrite(codec.Encode(resp), nil)` from the worker goroutine — `AsyncWrite` is the **only** safe way to write from off-loop.

### 7.2 Client side (request/response matching)

You must build a pending-request registry yourself, typically stashed on `Conn.Context()` or a shared map keyed by conn:

```go
type session struct {
    mu      sync.Mutex
    pending map[uint64]chan Response   // requestID → result channel
}
// caller goroutine:
ch := make(chan Response, 1)
sess.pending[reqID] = ch
c.AsyncWrite(codec.Encode(reqWithID(reqID, payload)), nil)  // or c.Write on-loop
select {
case resp := <-ch: ...
case <-time.After(timeout): delete(sess.pending, reqID); ...
}
// in OnTraffic:
for {
    resp, err := codec.Decode(c); if err == ErrIncompletePacket { break }
    reqID := resp.ID
    if ch, ok := sess.pending[reqID]; ok { ch <- resp; delete(sess.pending, reqID) }
}
```

Caveats gnet imposes: the `pending` map is accessed from both the caller goroutine and the event-loop goroutine (in `OnTraffic`), so it needs its own mutex (it is *not* covered by gnet's "on-loop only" rule because you touch it from off-loop too). `Conn.Context()`/`SetContext()` themselves must only be read/written on-loop, so fetch the session pointer once in `OnTraffic` and operate on its mutex-guarded map. This is exactly the machinery gnetx must provide.

---

## 8. Concurrency model & gotchas

### 8.1 The one rule: a connection is pinned to one event-loop goroutine

Every `OnOpen`/`OnTraffic`/`OnClose` for a given conn fires **on the same event-loop goroutine**. All `Reader` methods (`Next`/`Peek`/`Discard`/`Read`/`InboundBuffered`) and the synchronous `Writer` methods (`Write`/`Writev`/`Flush`/`OutboundBuffered`), plus `Context`/`SetContext`/`LocalAddr`/`RemoteAddr`, are **not concurrency-safe** and may only be called inside `EventHandler` callbacks (i.e. on-loop).

### 8.2 What you CAN call from any goroutine

`AsyncWrite`/`AsyncWritev` (with `AsyncCallback`), `Wake`, `Close`, `CloseWithCallback`, `SetDeadline`/`SetReadDeadline`/`SetWriteDeadline`, all `Socket` methods, `EventLoop().Register/Enroll/Execute`, `Engine.Stop/Register/CountConnections`. These post tasks onto the owning loop's poller queue (`poller.Trigger`).

### 8.3 Blocking inside `OnTraffic` is catastrophic

`OnTraffic` runs synchronously inside `eventloop.read`. A slow handler stalls **every other connection on the same loop** (and under ET, can starve re-arming reads). Rules of thumb:
- Never do blocking I/O (sync DB call, HTTP fetch, file read) in `OnTraffic`/`OnOpen`/`OnTick`.
- Offload heavy work to a worker pool; reply via `AsyncWrite`.
- `Runnable.Run` (passed to `EventLoop.Execute`) has the same "don't block" rule.

### 8.4 `OnTick` fires per event-loop, not once

`eventloop.ticker` is a separate goroutine **per loop**. Under `Multicore` with N CPUs, `OnTick` is called N times per `delay` interval, concurrently. Any shared state it touches must be synchronized, or you must make it idempotent. This also means OnTick is the natural place for per-loop sweeps, but you can only reach *this loop's* conns through your own registry (the public API does not expose "iterate conns on this loop").

### 8.5 Buffer slice lifetime

Slices from `Next`/`Peek` point into gnet's reusable ring buffer. They are invalidated by the next `Read`/`Next`/`Discard` on that conn. **Never** send them to another goroutine without copying. The codec example `copy(body, buf[off:off+bodyLen])` is the model.

### 8.6 UDP differs

For UDP, `OnTraffic` gets a fresh `*conn` per datagram (or a transient conn), `AsyncWrite` goes synchronous (the docs warn it may become `ErrUnsupportedOp` for UDP), and `AsyncCallback`'s `Conn` may already be released — don't access it. Use `Conn.Write`/`SendTo` for UDP replies.

### 8.7 Cross-loop writes

Writing to a conn from a different loop (or any goroutine) must go through `AsyncWrite`/`AsyncWritev`. There is no broadcast API; a gnetx "broadcast to all sessions" must iterate its own registry and `AsyncWrite` per conn.

### 8.8 macOS/*BSD multi-loop caveat

`SO_REUSEPORT` is force-disabled on Darwin/*BSD when `Multicore || NumEventLoop>1`, so multi-loop still funnels accepts through one acceptor goroutine (Linux/DragonFly/FreeBSD get true kernel-level accept LB). Don't expect linear scaling on macOS.

---

## 9. Idiomatic example walkthroughs

### 9.1 Echo server (`gnet-io/gnet-examples/echo_tcp/echo.go`)

```go
type echoServer struct {
    gnet.BuiltinEventEngine
    eng       gnet.Engine
    addr      string
    multicore bool
}
func (es *echoServer) OnBoot(eng gnet.Engine) gnet.Action {
    es.eng = eng
    log.Printf("echo server with multi-core=%t is listening on %s\n", es.multicore, es.addr)
    return gnet.None
}
func (es *echoServer) OnTraffic(c gnet.Conn) gnet.Action {
    buf, _ := c.Next(-1)   // grab all buffered bytes (zero-copy slice into ring buffer)
    c.Write(buf)           // synchronous echo
    return gnet.None
}
func main() {
    // flags: --port 9000 --multicore
    echo := &echoServer{addr: fmt.Sprintf("tcp://:%d", port), multicore: multicore}
    log.Fatal(gnet.Run(echo, echo.addr, gnet.WithMulticore(multicore)))
}
```
Takeaways: embed `BuiltinEventEngine` so you only implement `OnBoot`/`OnTraffic`; `c.Next(-1)` is the cheapest "all bytes" read; `c.Write` is fine for synchronous on-loop replies; `gnet.Run` blocks.

### 9.2 Framed-protocol server (`simple_protocol/server/server.go` + `protocol/proto.go`)

Wire format `[2B magic=1314][4B BE body len][body]`. The codec (`SimpleCodec.Decode`) was shown in §4.2. The server:

```go
type simpleServer struct {
    gnet.BuiltinEventEngine
    eng       gnet.Engine
    batchRead int
    connected, disconnected int32
}
func (s *simpleServer) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
    c.SetContext(new(protocol.SimpleCodec))   // stash codec per-conn
    atomic.AddInt32(&s.connected, 1)
    return []byte("sweetness\r\n"), gnet.None  // send greeting from OnOpen
}
func (s *simpleServer) OnClose(c gnet.Conn, err error) gnet.Action {
    // ...counters; when connected==0, return gnet.Shutdown
}
func (s *simpleServer) OnTraffic(c gnet.Conn) gnet.Action {
    codec := c.Context().(*protocol.SimpleCodec)
    var packets [][]byte
    for i := 0; i < s.batchRead; i++ {           // batch up to batchRead frames per event
        data, err := codec.Decode(c)
        if err == protocol.ErrIncompletePacket { break }  // not enough bytes yet
        if err != nil { return gnet.Close }              // bad magic → drop conn
        packets = append(packets, must(codec.Encode(data)))
    }
    if n := len(packets); n > 1 { _, _ = c.Writev(packets) } else if n == 1 { _, _ = c.Write(packets[0]) }
    if len(packets) == s.batchRead && c.InboundBuffered() > 0 {
        if err := c.Wake(nil); err != nil { return gnet.Close }  // re-trigger OnTraffic for leftover
    }
    return gnet.None
}
```
Takeaways:
- **Per-conn codec instance via `SetContext`/`Context`** — the standard place for per-conn state.
- **Batch decoding** per `OnTraffic` event maximises throughput; cap with `batchRead` to stay fair.
- **`io.ErrShortBuffer` ⇒ `ErrIncompletePacket` ⇒ break** and leave bytes in the buffer; the next readable event resumes.
- **`c.Wake(nil)`** when you hit the batch cap but bytes remain — gnet won't auto-fire `OnTraffic` again unless there's a new readable event, so you force it.
- **`Writev`** for multi-frame replies (one syscall, vectorised).
- Returning `gnet.Shutdown` from `OnClose` when the last conn leaves is the example's "self-stop" trick.

---

## 10. Extension points for gnetx & gaps gnetx must fill

### 10.1 What gnet gives us to layer on (good extension points)

| gnet surface | gnetx use |
|---|---|
| `EventHandler` interface | gnetx implements `EventHandler` itself; adapts user's higher-level `Handler`/`middleware` chain. The user no longer writes `OnTraffic` directly. |
| `BuiltinEventEngine` | Embed in gnetx's engine struct to get no-op defaults. |
| `Conn.SetContext`/`Context` | Stash a gnetx `Session` (codec state, last-active, pending-request map, metadata) per conn. |
| `Reader.Peek`/`Discard`/`Next`/`InboundBuffered` | The primitives a gnetx `Codec.Decode(c)` calls. Build `LengthPrefixCodec`, `LineDelimiterCodec`, `FixedLengthCodec` on top. |
| `Writer.AsyncWrite`/`AsyncWritev` (+`AsyncCallback`) | Safe off-loop reply path for worker-pool handlers; the backbone of request/response futures. |
| `Conn.Wake` | Re-arm `OnTraffic` after partial decode / batch cap; also usable for "flush queued writes" semantics. |
| `Engine.Stop(ctx)` + `OnShutdown` | Graceful shutdown base; gnetx adds drain (stop accepting → finish in-flight → close). |
| `OnTick` (per-loop) | Idle/heartbeat sweeps per loop (each loop only sees its own conns via gnetx's per-loop registry). |
| `WithTicker`, `WithMulticore`, `WithNumEventLoop`, `WithLoadBalancing`, `WithTCPKeepAlive*`, `WithTCPNoDelay`, `WithReadBufferCap`/`WithWriteBufferCap`, `WithLogger` | Directly re-exposed as gnetx server/client options. |
| `Client.Dial`/`DialContext`/`Enroll` | gnetx client + connection pool + request/response futures on top. `DialContext`'s `ctx any` is a free per-conn tag. |
| `Engine.Register` / `EventLoop.Enroll`/`Execute` | Adopt externally-accepted conns (e.g. from a custom listener, TLS handshaked conn) into gnet's loops. |
| `SourceAddrHash` LB | Session affinity so a given client always lands on the same loop — simplifies per-client state. |

### 10.2 Gaps gnetx must fill (gnet deliberately does not)

1. **Codec / packet framing** — gnet ships none. gnetx must provide at minimum: length-prefix (uint16/uint32 BE/LE), line/newline delimiter, fixed-length, and a pluggable `Codec{Encode, Decode}` interface. Template: the `simple_protocol` `SimpleCodec`.
2. **Session abstraction** — gnet's `Conn.Context()` is an opaque `any`. gnetx must define a `Session` with lifecycle (open/active/idle/closing), metadata, and a per-conn registry, mapped to `SetContext`.
3. **Request/response correlation** — gnet has zero. gnetx must add request IDs, a pending-future map (mutex-guarded, since touched off-loop), timeout, and a `Send(req) (resp, error)` API on the client. Server side: a `Handler(req) (resp, error)` contract, with offloading to a worker pool replying via `AsyncWrite`.
4. **Idle timeout & heartbeat** — gnet only has kernel TCP keepalive and the blunt per-loop `OnTick`. gnetx must track `lastActiveAt` per session (updated in `OnTraffic`), sweep in `OnTick` per loop, and close idle conns via `Conn.Close()` (safe cross-loop). App-level ping/pong must also be layered.
5. **Graceful drain on shutdown** — `Engine.Stop` waits for conns to close but doesn't *drain* (stop accepting, let in-flight finish, optionally send a "going away" frame, then close). gnetx must orchestrate this on top of `OnShutdown`/`Engine.Stop`.
6. **Connection registry & broadcast** — gnet exposes only `CountConnections`, no enumeration. gnetx must keep its own registry (per-loop for OnTick sweeps, plus a global map for broadcast/lookup) and broadcast via per-conn `AsyncWrite`.
7. **Router / middleware chain** — gnet's `EventHandler` is flat. gnetx should provide a request router (by opcode/path) and middleware (auth, logging, metrics, rate-limit) wrapping the user handler.
8. **TLS** — explicitly on gnet's roadmap, **not implemented**. gnetx must either layer TLS by enrolling a `tls.Conn` via `Client.Enroll`/`Engine.Register` (caveat: loses zero-copy, reintroduces net.Conn overhead) or wait for native gnet TLS.
9. **Observability** — gnet has zap logging only. gnetx must add metrics (conns, bytes, frame counts, latency), tracing hooks, and structured request logging.
10. **Client connection pool & reconnect** — gnet's `Client` dials one conn at a time. gnetx must add pooling, health checks, exponential-backoff reconnect, and fan-out.

### 10.3 Threading contract gnetx must enforce

- gnetx's `OnTraffic` runs on-loop: it must call `Codec.Decode` (Peek/Discard only), dispatch to a handler, and either reply synchronously (`c.Write`) or hand off to a worker that replies via `c.AsyncWrite`. **Never block.**
- The gnetx `Session`'s pending-request map is touched both on-loop (matching responses in `OnTraffic`) and off-loop (registering/timing out futures) → must be mutex-guarded (not relying on gnet's loop affinity).
- Per-loop idle sweeps in `OnTick` are safe for *this loop's* conns; closing from `OnTick` is safe (`Conn.Close` is concurrency-safe). Cross-loop operations must go through `AsyncWrite`.
- On macOS/*BSD, multi-loop does not kernel-LB accepts; gnetx should not promise linear scaling there and may default `SourceAddrHash` for affinity-friendly behaviour.

---

## 11. Versioning & Go 1.26 safety

- **Module path**: `github.com/panjf2000/gnet/v2` (v2 via `/v2` suffix; v1 is `github.com/panjf2000/gnet`, deprecated path).
- **Latest tagged version**: **v2.9.8** (per pkg.go.dev, published 2026-05-25). Latest GitHub *release* with release notes: **v2.9.0** "Millennium Actress" (2025-06-11), whose headline features are:
  - Client can run with multiple event-loops (#709).
  - Customizable `TCP_KEEPINTVL` and `TCP_KEEPCNT` (#708) → `WithTCPKeepInterval`/`WithTCPKeepCount`.
- **Go version requirement**: `go.mod` declares `go 1.20`; README badge `>=1.20`. Notable dependency pins in `go.mod`:
  ```
  golang.org/x/sync v0.11.0  // don't upgrade beyond — v0.12.0+ requires Go 1.23+
  golang.org/x/sys   v0.30.0 // don't upgrade beyond — v0.31.0+ requires Go 1.23+
  ```
  Other deps: `panjf2000/ants/v2 v2.12.1`, `valyala/bytebufferpool v1.0.0`, `go.uber.org/zap v1.28.0`, `gopkg.in/natefinch/lumberjack.v2 v2.2.1`, `stretchr/testify v1.11.1`.
- **Go 1.26 safety**: Safe. gnet targets Go 1.20+ and uses only stable syscall/`net`/`os` APIs through `golang.org/x/sys/unix`. The maintainers' own pins cap `x/sys`/`x/sync` at versions that still support Go 1.20, so the default `go mod tidy` result builds on anything ≥1.20, including 1.26. If gnetx's own `go.mod` bumps `x/sys` to ≥v0.31.0 (needs Go 1.23+), that's still satisfied by Go 1.26. No known use of removed/deprecated stdlib APIs. The only platform caveat: the **Windows port is officially dev/test-only** (per README); production should run on Linux/macOS/*BSD.
- **Roadmap items not yet in v2.9.x** (relevant to gnetx planning): **TLS** (not done — gnetx must layer it), **io_uring**, **KCP**.

---

## Caveats / Not verified

- Exact line numbers in `gnet.go` are taken from pkg.go.dev's v2.9.8 rendering and may drift on the `dev` branch; symbol names and signatures are stable and were confirmed against the raw `dev` source fetched on 2026-07-02.
- The `EventLoop.Schedule` method is declared in the interface but currently returns `errorx.ErrUnsupportedOp` ("TODO: not supported yet, implement this"). Do not rely on it for delayed tasks in gnetx — use a `time.Timer` + `EventLoop.Execute` instead.
- The legacy `gnet.Stop(ctx, protoAddr)` global func is **deprecated** (can leak engines under `WithReuseAddr`+`WithReusePort`); always use `Engine.Stop(ctx)`.
- `AsyncWrite` for UDP is documented as going synchronously and may become `ErrUnsupportedOp` in a future gnet version; gnetx should not build critical UDP logic on it.
- Native TLS is absent; the `Client.Enroll(tls.Conn)` / `Engine.Register(tls.Conn)` workaround reintroduces `net.Conn` copying and was not benchmarked here.
- No per-connection public iteration API exists (only `CountConnections`); any "list/broadcast conns" feature in gnetx must be built on gnetx's own registry.
