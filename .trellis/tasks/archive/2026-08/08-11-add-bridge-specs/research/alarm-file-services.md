# Research: alarm & file services

- **Query**: Analyze app/alarm and app/file services including their common packages
- **Scope**: internal (codebase analysis)
- **Date**: 2026-08-11

---

## 1. app/alarm/ — Alarm Service

### 1.1 Overview

A go-zero gRPC service for sending alarms to Feishu (Lark) IM. Creates chat groups, sends interactive alarm cards, and handles via commented-out event handlers.

### 1.2 Entry Point (`app/alarm/alarm.go`)

- go-zero gRPC server
- `tool.PrintGoVersion()` on startup
- Registers `alarm.AlarmServer` via goctl-generated code
- Enables gRPC reflection in DevMode/TestMode
- No Nacos registration, no interceptor — simplest service in the repo

### 1.3 Proto Definition (`app/alarm/alarm.proto`)

- package: `alarm`, go_package: `./alarm`
- Messages: `Req/Res` (Ping/Pong), `AlarmReq` (chatName, description, title, project, dateTime, alarmId, content, error, userId, ip), `AlarmRes` (empty)
- Service: `Alarm` with 2 RPCs: `Ping`, `Alarm`

### 1.4 Server Layer (`internal/server/alarmserver.go`)

goctl-generated (v1.8.5). Standard pattern:
- `AlarmServer` embeds `svc.ServiceContext` + `alarm.UnimplementedAlarmServer`
- Each RPC method: `logic.NewXxxLogic(ctx, svcCtx).Xxx(in)`

### 1.5 Logic Layer (`internal/logic/`)

| File | Behavior |
|---|---|
| `pinglogic.go` | Returns `{Pong: "pong"}` |
| `alarmlogic.go` | Core alarm: merge config UserId, deduplicate, create/update chat via AlarmX, send card message. Also contains helpers `DoP2ImMessageReceiveV1` (handles /solve command, renames chat with [solved] prefix), `getChatInfo`, `updateChatName` |

### 1.6 ServiceContext (`internal/svc/servicecontext.go`)

```go
type ServiceContext struct {
    Config      config.Config
    Httpc       httpc.Service
    RedisClient *redis.Redis
    AlarmX      *alarmx.AlarmX
}
```

Initialized with: `redis.MustNewRedis`, `httpc.NewService("httpc")`, `alarmx.NewAlarmX(larkClient, redisClient)` using `lark.NewClient` with a custom HTTP client wrapping go-zero httpc.

### 1.7 Config (`internal/config/config.go`)

```go
type Config struct {
    zrpc.RpcServerConf
    Alarmx struct {
        AppId, AppSecret, EncryptKey, VerificationToken string
        UserId []string
        Path   string
    }
}
```

### 1.8 YAML Config (`etc/alarm.yaml`)

- `Name: alarm.rpc`, `ListenOn: 0.0.0.0:21011`, `Mode: dev`
- Redis: single node at `127.0.0.1:6379`, key `alarm`
- Alarmx: hardcoded dev credentials, `UserId: [bc243e9d]`, `Path: ./app/alarm/alarm.json`

### 1.9 Alarm Card (`alarm.json`)

Feishu interactive card JSON template with placeholders: `${title}`, `${project}`, `${dateTime}`, `${alarmId}`, `${content}`, `${error}`, `${ip}`, `${button_name}`. Rendered by `alarmx.buildCard()`. Also has a `card.json` (visible as a separate, similar template).

### 1.10 Key Conventions / Patterns

- No database — purely a gateway to Feishu. No models, no DB.
- Redis for chat ID caching: key format `{appName}:alarm:{chatName}`, TTL 7 days
- Single RPC service — 2 methods (Ping + Alarm)
- go-zero httpc as Lark HTTP client — enables circuit-breaking

---

## 2. common/alarmx/ — Alarm Common Package

### 2.1 Overview

Encapsulates all Feishu/Lark IM operations. Thin wrapper around `larksuite/oapi-sdk-go/v3`.

### 2.2 Structure (`alarmx.go` — 222 lines, single file)

**Core types:**
- `AlarmInfo` — alarm fields (Title, Project, DateTime, AlarmId, Content, Error, UserId, Ip)
- `AlarmX` — holds `*lark.Client` + `*redis.Redis`
- `AlarmxHttpClient` — wraps `httpc.Service` to implement `larkcore.HttpClient`

