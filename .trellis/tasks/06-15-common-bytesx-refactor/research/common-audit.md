# Research: `common/` Directory Audit — Unification & Deduplication Opportunities

- **Query**: Audit common/ directory for code unification and deduplication opportunities
- **Scope**: Internal (code search)
- **Date**: 2026-06-15

## Findings

### 1. Package Inventory & Function Counts

| Package | Files | Exported Functions | Category |
|---|---|---|---|
| `alarmx/` | 1 | 14 | Lark alarm integration |
| `antsx/` | 11 | 363 | High-level goroutine primitives (promise, stream, emitter, reactor) |
| `asynqx/` | 5 | 19 | Asynq task queue client/server |
| `bytex/` | 1 | 19 | Bytes <-> uint16/int16/uint32/int32/bool conversions |
| `carbonx/` | 1 | 1 (init) | Carbon timezone/locale init |
| `configx/` | 2 | 6 | Kafka consumer config, MockConfig |
| `copierx/` | 1 | 0 | copier Option (type converters) — exported var only |
| `ctxdata/` | 1 | 5 | Context keys/getters |
| `ctxprop/` | 3 | 10 | Context propagation (gRPC MD, JWT claims, MCP _meta) |
| `dbx/` | 3 | 15 | Database connection factory (MySQL/Postgres/SQLite/TDengine) |
| `djisdk/` | 8 | 199 | DJI SDK protocol client |
| `dockerx/` | 1 | 6 | Docker client/container helpers |
| `einox/` | 11 subdirs | 475 | AI agent framework (LLM, MCP tools, knowledge, memory) |
| `executorx/` | 1 | 3 | Chunked message pusher |
| `filex/` | 2 | 27 | File I/O (capture, MD5, copy, temp) |
| `gisx/` | 1 | 3 | GIS/H3 geo-conversion |
| `gormx/` | ~15 | 201 | GORM wrapper (config, tenant, audit, batch, upsert, pagination) |
| `gtwx/` | 3 | 12 | Gateway error handler, CORS |
| `iec104/` | 8 | 139 | IEC 104 protocol (client, server, types) |
| `imagex/` | 4 | 45 | Image processing (EXIF, imaging) |
| `Interceptor/` | 2 | 5 | gRPC interceptors (metadata, logger) |
| `lalx/` | 1 | 0 | LAL streaming server types (structs only) |
| `mcpx/` | 8 | 101 | MCP protocol (JSON-RPC, server, client) |
| `mediax/` | 1 | 8 | Video screenshot via FFmpeg |
| `modbusx/` | 2 | 23 | Modbus TCP client + pool |
| `mqttx/` | 8 | 84 | MQTT client (dispatcher, reply router, topic log) |
| `nacosx/` | 6 | 23 | Nacos service discovery (resolver, register, config) |
| `netx/` | 9 | 168 | HTTP client (request, response, transport, encode, download, upload) |
| `ossx/` | 6 | 56 | OSS abstraction (MinIO template, config cache) |
| `powerwechatx/` | 1 | 13 | PowerWeChat log adapter |
| `skillmd/` | 1 | 4 | Skill markdown parser |
| `socketiox/` | 4 | 73 | Socket.IO server |
| `ssex/` | 1 | 18 | SSE writer |
| `stream/` | 2 | 33 | Stream sender interface + gRPC sender implementation |
| `tool/` | 4 | 53 | Mixed utilities (error, ID gen, backoff, time, pagination, tokens) |
| `trace/` | 1 | 10 | OpenTelemetry carrier wrappers |
| `wsx/` | 4 | 80 | WebSocket client |
| `type.go` (root `common`) | 1 | 2 | `DateTime` type with JSON marshal |

**Missing packages** (referenced in task request but do not exist):
- `iox/` — **no files found**, does not exist
- `streamx/` — **no files found**, does not exist

---

### 2. Identified Duplication

