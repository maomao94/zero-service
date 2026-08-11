# Research: Remaining Services & Common Packages

- **Query**: Analyze architecture, key types, and inter-connections of 4 services + 9 common packages
- **Scope**: internal
- **Date**: 2026-08-11

---

## 1. app/lalhook/ — LAL Hook Service

### Entry Pattern
- **`lalhook.go`** — Standard go-zero REST entry (`main` package)
- Uses `conf.MustLoad` to load `config.Config` YAML
- Creates `rest.MustNewServer` with custom CORS (`rest.WithCustomCors`) allowing dynamic Origin, credential-carrying, and specific headers
- `svc.NewServiceContext(c)` → `handler.RegisterHandlers(server, ctx)` → `server.Start()`
- Calls `tool.PrintGoVersion()` at startup

### Key Types/Interfaces
- **`config.Config`** — embeds `rest.RestConf`, has `DB.DataSource` field
- **`svc.ServiceContext`** — holds `Config` + `HlsTsFilesModel` (model-layer DB model using `github.com/Masterminds/squirrel` query builder)
- **`types.*`** (`types.go`) — goctl-generated request/response types:
  - Webhook event types: `OnPubStartRequest`, `OnPubStopRequest`, `OnSubStartRequest`, `OnSubStopRequest`, `OnRelayPullStartRequest`, `OnRelayPullStopRequest`, `OnHlsMakeTsRequest`, `OnRtmpConnectRequest`, `OnServerStartRequest`, `OnUpdateRequest`
  - API types: `ApiListTsRequest`, `ApiListTsReply`, `ApiTsFile`
  - Aggregate types: `GroupInfo`, `PubInfo`, `SubInfo`, `PullInfo`, `FpsInfo`
  - Response type: `EmptyReply` (used by all webhook handlers as they mostly just acknowledge)

### Architecture
- **Two route groups** (`routes.go`):
  - `/v1/api` — single endpoint `POST /ts/list` for querying TS file records from DB
  - `/v1/hook` — 10 webhook endpoints (`onHlsMakeTs`, `onPubStart`, `onPubStop`, `onRelayPullStart`, `onRelayPullStop`, `onRtmpConnect`, `onServerStart`, `onSubStart`, `onSubStop`, `onUpdate`)
  - Both groups use `rest.WithTimeout(7200000*time.Millisecond)` (2-hour timeout)
- **Logic layer** splits into two packages:
  - `internal/logic/webhook/` — handlers for LAL server HTTP Notify callbacks
  - `internal/logic/api/` — handlers for query APIs
- **All logic structs** embed `logx.Logger` and hold `ctx context.Context` + `svcCtx *svc.ServiceContext`
- **Most webhook handlers are stubs** — they log and return `EmptyReply`, actual business logic `// todo` placeholders
- **`ListTsFilesLogic`** is fully implemented: constructs a `squirrel.SelectBuilder` with optional filters (stream name, time range, event type), queries `HlsTsFilesModel`, maps to `ApiTsFile` list

### Unique Conventions
- Uses `github.com/Masterminds/squirrel` for SQL query building on the model layer
- Webhook endpoints accept POST from lalserver's HTTP Notify system
- All responses go through go-zero's REST framework with automatic JSON marshaling
- 2-hour timeout on all routes suggests long-polling or large TS file queries

---

## 2. app/lalproxy/ — LAL Proxy Service

### Entry Pattern
- **`lalproxy.go`** — Standard go-zero zRPC entry
- Uses `conf.MustLoad` → `svc.NewServiceContext(c)` → `zrpc.MustNewServer`
- Registers gRPC service: `lalproxy.RegisterLalProxyServer(grpcServer, server.NewLalProxyServer(ctx))`
- gRPC reflection enabled in DevMode/TestMode
- Optional Nacos service registration (same pattern as podengine/logdump)
- `s.AddUnaryInterceptors(interceptor.LoggerInterceptor)` + `logx.AddGlobalFields`
- Calls `tool.PrintGoVersion()` at startup