**Key methods:**
| Method | Description |
|---|---|
| `AlarmChat()` | Get or create chat: Redis lookup → CreateAlertChat or UpdateAlertChat |
| `CreateAlertChat()` | Create Feishu group chat via Lark SDK |
| `UpdateAlertChat()` | Add members to existing chat |
| `SendAlertMessage()` | Build card from JSON template + send interactive message |
| `ImChatCreate/ImChatMembersCreate/ImMessageCreate/ImChatUpdate/ImChatGet` | Thin wrappers around Lark SDK |

**Card rendering (`buildCard`):**
- Reads JSON template from file path
- Replaces `${placeholders}` with actual values
- `EscapeString()` handles JSON character escaping

### 2.3 Dependencies
- `larksuite/oapi-sdk-go/v3` — Lark SDK
- `go-zero/core/stores/redis` — chat ID caching
- `go-zero/rest/httpc` — wrapped HTTP client

---

## 3. app/file/ — File Service

### 3.1 Overview

A go-zero gRPC service managing OSS configurations and file operations (upload, sign, relay, video capture). Only MinIO backend implemented. Supports multi-tenant mode with tenant-prefixed buckets.

### 3.2 Entry Point (`app/file/file.go`)

- go-zero gRPC server with Nacos service registration (optional, gated by `NacosConfig.IsRegister`)
- `tool.PrintGoVersion()` on startup
- Registers `file.FileRpcServer` via goctl
- gRPC reflection in DevMode/TestMode
- Nacos registration with gRPC port metadata
- Unary interceptor: `interceptor.LoggerInterceptor`
- Global log field: `app` = config Name
- Imports: `zero-service/common/interceptor/rpcserver`, `common/carbonx`, `common/nacosx`, `common/tool`

### 3.3 Proto Definition (`app/file/file.proto`)

- package: `file`, go_package: `./file`
- Long proto (291 lines) with Java options
- Key messages: `Req/Res`, `Oss` (12 fields), `File` (with `ImageMeta`), `OssFile`, CRUD request/response pairs, `PutFileReq/Res`, `PutStreamFileReq/Res`, `RelayFileReq/Res`, `CaptureVideoStreamReq/Res`
- Service `FileRpc` with 15 RPCs: Ping, OssDetail, OssList, CreateOss, UpdateOss, DeleteOss, MakeBucket, RemoveBucket, StatFile, SignUrl, PutFile, PutStreamFile (stream), RelayFile, RemoveFile, RemoveFiles, CaptureVideoStream

### 3.4 Server Layer (`internal/server/filerpcserver.go`)

goctl-generated (v1.9.2). Standard pattern like alarm but with 15 methods. `PutStreamFile` is special: uses `stream.Context()` for context construction.

### 3.5 Logic Layer

| File | Purpose |
|---|---|
| `pinglogic.go` | Ping/Pong |
| `ossdetaillogic.go` | Get single OSS config by ID |
| `osslistlogic.go` | List OSS configs with pagination |
| `createosslogic.go` | Create OSS config + invalidate cache |
| `updateosslogic.go` | Update OSS config + invalidate cache |
| `deleteosslogic.go` | Delete OSS config + invalidate cache |
| `makebucketlogic.go` | Create bucket if not exists |
| `removebucketlogic.go` | Remove bucket if exists |
| `statfilelogic.go` | Get file info |
| `signurllogic.go` | Generate signed URL (with validation) |
| `putfilelogic.go` | Upload from local file path |
| `putstreamfilelogic.go` | Upload via gRPC stream (wrapper around ossx.UploadStream) |
| `relayfilelogic.go` | Download source + upload to multiple OSS targets (200MB limit, concurrent) |
| `removefilelogic.go` | Delete single file |
| `removefileslogic.go` | Batch delete files |
| `capturevideostreamlogic.go` | Capture video stream frame → upload to OSS |
| `helper.go` | Shared: `ossOrderBy` (SQL-safe), `calcExpires`, `toPbOss` (model→proto) |
| `stream_upload_helper.go` | Shared: `resolveContentType`, `buildCaptureOptions`, `processUploadResult`, `scheduleThumbGeneration`, `generateAndUploadVariant` |

