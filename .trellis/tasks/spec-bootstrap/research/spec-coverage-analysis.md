# Research: Spec Coverage Analysis for Trellis Spec Bootstrap

- **Query**: Analyze Go project at zero-service for spec coverage gaps
- **Scope**: internal (codebase + .trellis/spec/backend/)
- **Date**: 2026-07-03

## EXISTING_PACKAGES (common/ — 38 packages)

| # | Package | Type | Source |
|---|---------|------|--------|
| 1 | `alarmx` | domain | `common/alarmx/alarmx.go` |
| 2 | `antsx` | domain | `common/antsx/` (15 files: Promise, Invoke, ReplyPool, Emitter, Reactor, Stream, Tee, Unbounded) |
| 3 | `asynqx` | comm | `common/asynqx/` (5 files: Client, SchedulerServer, TaskServer) |
| 4 | `bytex` | domain | `common/bytex/bytex.go` |
| 5 | `carbonx` | domain | `common/carbonx/carbonx.go` |
| 6 | `configx` | infra | `common/configx/` (kqConfig.go, mockconfig.go) |
| 7 | `copierx` | infra | `common/copierx/type.go` |
| 8 | `ctxdata` | infra | `common/ctxdata/ctxData.go` |
| 9 | `ctxprop` | infra | `common/ctxprop/` (claims.go, ctx.go, grpc.go) |
| 10 | `dbx` | infra | `common/dbx/` (dbx.go, sqlitesql.go, taossql.go) |
| 11 | `djisdk` | domain | `common/djisdk/` (15 files: Client, Handler, DRC, Protocol, GeoFence, Topics) |
| 12 | `dockerx` | infra | `common/dockerx/dockerx.go` |
| 13 | `einox` | ai | `common/einox/` (12 sub-packages: agent, checkpoint, knowledge, memory, model, middleware, runtime, tool, etc.) |
| 14 | `executorx` | domain | `common/executorx/chunkmessagespusher.go` |
| 15 | `filex` | infra | `common/filex/` (filex.go, md5.go + tests) |
| 16 | `gisx` | domain | `common/gisx/` (7 files: gisx.go, store.go, validate.go, geos/, doc.go) |
| 17 | `gnetx` | comm | `common/gnetx/` (30 files: Server, Client, Dialer, Codec×3, Session, Router, Handler, etc.) |
| 18 | `gormx` | infra | `common/gormx/` (19 files: batch, callbacks, config, db, delete, driver, hooks, logger) |
| 19 | `gtwx` | infra | `common/gtwx/` (cors.go, errorhandler.go, openai_error.go) |
| 20 | `iec104` | domain | `common/iec104/` (client/, server/, types/, util/, waitgroup/, log.go) |
| 21 | `imagex` | infra | `common/imagex/` (exifx, imaging.go) |
| 22 | `Interceptor` | infra | `common/Interceptor/` (rpcclient/, rpcserver/) |
| 23 | `lalx` | domain | `common/lalx/laltype.go` |
| 24 | `mcpx` | ai | `common/mcpx/` (8 files: client, server, auth, config, async_result, wrapper, memory_handler) |
| 25 | `mediax` | infra | `common/mediax/mediax.go` |
| 26 | `modbusx` | domain | `common/modbusx/` (client.go, config.go) |
| 27 | `mqttx` | comm | `common/mqttx/` (10 files: client, config, dispatcher, message, reply_router, topic_log) |
| 28 | `nacosx` | infra | `common/nacosx/` (7 files: builder, config, options, register, resolver, target) |
| 29 | `netx` | comm | `common/netx/` (16 files: client, download, upload, encode, request, response, transport) |
| 30 | `ossx` | infra | `common/ossx/` (8 files: minio_oss, md5, stream, template_resolver, ossconfig/) |
| 31 | `powerwechatx` | domain | `common/powerwechatx/types.go` |
| 32 | `skillmd` | domain | `common/skillmd/` (skillmd.go + test) |
| 33 | `socketiox` | comm | `common/socketiox/` (5 files: server, handler, container + test) |
| 34 | `ssex` | comm | `common/ssex/` (writer.go + test) |
| 35 | `stream` | infra | `common/stream/` (stream.go, grpc_sender.go) |
| 36 | `tool` | infra | `common/tool/` (backoff.go, errorutil.go, idutil.go, util.go + tests) |
| 37 | `trace` | infra | `common/trace/carrier.go` |
| 38 | `wsx` | comm | `common/wsx/` (client.go, config.go, errors.go + test) |