#### A. CRITICAL: `bytex/` is fully duplicated inside `tool/util.go` (lines 300–468)

**Files:**
- `common/bytex/bytex.go` — entire file (238 lines, 19 exported functions)
- `common/tool/util.go` — lines 300–468 (exact copy of all structs + functions)

**Duplicated types & functions (13 functions + 2 structs):**

| Symbol | bytex/ location | tool/util.go location |
|---|---|---|
| `BinaryValues` (struct) | `bytex.go:8` | `util.go:301` |
| `BitValues` (struct) | `bytex.go:16` | *(not in tool)* |
| `BytesToUint16Slice` | `bytex.go:25` | `util.go:312` |
| `Uint16SliceToBytes` | `bytex.go:45` | `util.go:332` |
| `Uint16ToInt16` | `bytex.go:57` | `util.go:344` |
| `Uint16SliceToInt16Slice` | `bytex.go:61` | `util.go:348` |
| `Uint16ToUint32` | `bytex.go:73` | `util.go:360` |
| `Uint16ToInt32` | `bytex.go:77` | `util.go:364` |
| `Uint16SliceToUint32Slice` | `bytex.go:81` | `util.go:368` |
| `Uint16SliceToInt32Slice` | `bytex.go:89` | `util.go:376` |
| `Int16SliceToInt32Slice` | `bytex.go:97` | `util.go:384` |
| `BytesToBinaryValues` | `bytex.go:136` | `util.go:395` |
| `Uint16SliceToBinaryValues` | `bytex.go:166` | `util.go:421` |
| `BytesToBools` | `bytex.go:194` | `util.go:446` |
| `BoolsToBytes` | `bytex.go:207` | `util.go:459` |

**Impact:** 91 lines (including comments) of exact duplicate code across 2 packages.

**Note:** `tool/util.go` is **missing** `BytesToBitValues` and `BoolsToBitValues` (only in `bytex.go`).

---

#### B. `SimpleUUID` — duplicated within the same package

| Location | Signature | Notes |
|---|---|---|
| `tool/idutil.go:53` | `func (u *IdUtil) SimpleUUID() (string, error)` | Method on `IdUtil` |
| `tool/util.go:134` | `func SimpleUUID() (string, error)` | Standalone function |

All callers (mcpx, modbusx) call `tool.SimpleUUID()` — the standalone version. **The method on `IdUtil` is effectively dead code** (never called externally).

---

#### C. `ParseDatabaseType` — similar logic in dbx and gormx

| Location | File |
|---|---|
| `common/dbx/dbx.go:31` | `func ParseDatabaseType(datasource string) DatabaseType` |
| `common/gormx/driver.go:21` | `func ParseDatabaseType(dsn string) DatabaseType` |

Both parse a datasource string to determine DB type (MySQL/Postgres/SQLite/TAOS). They have **different implementation** (dbx checks for `@tcp(`, gormx checks for `charset=`), but serve the same purpose. The `dbx` version also handles TDengine (`taos`), while `gormx` version does not.

---

#### D. Token estimation — only in `tool/util.go` (no duplicates)

`EstimateTokens`, `EstimateMessagesTokens`, `CountSignificantDigits` at `tool/util.go:481–543` — unique to this file, no duplicates elsewhere.

---

#### E. No orphaned/commented-out functions found

The `rg '^//\s*func\s'` search returned zero results across `common/`.

---

### 3. Small/Mergeable Packages