### Logic patterns:
- Every Logic struct has: `ctx context.Context`, `svcCtx *svc.ServiceContext`, `logx.Logger`
- Constructor: `NewXxxLogic(ctx, svcCtx)` — embeds `logx.WithContext(ctx)`
- OSS operations use `l.svcCtx.GetOssTemplate(ctx, tenantId, code)` which calls `OssTemplateResolver`
- `copier.Copy` for model-to-proto conversion (with `// nolint:errcheck`)
- Timestamp formatting via `carbon.CreateFromStdTime().ToDateTimeString()`
- Error codes via `tool.NewErrorByPbCode(extproto.Code__xxx, msg)`

### 3.6 ServiceContext (`internal/svc/servicecontext.go`)

```go
type ServiceContext struct {
    Config              config.Config
    DB                  *gormx.DB
    Validate            *validator.Validate
    ThumbTaskRunner     *threading.TaskRunner
    Httpc               httpc.Service
    NetClient           *netx.Client
    OssTemplateResolver ossx.TemplateResolver
}
```

Key initialization:
- DB via `gormx.MustOpenWithConf` — auto-migrates Oss table in dev/test
- Validator via `validator.New()` — used in SignUrl for requirement checks
- `ThumbTaskRunner` via `threading.NewTaskRunner(concurrency)` — async thumbnail generation
- `NetClient` via `netx.NewClient(netx.WithEngine(netx.NewHTTPEngine(httpcSvc)))` — reusable HTTP client
- `OssTemplateResolver` via `ossx.NewTemplateResolver(c.Oss.TenantMode, svc.loadOssConfig)`
- `loadOssConfig` queries DB by tenant_id + oss_code

### 3.7 Config (`internal/config/config.go`)

```go
type Config struct {
    zrpc.RpcServerConf
    NacosConfig          struct { IsRegister, Host, Port, Username, PassWord, NamespaceId, ServiceName }
    DB                   gormx.Config
    Oss                  OssConf { TenantMode bool }
    ThumbTaskConcurrency int  // default 2
    Upload               UploadConf
}
type UploadConf struct {
    TempDir                string  // default /opt/data/temp
    KeepTempFiles          bool    // default false
    RelayUploadConcurrency int     // default 4
    Image                  ImageUploadConf
}
type ImageUploadConf struct {
    MaxExifRead int              // default 65536
    Thumb       ImageVariantConf { Enabled, Width, Height }
}
```

### 3.8 YAML Config (`etc/file.yaml`)

- `Name: file.rpc`, `ListenOn: 0.0.0.0:21003`, `Timeout: 600000`
- `Mode: dev`
- Nacos: disabled, host `127.0.0.1:8848`
- DB: MySQL DSN (or PostgreSQL/SQLite — auto-detected), with GORM settings (MaxIdleConns, SlowThreshold, LogLevel, etc.)
- Oss: `TenantMode: true`
- Upload: `TempDir: /opt/data/temp`, `KeepTempFiles: false`, `RelayUploadConcurrency: 4`
- Image: `MaxExifRead: 65536`, Thumb: `Enabled: true, Width: 300, Height: 300`

### 3.9 Model (`model/gormmodel/oss.go`)

```go
type Oss struct {
    gormx.LegacyStringBaseModel  // Id (string UUID), CreateTime, UpdateTime, DeleteTime
    gormx.VersionMixin           // Version (int64, optimistic locking)
    TenantId   string  // uniqueIndex: uk_oss_tenant_id_oss_code
    Category   int     // 1-minio 2-qiniu 3-ali 4-tecent
    OssCode    string  // uniqueIndex with TenantId
    Endpoint, AccessKey, SecretKey, BucketName, AppId, Region, Remark string
    Status     int     // 1-open 2-closed
}
// TableName: "oss"
```

### 3.10 Key Conventions / Patterns

- **OSS template pool with caching**: `ossx.Template()` uses double-checked locking to cache `OssTemplate` per tenant. Cache invalidated when config changes (CreateOss/UpdateOss/DeleteOss call `ossx.CacheInvalidate(tenantId)`)
- **Multi-tenant bucket naming**: `{tenantId}-{bucketName}` when `tenantMode=true`
- **TemplateResolver pattern**: `ossx.NewTemplateResolver(tenantMode, getConfigFn)` creates a `func(ctx, tenantId, code) (OssTemplate, error)` — injected into ServiceContext for DI/testability
- **Stream upload with capture**: `ossx.UploadStream` uses `filex.Capture` to simultaneously stream to OSS, capture head bytes (for MIME detection), and optionally tee to temp file (for thumbnail generation)
- **Async thumbnail generation**: via `threading.TaskRunner` — non-blocking, with `context.WithoutCancel` for background tasks
- **RelayFile concurrency**: uses `antsx.Reactor` goroutine pool, configurable concurrency, single target failure doesn't block others
- **Content type detection**: `ossx.DetectContentType` — caller-specified overrides auto-detection from first 512 bytes
- **SQL-safe ordering**: whitelist map of allowed ORDER BY columns via `clause.OrderByColumn`
- **Carbon for time formatting**: `github.com/dromara/carbon/v2` for DateTime string conversion