### Key Types/Interfaces
- **`config.Config`** — embeds `zrpc.RpcServerConf`, has `NacosConfig` (optional) and `LalServer` config (IP, Port, Timeout)
- **`svc.ServiceContext`** — holds `Config`, `LalBaseUrl` (formatted as `http://{ip}:{port}`), and `LalClient httpc.Service` (go-zero's HTTP client with timeout)
- **Generated protobuf**: `lalproxy/` package contains `.pb.go`, `_grpc.pb.go`, `.pb.validate.go`

### Architecture
- **gRPC server wrapping LAL HTTP API** — proxy pattern
- 9 RPC methods (`server/lalproxyserver.go`):
  - `GetGroupInfo` — GET `/api/stat/group?stream_name=`
  - `GetAllGroups` — GET `/api/stat/all_group`
  - `GetLalInfo` — GET `/api/stat/lal_info`
  - `StartRelayPull` — POST `/api/ctrl/start_relay_pull` (JSON body)
  - `StopRelayPull` — GET `/api/ctrl/stop_relay_pull`
  - `KickSession` — POST `/api/ctrl/kick_session` (JSON body)
  - `StartRtpPub` — POST `/api/ctrl/start_rtp_pub` (JSON body)
  - `StopRtpPub` — (not yet implemented, uses KickSession)
  - `AddIpBlacklist` — POST `/api/ctrl/add_ip_blacklist` (JSON body)
- **Logic layer pattern**:
  - Each logic struct: `ctx context.Context`, `svcCtx *svc.ServiceContext`, `logx.Logger`
  - Constructor: `NewXxxLogic(ctx, svcCtx)`
  - Method signature: `(in *lalproxy.XxxReq) (*lalproxy.XxxRes, error)`
  - Validation at start, then HTTP call via `svcCtx.LalClient.Do()`, JSON unmarshal into typed response
  - Uses `lalx.GroupData` / `lalx.LalServerData` for LAL data types
  - Uses `copier.Copy()` to map `lalx.*` types to protobuf-generated types
  - Error handling: `tool.NewErrorByPbCode(extproto.Code__1_06_THIRD_PARTY, ...)` for third-party failures

### Connections
- Depends on **`common/lalx`** for LAL data types (`GroupData`, `LalServerData`, `PubSessionInfo`, etc.)
- Depends on **`common/tool`** for error helpers
- Depends on **`third_party/extproto`** for error codes

### Unique Conventions
- gRPC proxy over HTTP: wraps lalserver's REST API behind a gRPC interface
- Uses `copier` for struct mapping between domain types and protobuf types
- LAL API errors are returned in response struct (not as gRPC errors) — `ErrorCode` + `Desp` fields
- Configurable LAL server address with defaults (`127.0.0.1:8083`, 5s timeout)

---

## 3. app/podengine/ — Pod Engine Service

### Entry Pattern
- **`podengine.go`** — Standard go-zero zRPC entry
- Same Nacos registration / interceptor pattern as lalproxy and logdump
- **Unique**: imports `_ "zero-service/common/carbonx"` (carbon time library side-effect init)
- Calls `tool.PrintGoVersion()` (note: placed after context creation, unlike lalproxy/logdump)

### Key Types/Interfaces
- **`config.Config`** — embeds `zrpc.RpcServerConf`, has `NacosConfig` (optional) and `DockerConfig map[string]string` (optional, maps node name to Docker host address)
- **`svc.ServiceContext`** — holds `Config`, `DockerClients map[string]*client.Client` (thread-safe with `sync.RWMutex`)
  - Always creates a "local" Docker client via `dockerx.MustNewClient`
  - Optional remote Docker hosts from config
  - `GetDockerClient(name)` with read lock — defaults to "local" if name empty

### Architecture
- **Docker container orchestration service** — "pod" = Docker container
- 9 RPC methods (`server/podengineserver.go`):
  - `CreatePod` — creates container with image, env, ports, volumes, resources, restart policy
  - `StartPod`, `StopPod`, `RestartPod` — container lifecycle
  - `GetPod` — container inspect
  - `ListPods` — container list with filters (id, name, labels), pagination (offset/limit)
  - `DeletePod` — container removal
  - `GetPodStats` — container stats
  - `ListImages` — Docker image list
- **CreatePod** is the most complex:
  - Validates request via protobuf validation
  - Builds `container.Config` (image, env, cmd, labels, stop timeout)
  - Builds `container.HostConfig` (port bindings, restart policy, network mode, resources, volume mounts)
  - Parses ports (`host:container` format), resources (cpu/memory/cpuRequest/memoryRequest), volume mounts (`host:container[:ro]` format)
  - Default terminationGracePeriodSeconds = 60
  - Container naming: `{podName}-{containerName}` (lowercase)
  - Returns `PodPb` with phase PENDING after creation
- Uses **`dockerx`** package for:
  - `MustNewClient` (with OTel tracing)
  - `BuildEnvList` (map to []string)
  - `ExtractContainerVolumeMounts` (for ListPods display)
- Uses **`carbon`** for timestamp formatting

### Connections
- Depends on **`common/dockerx`** for Docker client creation and helper functions
- Depends on **`common/carbonx`** (side-effect import for carbon time library)
- Depends on **`common/tool`** for error helpers

### Unique Conventions
- Multi-Docker-host support via named clients in map
- Protobuf validation via generated `.Validate()` methods
- Kubernetes-like API surface (Pod, Container, phases: PENDING/RUNNING/SUCCEEDED/STOPPED/UNKNOWN)
- Custom resource parsing from string maps (not protobuf native — `map[string]string` for resources)

---

## 4. app/logdump/ — Log Dump Service

### Entry Pattern
- **`logdump.go`** — Standard go-zero zRPC entry, same pattern as lalproxy
- Calls `tool.PrintGoVersion()` before context creation
- `s.AddUnaryInterceptors(interceptor.LoggerInterceptor)` + `logx.AddGlobalFields`

### Key Types/Interfaces
- **`config.Config`** — embeds `zrpc.RpcServerConf`, has `NacosConfig` (optional) and `ExtraFields []string` (optional, list of allowed extra log field names)
- **`svc.ServiceContext`** — minimal: only holds `Config` (no DB, no external clients)

### Architecture
- **Log aggregation/receiving service** — receives logs via gRPC and writes them through go-zero's `logx` system
- 2 RPC methods (`server/logdumpserver.go`):
  - `Ping` — health check
  - `PushLog` — receives `PushLogReq` containing `[]*LogEntry`
- **PushLogLogic**:
  - Iterates over received log entries
  - Builds `logx.LogField` slice: always includes `seq` and `service`
  - Extra fields: only those in `ExtraFields` config whitelist are added as structured fields; ALL extra fields are concatenated into a human-readable string
  - Message format: `[{service}] {message} | key1=val1, key2=val2`
  - Log level routing: `logdump.LogLevel_ERROR` → `l.Logger.Error()`, default → `l.Logger.Info()`
  - Returns empty `PushLogRes`

### Connections
- Depends on **`common/tool`** (via entry pattern)
- No dependency on other common packages in logic

### Unique Conventions
- Minimal service — acts as a gRPC-to-logx bridge
- Config-based field whitelisting for structured logging
- Uses go-zero's structured logging (`WithFields`)
- All log entries from a single gRPC call are logged individually

---

## 5. common/lalx/ — LAL Utilities

### Overview
Single file: `laltype.go` — Go struct definitions for LAL server's HTTP API JSON responses.

### Key Types
- **`GroupData`** — Core aggregate: `StreamName`, `AppName`, `AudioCodec`, `VideoCodec`, `VideoWidth`, `VideoHeight`, `Pub *PubSessionInfo`, `Subs []*SubSessionInfo`, `Pull *PullSessionInfo`, `Pushs []*PushSessionInfo`, `InFramePerSec []*FrameData`
- **`PubSessionInfo`** — Publisher session: `SessionId`, `Protocol` (RTMP/RTSP), `BaseType` ("PUB"), `StartTime`, `RemoteAddr`, `ReadBytesSum`, `WroteBytesSum`, bitrate fields
- **`SubSessionInfo`** — Subscriber session: same fields, `BaseType` ("SUB")
- **`PullSessionInfo`** — Relay pull session: same fields, `BaseType` ("PULL")
- **`PushSessionInfo`** — Relay push session (stub, no fields)
- **`FrameData`** — FPS data point: `UnixSec`, `V int32`
- **`LalServerData`** — Server info: `ServerId`, `BinInfo`, `LalVersion`, `ApiVersion`, `NotifyVersion`, `WebUiVersion`, `StartTime`

### Connections
- Used by **`app/lalproxy`** — `GetGroupInfo` parses LAL API response into `lalx.GroupData`, then copies to protobuf types via `copier`
- Used by **`app/lalhook`** — (types.go has equivalent but separate type definitions; `lalhook` does NOT import `lalx`)

### Unique Conventions
- Every struct field has explicit JSON tags with snake_case naming
- Comments document which LAL API endpoint each struct corresponds to
- `Pushs` uses `[]interface{}` in lalhook's types, but `[]*PushSessionInfo` in lalx — shows lalx has stronger typing
- lalhook defines its own duplicate types rather than importing lalx

---

## 6. common/netx/ — Networking Utilities

### Overview
A full-featured HTTP client library with engine abstraction, request/response builders, encoding utilities, upload/download support.

### Key Types/Interfaces
- **`Engine` interface** — `Do(req *http.Request) (*http.Response, error)`
- **`DefaultEngine`** — wraps `*http.Client`
- **`HTTPCEngine`** — wraps go-zero's `httpc.Service` (for circuit breaker/middleware integration)
- **`Client`** — main HTTP client: configurable engine, TLS, headers, size limits
  - `ClientOptions` + `ClientOption` functional options pattern
  - Methods: `Get`, `Post`, `Put`, `Delete`, `Patch`, `Head`, `Options`
  - `Do(ctx, *Request)` — primary execution method
- **`Request`** — HTTP request builder: URL, Method, Headers, QueryParams, FormData, Body ([]byte or io.Reader), ContentType
  - Supports body kinds: `bodyKindNone`, `bodyKindRaw`, `bodyKindJSON`, `bodyKindForm`, `bodyKindReader`
  - Chainable builder methods + functional `RequestOption` functions
- **`Response`** — HTTP response wrapper: StatusCode, Headers, Data, CostMs, CostFormatted, Success, Err
  - `JSON(target)`, `XML(target)`, `Text()`, `Decode(target)` — auto content-type detection
  - Even on network errors, returns non-nil `Response` with Err field (never returns error)

### Key Functions
- **`NewClient(opts ...ClientOption)`** — functional options: Engine, TLS, Headers, size limits, HTTP client options
- **`NewRequest(url, method, opts ...RequestOption)`**
- **`NewTransport(opts ...TransportOption)`** — creates `http.Transport` with sensible defaults (30s dial, 10s TLS, 90s idle, 100 max idle, HTTP/2)
- **`NewHTTPClient(opts ...TransportOption)`** — creates `http.Client` using `NewTransport`
- **`NewHTTPCService(name, opts ...TransportOption)`** — creates go-zero `httpc.Service`
- **`NewHTTPEngine(svc httpc.Service)`** — wraps `httpc.Service` as `Engine`
- **Package-level helpers**: `Get()`, `Post()`, `Put()`, `Delete()`, `Patch()`, `Head()`, `Options()`, `SendRequest()` — use package-level `defaultClient`

### Encoding
- `ValidateAndFlatten` — JSON → flat key-value map
- `EncodeURLEncoded` — JSON → URL-encoded string
- `EncodeURLEncodedIfNeeded` — smart: JSON → flat, or pass through if already encoded
- `EncodeMultipart` — key-value map → multipart/form-data

### Unique Conventions
- **Engine abstraction** allows swapping `http.Client` for go-zero's `httpc.Service` (with built-in circuit breaking)
- **Errors are never returned from `Do()`** — they're always captured in `Response.Err` with semantic status codes (504 for timeout, 503 for network)
- **`WithHTTPClientOption`** allows passing raw `func(*http.Client)` for flexibility beyond Engine abstraction
- Default size limits: 10MB response, 32MB upload

---

## 7. common/wsx/ — WebSocket Utilities

### Overview
A robust WebSocket client with auto-reconnect, state machine, authentication, heartbeat, token refresh, and OpenTelemetry tracing.

### Key Types/Interfaces
- **`Client` interface** — `Send(ctx, []byte)`, `SendJSON(ctx, any)`, `Close()`, `State()`
- **`client` struct** — internal implementation:
  - `cfg Config`, `opts clientOptions`
  - `conn atomic.Pointer[websocket.Conn]` — atomic pointer for lock-free reads
  - `state atomic.Int32` — connection state machine
  - `writeMu sync.Mutex` — serializes writes
  - `closed atomic.Bool`, `wg sync.WaitGroup`
  - `metrics *stat.Metrics`, `tracer oteltrace.Tracer` — observability
- **`Config`** — URL + timeouts: `DialTimeout` (10s), `WriteTimeout` (10s), `ReadTimeout` (60s), `AuthTimeout` (5s), `HeartbeatInterval` (30s), `ReconnectInterval` (1s), `TokenRefreshInterval` (30min)
- **`ConnState`** — state machine: `Disconnected` → `Connecting` → `Connected` → `Authenticated` / `AuthFailed` → `Reconnecting`
- **`clientOptions`** — functional options: `headers`, `dialer`, `onAuthenticate`, `onMessage`, `onStateChange`, `onTokenRefresh`, `onHeartbeat`, `metrics`
- **`MessageHandler` interface** + `MessageHandlerFunc` adapter

### Architecture
- **`MustNewClient(cfg, opts...)`** — panics on error, registers `proc.AddWrapUpListener` for graceful shutdown
- **`NewClient(cfg, opts...)`** — validates URL, normalizes config, starts `running()` goroutine
- **`running()`** — main loop: dial → authenticate → heartbeat + token refresh → read until disconnect → reconnect
- **`readLoop()`** — reads messages, dispatches via `threading.GoSafe` with trace spans
- **`heartbeater()`** — sends text/json heartbeat (via `onHeartbeat`) or WebSocket ping
- **`startTokenRefresher()`** — periodic token refresh, triggers reconnect on failure
- **`Send()`** — only allows sending when state is `StateAuthenticated`
- **`Close()`** — idempotent, sends close frame, waits for goroutines

### Unique Conventions
- **Atomic state machine** for connection lifecycle
- **Authentication phase** before allowing sends
- **Auto-reconnect** with configurable interval
- **OpenTelemetry** spans on every received message
- **go-zero `proc.AddWrapUpListener`** integration for graceful shutdown
- **`stat.Metrics`** integration for throughput/drop monitoring
- MD5-hashed URL for metrics naming and session identification
- Write serialization via `sync.Mutex` (not channel-based)

---

## 8. common/socketiox/ — Socket.IO Utilities

### Overview
A comprehensive Socket.IO server implementation built on `github.com/doquangtan/socketio/v4`, with room management, broadcasting, stat reporting, Nacos service discovery for multi-node clusters, and context-aware metadata.

### Key Types/Interfaces
- **`Server`** — main server, wraps `*socketio.Io`:
  - `eventHandlers map[string]EventHandler`
  - `sessions map[string]*Session` (thread-safe with `sync.RWMutex`)
  - Hooks: `tokenValidator`, `tokenValidatorWithClaims`, `connectHook`, `disconnectHook`, `preJoinRoomHook`
  - `contextKeys []string` — keys to extract from JWT claims into session metadata
- **`Session`** — per-connection session:
  - `socketId`, `socket *socketio.Socket`, `metadata map[string]string`
  - Methods: `JoinRoom()`, `LeaveRoom()`, `EmitAny()`, `EmitString()`, `EmitDown()`, `EmitEventDown()`, `ReplyEventDown()`
  - `GetMetadata(key)`, `AllMetadata()`, `SetMetadata(key, val)`
  - `Close()` via `Disconnect()`
- **`EventHandler` interface** — `Handle(ctx, event, payload) (string, error)`
- **`EventHandlers`** — `map[string]EventHandler`
- **`TokenValidator`**, **`TokenValidatorWithClaims`**, **`ConnectHook`**, **`DisconnectHook`**, **`PreJoinRoomHook`** — function types

### Built-in Events
- `__connection__` — connect hook, joins initial rooms
- `__disconnect__` — disconnect hook, cleans session
- `__up__` — main upstream event, routed to registered `EventUp` handler
- `__join_room_up__` — join room (with preJoinRoomHook)
- `__leave_room_up__` — leave room
- `__rooms_page_up__` — paginated room list
- `__room_broadcast_up__` — broadcast to room
- `__global_broadcast_up__` — broadcast to all
- `__stat_down__` — periodic stat push (every minute) with Nps, room count, metadata
- `__down__` — downstream event

### Request/Response Types
- **`SocketUpReq`** — `{payload, reqId, room, event}`
- **`SocketResp`** — `{code, msg, payload, reqId}` (200/400/500)
- **`SocketDown`** — `{event, payload, reqId}`
- **`SocketRoomsPageRes`** — `{total, page, pageSize, totalPages, rooms}`
- **`StatDown`** — `{socketId, roomCount, rooms, nps, metadata, roomLoadError}`

### Architecture
- **`NewServer(opts...)`** — creates `socketio.Io`, binds all event handlers, starts stat loop
- **Authentication** — supports `TokenValidator` (bool) and `TokenValidatorWithClaims` (map of claims)
- **Token claims** → session metadata via `contextKeys` config
- **Custom events** — registered via `WithEventHandlers` or `WithHandler`, bypass `__up__` routing
- **Ack support** — both Ack-based and reply-via-downstream patterns
- **Stat loop** — every minute, pushes `__stat_down__` to each session with room info and Nps
- **Session management** — `GetSession(id)`, `GetSessionByDeviceId()`, `GetSessionByUserId()`, `GetSessionByKey()`

### Container (Multi-Node)
- **`SocketContainer`** — gRPC client pool for multi-node Socket.IO:
  - `ClientMap map[string]socketgtw.SocketGtwClient`
  - Supports Direct endpoints, Etcd discovery, Nacos discovery
  - `MustNewPubContainer(c zrpc.RpcClientConf)` — creates from config
  - Nacos: subscribes to service changes, polls every 60s as fallback
  - Etcd: uses go-zero's `discov.Subscriber` with subset sampling (max 32)

### Connections
- Depends on **`socketapp/socketgtw`** (generated gRPC client for inter-node communication)
- Depends on **`common/ctxdata`** for `CtxAuthorizationKey`
- Depends on **`common/Interceptor/rpcclient`** for gRPC metadata interceptor

### Unique Conventions
- **Ack + ReplyDown dual response pattern** — if client sends with Ack callback, respond via Ack; otherwise via `__down__` event
- **Event name protection** — `__down__` is reserved, cannot be used as broadcast event
- **Nacos gRPC port** extracted from `metadata["gRPC_port"]` for health checking
- **Stat messages** include `Nps` (network packets per second) from socketio library
- Even sentinel messages have response codes (200/400/500)
- `visibleSessionRooms` filters out socket's own ID from room list

---

## 9. common/ssex/ — SSE Utilities

### Overview
A concise Server-Sent Events writer implementation for HTTP streaming.

### Key Types
- **`Writer`** — wraps `http.ResponseWriter` + `http.Flusher`:
  - `mu sync.Mutex` for thread safety
  - `buf []byte` for line-buffered writing
  - Implements `io.Writer` interface (for streaming A2UI output)

### Methods
- **`NewWriter(w http.ResponseWriter)`** — requires Flusher support
- **`Write(p []byte)`** — io.Writer: buffers until newline, emits `data: {line}\n\n` per line, auto-flushes
- **`WriteEvent(event, data)`** — `event: {event}\ndata: {data}\n\n`
- **`WriteData(data)`** — `data: {data}\n\n`
- **`WriteComment(comment)`** — `: {comment}\n\n` (client-ignored)
- **`WriteKeepAlive()`** — sends `: keepalive\n\n` comment
- **`WriteJSON(v any)`** — JSON marshal → `data: {json}\n\n`
- **`WriteDone()`** — OpenAI-style `data: [DONE]\n\n`
- **`Flush()`**, **`BufferFlush()`** — manual flush controls
- **`ResponseWriter()`** — exposes underlying writer

### Unique Conventions
- Line-buffered `io.Writer` for stream-to-SSE conversion
- OpenAI-compatible `[DONE]` marker
- Backward compatibility: `LineWriter` = `Writer`, `NewLineWriter` = `NewWriter`

---

## 10. common/flowx/ — Flow Utilities

### Overview
Wrapper around `github.com/Azure/go-workflow` with go-zero logging integration and functional options.

### Key Types
- **`LoggingStepInterceptor`** — `flow.StepInterceptor` that logs step start/duration/error via go-zero `logx`
- **`FlowOptions`** — mirrors `flow.WorkflowOption` with pointer fields for nil=unset semantics:
  - `MaxConcurrency`, `DontPanic`, `SkipAsError`, `DontInherit`
  - `Clock`, `StepDefaults`, `Mutators`, `StepInterceptors`, `AttemptInterceptors`

### Key Functions
- **`New(opts ...FlowOption)`** — creates `*flow.Workflow` from options
- **`StepFields(extra...)`** — injects `step=<name>` + extra fields into logx context
- **`AttemptFields(extra...)`** — injects `attempt=<N>` + extra fields into logx context
- **`WithMaxConcurrency(n)`**, **`WithDontPanic()`**, **`WithSkipAsError()`**, **`WithDontInherit()`**, **`WithClock(c)`**
- **`WithStepDefaults(sd)`**, **`WithStepInterceptor(ic)`**, **`WithAttemptInterceptor(ic)`**, **`WithMutator(m)`**

### Unique Conventions
- Uses **`github.com/Azure/go-workflow`** for durable workflow orchestration
- Logging interceptors are go-zero aware (structured logging with context fields)
- Interceptor ordering matters: `StepFields` should be outermost (before `LoggingStepInterceptor`)
- `Mutator` support for cross-cutting step configuration injection

---

## 11. common/mediax/ — Media Utilities

### Overview
Video screenshot utility using `ffmpeg-go` bindings.

### Key Types
- **`Screenshotter`** — holds `inputPath string` (local file or stream URL)

### Methods
- **`NewScreenshotter(inputPath)`** — validates non-empty path
- **`CaptureFrameToFile(ctx, timePoint, localFilePath)`**:
  - For local files: seeks to `timePoint` seconds, captures 1 frame as JPEG
  - For live streams: `timePoint = -1` for current frame
  - Quality: `q:v = 2` (1-31 scale, 1=best)
  - Validates output file exists and non-empty, cleans up on failure
  - Logs timing and file size
- **`CaptureFrameByIndexToFile(ctx, frameIndex, localFilePath)`**:
  - Uses `select=eq(n,{frameIndex})` filter
  - Same validation/cleanup pattern
- **`GenerateTempFilePath(baseDir, ext)`** — `{baseDir}/YYYYMMDD/{uuid}{ext}`

### Unique Conventions
- Uses **`github.com/u2takey/ffmpeg-go`** (Go bindings for FFmpeg)
- File validation + cleanup on failure (atomic-like pattern)
- Detailed debug logging of FFmpeg stderr output

---

## 12. common/dockerx/ — Docker Utilities

### Overview
Docker client helper functions used by podengine.

### Key Functions
- **`MustNewClient(ops ...client.Opt)`** — creates Docker client with OTel tracing (`otel.GetTracerProvider()`), panics on error
- **`ParseContainerEnv(env []string)`** — splits `KEY=VALUE` pairs to `map[string]string`
- **`ExtractContainerPorts(networkSettings)`** — formats ports as `{hostIP}:{hostPort}->{containerPort}/{proto}`
- **`ExtractContainerVolumeMounts(mounts)`** — formats as `{source}:{destination}:{mode}`
- **`ParseContainerResources(resources)`** — Docker resource struct → `map[string]string` (cpu, memory, cpuRequest, memoryRequest)
- **`BuildEnvList(envMap)`** — `map[string]string` → `[]string` (reverse of ParseContainerEnv)

### Unique Conventions
- All functions are pure helpers — no structs or interfaces
- OTel tracing automatically injected into Docker client
- Resource parsing handles CPU quota (÷100000) and shares (÷1024) conversions
- Memory parsing in podengine (not dockerx) handles units (K/M/G/T)

---

## 13. common/executorx/ — Executor Utilities

### Overview
A chunked message pusher built on go-zero's `executors.ChunkExecutor`.

### Key Types
- **`ChunkSender`** — `func(messages []string)` — callback to flush a chunk of messages
- **`ChunkMessagesPusher`** — wraps `*executors.ChunkExecutor`:
  - `inserter *executors.ChunkExecutor`
  - `chunkSender ChunkSender`
  - `writerLock sync.Mutex`

### Methods
- **`NewChunkMessagesPusher(chunkSender, chunkBytes)`** — creates with byte threshold
- **`Write(val string)`** — adds message to chunk buffer (thread-safe via mutex)
- **`execute(vals []interface{})`** — internal: type-asserts to strings, calls `chunkSender`

### Unique Conventions
- Uses go-zero's built-in `ChunkExecutor` for batching
- Byte-based chunking (not count-based)
- Simple string-only message type

---

## Cross-Cutting Patterns Summary

1. **Entry point uniformity**: All services use `flag.String("f", ...)` + `conf.MustLoad` + `tool.PrintGoVersion()`. zRPC services share identical Nacos registration and interceptor patterns.

2. **Logic layer pattern**: `xxxLogic` struct embedding `logx.Logger`, constructor `NewXxxLogic(ctx, svcCtx)`, method receiving protobuf request and returning protobuf response + error.

3. **Error handling**: `tool.NewErrorByPbCode(extproto.Code__1_XX_XXX, msg)` for structured gRPC errors.

4. **Service communications**:
   - `lalhook` (REST) ← lalserver HTTP Notify callbacks
   - `lalproxy` (gRPC) → lalserver HTTP API (proxy pattern)
   - `podengine` (gRPC) → Docker Engine API
   - `logdump` (gRPC) → go-zero logx

5. **Type duplication**: `lalhook/types.go` duplicates LAL types that also exist in `common/lalx` — lalhook does NOT import lalx.

6. **Functional options pattern**: Used consistently across `netx`, `wsx`, `flowx`, `socketiox` for configuration.

7. **go-zero integration depth**:
   - Deep: `netx` (wraps `httpc.Service`), `wsx` (uses `logx`, `stat.Metrics`, `proc`), `socketiox` (uses `logx`, `threading`, Nacos/Etcd discovery)
   - Shallow: `mediax`, `dockerx`, `executorx` (only use `logx`)

8. **Observability**: `wsx` and `socketiox` have built-in metrics/tracing; `dockerx` injects OTel; `flowx` has logging interceptors.