## SPECS_FOUND (Dedicated spec files for common/ packages)

12 of 38 packages have dedicated spec files:

| Package | Dedicated Spec File(s) |
|---------|----------------------|
| `antsx` | `antsx-invoke-guidelines.md`, `antsx-promise-guidelines.md`, `antsx-replypool-guidelines.md` |
| `bytex` | `bytex-contracts.md` |
| `ctxprop` | `ctxprop-guidelines.md` |
| `djisdk` | `djisdk-guidelines.md` |
| `gisx` | `gisx-guidelines.md` |
| `gnetx` | `gnetx/` (7 files: index, codec, server, client, session, handler, request-response) |
| `gormx` | `gormx-guidelines.md` |
| `iec104` | `iec104-control-commands.md` |
| `mqttx` | `mqttx-guidelines.md` |
| `netx` | `netx-guidelines.md` |
| `socketiox` | `socketiox-guidelines.md`, `socketiox-contracts.md` |
| `wsx` | `wsx-guidelines.md` |

Additionally, the following app/ service-specific specs exist:
- `djicloud-hooks-guidelines.md` — covers `app/djicloud`
- `djicloud-models.md` — covers `app/djicloud` GORM models
- `drc-concurrency.md` — covers DRC Manager in `app/djicloud`
- `geofence-guidelines.md` — covers geofence in `app/djicloud`
- `drone-station-sdk-template.md` — template for new SDK packages
- `mr-concurrency.md` — covers go-zero mr.MapReduce usage patterns
- `database-guidelines.md` — general DB model conventions

## MISSING_COMMON (Packages with NO dedicated spec)

26 of 38 common/ packages have **no dedicated spec file**:

| Package | Risk Level | Notes |
|---------|-----------|-------|
| `alarmx` | LOW | Single file (`alarmx.go`), simple scope |
| `asynqx` | MEDIUM | 5 files, task queue infrastructure (asynq) |
| `carbonx` | LOW | Single file, time formatting |
| `configx` | LOW | Mock config, kq config |
| `copierx` | LOW | Single type definition |
| `ctxdata` | LOW | Single file, context data access |
| `dbx` | MEDIUM | Multi-DB support (SQLite, TDengine/Taos) |
| `dockerx` | LOW | Single file, Docker operations |
| `einox` | **HIGH** | Complex AI agent framework with 12 sub-packages |
| `executorx` | LOW | Single file, chunk message pusher |
| `filex` | LOW | File operations (2 files) |
| `gtwx` | MEDIUM | Gateway error handling (CORS, OpenAI errors) |
| `imagex` | MEDIUM | EXIF parsing, image processing |
| `Interceptor` | MEDIUM | RPC interceptors (client/server) |
| `lalx` | LOW | Single type definition |
| `mcpx` | **HIGH** | MCP client/server (8 files, auth, config, memory) |
| `mediax` | LOW | Media processing (single file) |
| `modbusx` | MEDIUM | Modbus client abstraction (2 files) |
| `nacosx` | MEDIUM | Nacos service discovery (7 files) |
| `ossx` | MEDIUM | OSS object storage (8 files, MinIO) |
| `powerwechatx` | LOW | Single type definition |
| `skillmd` | MEDIUM | Skill metadata (tested) |
| `ssex` | MEDIUM | SSE writer (tested) |
| `stream` | MEDIUM | Stream utilities (grpc sender) |
| `tool` | MEDIUM | General utilities (backoff, errors, IDs) |
| `trace` | MEDIUM | OTel trace carrier |

**Priority candidates for new specs** (based on code volume + architectural significance):
1. **einox** — 12 sub-packages, AI agent framework
2. **mcpx** — 8 files, MCP protocol implementation
3. **nacosx** — 7 files, service discovery infrastructure
4. **ossx** — 8 files with MinIO integration
5. **modbusx** — Modbus client, used by bridgemodbus service
6. **asynqx** — Asynq task queue infrastructure

## MISSING_APP_SERVICES (Services with significant custom logic but NO spec)

### Zero Spec Mentions (18 services)

These services appear in ZERO spec files:

| Service | .go Files | Category | Notable Custom Code |
|---------|-----------|----------|-------------------|
| `app/alarm` | 5 | basic | Alarm service (proto) |
| `app/bridgedump` | 7 | bridge | Cable fault data dump |
| `app/bridgegtw` | 6 | bridge | API gateway for bridges |
| `app/bridgekafka` | 5 | bridge | Kafka protocol bridge |
| `app/bridgemodbus` | 24 | bridge | **Modbus protocol bridge with 24 logic files** |
| `app/bridgemqtt` | 6 | bridge | MQTT protocol bridge |
| `app/iecagent` | 5 | device | IEC104 slave agent |
| `app/iecstash` | 4 | device | IEC104 data stash |
| `app/lalhook` | 26 | streaming | **Webhook handlers: on_sub_start, on_pub_start, on_rtmp_connect, on_relay_pull_stop, etc.** |
| `app/lalproxy` | 12 | streaming | Stream proxy (relay, RTP, group info) |
| `app/logdump` | 5 | basic | Log push service |
| `app/podengine` | 12 | orchestration | Pod lifecycle (create/delete/stop/list/images) |
| `app/xfusionmock` | 9 | test | Device mock (events, alarms, push) |
| `aiapp/aichat` | 15 | ai | OpenAI-compatible chat with provider layer |
| `aiapp/aisolo` | 43 | ai | **Eino Agent: modes, sessions, skills, turn management, workdir** |
| `aiapp/mcpserver` | 9 | ai | MCP Server with tools + skills |
| `aiapp/ssegtw` | 10 | ai | SSE streaming gateway |
| `socketapp/socketgtw` | 17 | socket | SocketIO handshake/router gateway |
| `socketapp/socketpush` | 16 | socket | WebSocket push service |

### Partial Spec Coverage (services mentioned but no dedicated spec)

| Service | Mentions In | Notes |
|---------|------------|-------|
| `app/ieccaller` (24 files) | `iec104-control-commands.md`, `mqttx-guidelines.md` | Covered indirectly by IEC104 spec |
| `app/gis` (30 files) | `gisx-guidelines.md` | gisx spec covers common/gisx, not app/gis service logic |
| `app/file` (22 files) | `netx-guidelines.md` | Tangential mention only |
| `app/trigger` (64 files) | `go-zero-conventions.md`, `netx-guidelines.md` | **Largest service with 64 .go files; only minor mentions** |
| `aiapp/aigtw` (65 files) | `error-handling.md` | **Largest AI service; only tangential error-handling mention** |

### Priority Candidates for New App Service Specs

1. **app/trigger** (64 files) — Trigger orchestration engine with planscope, cron, task payloads, invoker, execdelay
2. **aiapp/aigtw** (65 files) — AI gateway aggregating backends with custom error handling
3. **aiapp/aisolo** (43 files) — Eino Agent with complex session/mode/skill architecture
4. **app/bridgemodbus** (24 files) — Full Modbus protocol bridge
5. **app/lalhook** (26 files) — Streaming webhook handler
6. **socketapp/socketgtw** (17 files) — SocketIO gateway (could leverage socketiox spec)
7. **socketapp/socketpush** (16 files) — Push service

## PLACEHOLDER_ISSUES

**Result: ALL CLEAR — No placeholder content found in any spec file.**

Initial grep scans flagged files like `iec104-control-commands.md` (25 hits), `drone-station-sdk-template.md` (10 hits), `djisdk-guidelines.md` (4 hits), etc. However, **every match is a false positive** — they are naming convention references in code examples:

- `WithXxx` / `SendXxxCommand` / `XxxSliceToYyySlice` — Go naming convention patterns
- `OnXxx` / `ackXxxValue` — Handler registration patterns
- `app/<vendor>/internal/hooks/` — Template placeholder in drone-station-sdk-template (correct usage for a template)

All existing 31 spec files contain substantive, codebase-backed content with no "TODO", "TBD", "to be filled", or "placeholder" markers indicating incomplete sections.

## GNETX_STATUS: Excellent — Specs match code

### Completeness Check

The gnetx spec index (`gnetx/index.md`) lists 15 source files. **All 15 exist** in `common/gnetx/`:

| Spec Claims | Actual Code | Status |
|------------|-------------|--------|
| `codec*.go` | `codec.go`, `codec_lengthprefix.go`, `codec_delimiter.go`, `codec_fixed.go` | ✓ Match |
| `serializer.go` | ✓ Exists | ✓ Match |
| `server.go` | ✓ Exists (349 lines) | ✓ Match |
| `client.go` | ✓ Exists (327 lines) | ✓ Match |
| `dialer.go` | ✓ Exists | ✓ Match |
| `session.go` | ✓ Exists | ✓ Match |
| `handler.go` | ✓ Exists | ✓ Match |
| `router.go` | ✓ Exists (165 lines) | ✓ Match |
| `message.go` | ✓ Exists | ✓ Match |
| `errors.go` | ✓ Exists | ✓ Match |
| `options.go` | ✓ Exists | ✓ Match |
| `idle.go` | ✓ Exists | ✓ Match |
| `logger.go` | ✓ Exists | ✓ Match |
| `trace.go` | ✓ Exists (48 lines) | ✓ Match |
| `doc.go` | ✓ Exists (77 lines) | ✓ Match |

### Accuracy Check

| Aspect | Assessment |
|--------|-----------|
| Interface definitions | Accurate — `Codec`, `Serializer`, `CodecConn`, `Handler`, `Conn` all match actual code signatures |
| Thread model docs | Correct — server.md documents on-loop vs off-loop constraints exactly as implemented |
| Line number references | **Slightly stale** (off by 1-3 lines in codec.go, consistent with minor code evolution since spec was written) |
| Composite TID mechanism | Accurate — session.md and request-response.md describe the `sessionID + "|" + msg.TID()` pattern exactly as in `session.go:103-120` |
| ReplyPool ownership | Correct — ownership table in session.md matches actual Server/Client/Dialer initialization |
| Half-packet handling | Correct — `ErrIncompletePacket` contract documented accurately |
| Architecture diagram | Accurate — package layout in index.md matches actual file tree |

### Verdict

The gnetx specs are among the highest-quality in the repository. All 6 sub-specs (codec, server, client, handler, session, request-response) are comprehensive, code-backed, and consistent with the actual implementation. Only minor line number drift (1-3 lines) observed, which is expected maintenance overhead.

## ANY_OTHER_GAPS

### 1. Common packages with zero code yet still listed

- `executorx` — only `chunkmessagespusher.go` (very thin)
- `gtwx` — 3 small files but handles CORS, error handling, OpenAI errors (gateway-critical)
- `lalx` — single `laltype.go` (type-only)
- `powerwechatx` — single `types.go` (type-only)

### 2. Specs that could benefit from expansion

- `gisx-guidelines.md` — covers `common/gisx` well, but does NOT cover `app/gis` service logic (30 files: fence CRUD, H3 encoding, geohash, transform, nearby fences)
- `messaging-guidelines.md` — general messaging, but doesn't cover Kafka/bridge patterns used by `bridgekafka`, `iecstash/kafka/`
- `database-guidelines.md` — general DB conventions, but doesn't cover multi-DB patterns in `dbx/`

### 3. ai/ domain gap

The `aiapp/` services and `common/einox`, `common/mcpx` have substantial code (einox has 12 sub-packages) but no spec coverage. This is the largest un-documented domain area.

### 4. Service categories poorly represented

| Category | Services | Spec Coverage |
|----------|----------|--------------|
| DJI Cloud | djicloud | **Well-covered** (4 spec files) |
| IEC104 | iecagent, ieccaller, iecstash | **Moderately covered** (1 spec file) |
| Bridge/Protocol | bridgemodbus, bridgemqtt, bridgekafka, bridgedump, bridgegtw | **Uncovered** |
| Streaming | lalhook, lalproxy | **Uncovered** |
| AI | aichat, aigtw, aisolo, mcpserver, ssegtw | **Uncovered** |
| Socket | socketgtw, socketpush | **Uncovered** (only socketiox spec exists) |
| Infrastructure | alarm, file, logdump, podengine | **Uncovered** |
| Orchestration | trigger | **Uncovered** |

## Caveats

1. Two common/ directories treated as packages: `common/Interceptor/` (capital I), `common/`. Both are functional Go packages.
2. Some "missing" services (iecagent, iecstash) are thin wrappers (4-5 .go files) and may not warrant separate specs.
3. `drone-station-sdk-template.md` is a template, not a codebase spec — it's intentionally generic.
4. The `socketiox-contracts.md` spec covers SocketIO event contracts but does not cover the actual `socketapp/socketgtw` or `socketapp/socketpush` service implementations.