| Package | Functions | Lines | Assessment |
|---|---|---|---|
| `carbonx/` | 1 (init) | 16 | Single init func. Could be moved into the packages that use carbon, or into a shared location. |
| `copierx/` | 0 (exported var only) | 56 | Only exports an `Option` var. Could be merged into `tool/` or another relevant package. |
| `executorx/` | 3 | 44 | Single `ChunkMessagesPusher` — thin wrapper around go-zero `ChunkExecutor`. Low value as standalone package. |
| `executorx/` vs `stream/` | Both deal with chunked sending | | `ChunkMessagesPusher` in `executorx` chunks by byte size; `GRPCSender` in `stream` sends as gRPC chunks. Different concerns but both about "chunked sending." |
| `gisx/` | 3 | 59 | H3 geo conversion. Specialized, fine standalone. |
| `lalx/` | 0 (structs only) | 125 | Data types only. Could be merged into the caller or kept separate per Go convention. |
| `powerwechatx/` | 13 | 65 | Log adapter interface for PowerWeChat. Fine standalone. |

**`carbonx/`** is the best candidate for merging — it's a single init function setting global defaults for the carbon library. Could be an `init()` in a shared package or inlined where carbon is first imported.

---

### 4. Notable Patterns

#### Function categories across all packages

| Category | Packages | Notes |
|---|---|---|
| **Bytes <-> numeric** | `bytex/`, `tool/` | **DUPLICATED** |
| **HTTP client** | `netx/` | Rich, self-contained; no duplication |
| **Stream sending** | `stream/`, `ssex/`, `executorx/` | Different protocols (generic, SSE, chunked) |
| **Config** | `configx/`, `nacosx/` | Separate concerns |
| **Database** | `dbx/`, `gormx/` | Different ORMs (sqlx vs gorm) with overlapping DSN parsing |
| **Context propagation** | `ctxdata/`, `ctxprop/`, `Interceptor/` | Well-factored, clean separation |
| **File I/O** | `filex/`, `ossx/` | `filex` is local file ops, `ossx` is object storage |
| **Protocol clients** | `modbusx/`, `mqttx/`, `socketiox/`, `wsx/`, `djisdk/`, `iec104/` | All distinct, each wraps a different protocol |
| **AI/LLM** | `einox/`, `mcpx/`, `skillmd/` | Distinct layers |
| **Alarm** | `alarmx/` | Lark-based alarm, self-contained |
| **Timestamp/uuid** | `tool/` | `GenSecondTS`, `GenMilliTS`, `GenMicroTS`, `EncodeBase62`, `ShortPath` |
| **Pagination** | `tool/` | `CalculateOffset` (also in gormx/pagination.go) |

---

### 5. Recommended Unification Actions

1. **HIGH — Remove bytes conversion duplicate from `tool/util.go`**
   - Delete lines 300–468 (the `BinaryValues` struct, all byte conversion functions) from `tool/util.go`
   - Replace callers of `tool.BytesToUint16Slice()` etc. with `bytex.BytesToUint16Slice()` etc.
   - Keep `BitValues` and `BytesToBitValues`/`BoolsToBitValues` in `bytex/` only

2. **MEDIUM — Remove `IdUtil.SimpleUUID()` method**
   - Delete `tool/idutil.go:52-59` (the method on `IdUtil`)
   - All existing callers use `tool.SimpleUUID()` already

3. **LOW — Consider merging `carbonx/` into a shared bootstrap package**
   - Single init function setting carbon global defaults
   - Could go into a `common/bootstrap/` or similar

4. **LOW — Consider merging `copierx/` into `tool/`**
   - Only exports a `copier.Option` variable with type converters
   - Would fit naturally in `tool/` as a utility

5. **LOW — Evaluate unifying `ParseDatabaseType` between `dbx/` and `gormx/`**
   - Both parse DSN strings to determine database type
   - Could be extracted to a shared internal package
   - But note: different logic, different DatabaseType enum, and different consumers

6. **INFO — No orphaned/commented-out code found**

---

## Caveats

- `iox/` and `streamx/` directories do NOT exist in this codebase — they may have been planned but never created, or were already removed.
- The function counts above are from `rg '^func '` which includes both exported and unexported functions.
- The `copierx/` and `lalx/` packages have 0 exported functions but export important variables (`copierx.Option`) and types (`lalx.*SessionInfo`), so they are not truly empty.