---

## 4. common/ossx/ — OSS Common Package

### 4.1 Overview

Provides an abstraction layer over different OSS backends with a unified `OssTemplate` interface, template pool caching, and stream upload utilities.

### 4.2 Files

| File | Purpose |
|---|---|
| `ossx.go` | Core types: `OssTemplate` interface, `File`, `OssFile`, `Config`, template pool with caching, `NewTemplate/MustNewTemplate` factory |
| `minio_oss.go` | MinIO implementation of `OssTemplate` — all methods (MakeBucket, PutObject, SignUrl, RemoveFile, etc.) + MD5 calculation during upload |
| `template_resolver.go` | `TemplateResolver` function type + `NewTemplateResolver` constructor |
| `stream.go` | `UploadStream` — stream upload with head capture and content type detection, `ReadUploadHead`, `DetectContentType` |
| `md5.go` | `UploadWithMD5` — wraps reader with MD5 tee reader, delegates upload to callback |
| `osssconfig/ossconfig.go` | Simple `OssConf` struct with `TenantMode bool` |

### 4.3 OssTemplate Interface

```go
type OssTemplate interface {
    MakeBucket, RemoveBucket, StatFile, BucketExists,
    PutFile (multipart), PutStream, PutObject (generic Reader),
    SignUrl, RemoveFile, RemoveFiles
}
```

Currently only `MinioTemplate` implements it. Category constants: 1=Minio, 2=Qiniu, 3=Ali, 4=Tencent.

### 4.4 Template Caching

- Global map `templatePool` + `ossPool` with `sync.RWMutex`
- `Template()` uses double-checked locking
- Rebuilds when Endpoint/AccessKey/SecretKey change
- `CacheInvalidate(tenantId)` clears cache for a tenant

### 4.5 OssRule

- `fullBucketName(tenantId, bucketName)` — if tenantMode: `{tenantId}-{bucketName}`
- `filename(originalFilename, pathPrefix...)` — generates `{prefix}/YYYYMMDD/{uuid}.ext`

---

## 5. common/filex/ — File Utility Package

### 5.1 Overview

File capture (tee-to-temp), file operations, MD5 digest, and MIME helpers.

### 5.2 Files

| File | Purpose |
|---|---|
| `filex.go` | `HeadCaptureWriter`, `Capture` (tee-to-temp file), `CaptureOptions`, `CopyFile`, `ReadFileHead`, `IsImageContentType`, `RemoveTempFile`, `ExtractFilenameFromURL` |
| `md5.go` | `MD5Digest` + `NewMD5TeeReader` — streaming MD5 calculation |
| `filex_test.go` | Tests for filex |

### 5.3 Key Types

- `Capture` — simultaneously captures head bytes + optionally tees to temp file. Used by `ossx.UploadStream` for MIME detection and thumbnail source.
- `HeadCaptureWriter` — captures first N bytes of stream
- `MD5Digest` — streaming MD5 with `Hex()` output
- `CaptureOptions` — TempDir, TempPattern, NeedTemp, MaxHeadRead

---

## 6. Cross-Cutting Patterns

### Service patterns shared across both:
- go-zero gRPC server
- `tool.PrintGoVersion()` at startup
- gRPC reflection in dev/test modes
- goctl-generated proto + server stubs
- `logx.WithContext(ctx)` in every Logic struct

### Alarm-specific patterns:
- No DB — Redis-only for caching
- Lark SDK wrapped in common/alarmx
- JSON template-based card rendering (file-path driven)

### File-specific patterns:
- GORM + gormx.DB for persistence
- OssTemplate interface + template pool with LRU-like cache invalidation
- Multi-tenant bucket naming
- TemplateResolver DI pattern for testability
- Stream upload with tee-capture for concurrent digest + temp-file writing
- Async thumbnail via threading.TaskRunner
- SQL-safe ordering with whitelist
- Carbon for time formatting
- Nacos service registration (optional)
- RPC interceptor chain
