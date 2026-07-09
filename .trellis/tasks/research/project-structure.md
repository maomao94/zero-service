# Research: Project Structure & Coding Conventions

- **Query**: Explore project structure, module setup, common/ packages, interface patterns, rrule-go dependency, and spec guidelines
- **Scope**: internal
- **Date**: 2026-07-09

## Findings

### 1. Module Name

- **Module**: `zero-service`
- **Go version**: 1.26.0
- **Source**: `go.mod` line 1-3
- **Local replace**: `github.com/Azure/go-workflow` → `/Users/hehanpeng/GolandProjects/go-workflow`

### 2. common/ Directory Structure (41 entries)

```
common/
├── alarmx/          # 告警工具
├── antsx/           # 并行任务编排 (Promise, Invoke, ReplyPool)
├── asynqx/          # asynq 任务队列
├── bytex/           # Modbus 字节/寄存器工具 (Integer generic constraint)
├── carbonx/         # 时间格式化工具
├── configx/         # 配置工具
├── copierx/         # 对象深拷贝
├── ctxdata/         # 上下文数据存取
├── ctxprop/         # gRPC/JWT/MCP 上下文传播
├── dbx/             # 多数据库扩展 (多库路由, 租户隔离)
├── djisdk/          # DJI Cloud SDK (Client, Handler, DRC, Topic)
├── dockerx/         # Docker 操作封装
├── einox/           # Eino AI Agent 框架封装
├── executorx/       # 任务执行器
├── filex/           # 文件操作
├── flowx/           # Azure go-workflow 封装 (4 files)
├── gisx/            # GIS 空间计算 (FenceStore interface)
├── gnetx/           # TCP 框架: Codec/Server/Client/Dialer/Session/Router (31 files, 实验性)
├── gormx/           # GORM 增强 (BaseModel, 时间钩子, 分页, AuditUserID interface)
├── gtwx/            # 网关错误处理与路由工具
├── iec104/          # IEC104 协议 (ASDUCall interface, CommandHandler interface)
├── imagex/          # 图片处理
├── Interceptor/     # RPC 拦截器 (日志、异常恢复)
├── isp/             # ISP 协议
├── lalx/            # 直播/流媒体工具
├── mcpx/            # MCP 客户端/服务端 (AsyncResultStore, TaskObserver interfaces)
├── mediax/          # 媒体处理
├── modbusx/         # Modbus 协议工具
├── mqttx/           # MQTT 客户端 (Client interface, 10 files)
├── nacosx/          # Nacos 服务发现 (7 files)
├── netx/            # HTTP 客户端 (Engine interface, 16 files)
├── ossx/            # OSS 对象存储 (OssTemplate interface + MinioTemplate impl, 9 files)
├── powerwechatx/    # 企业微信集成
├── skillmd/         # 技能元数据
├── socketiox/       # SocketIO (EventHandler interface, 5 files)
├── ssex/            # SSE 流式输出
├── stream/          # 流处理 (Sender, JSONSender, ChunkSender interfaces)
├── tool/            # 通用工具 (错误构建, 类型转换)
├── trace/           # 链路追踪
├── wsx/             # WebSocket 客户端 (Client interface, 4 files)
└── type.go          # 公共类型定义 (DateTime)
```

### 3. rrule-go Dependency Status

- **Already a dependency**: `github.com/teambition/rrule-go v1.8.2` (line 64 of `go.mod`)
- **Confirmed**: ✅ Yes, it is already in go.mod

### 4. Coding Conventions Observed

#### 4a. Package Naming

- All common packages use short, descriptive names suffixed with `x` (e.g., `mqttx`, `wsx`, `ossx`, `netx`, `flowx`, `gormx`). Exceptions: `iecastd`, `tool`, `stream`, `trace`, `interceptor`.
- Package directory names are lowercase, no underscores, no mixed case.
- Each package is a self-contained directory, not a single file.

#### 4b. File Organization (within a common/ package)

**Typical structure** for a client/SDK-style package (e.g., `mqttx`, `wsx`, `netx`):

| File | Role |
|---|---|
| `client.go` | Main struct + constructor(s) + public API methods |
| `config.go` | Configuration struct + defaults normalization |
| `options.go` or in `config.go` | `XxxOptions` struct + `XxxOption` func type + `WithXxx(...)` factory functions |
| `errors.go` | Sentinel errors via `errors.New()` |
| `README.md` | Optional overview |
| `*_test.go` | Unit tests |

For interface+implementation packages (e.g., `ossx`):

| File | Role |
|---|---|
| `ossx.go` | Interface definition + factory function + shared types |
| `minio_oss.go` | Concrete implementation of the interface |
| `md5.go` | Helper utilities |
| `stream.go` | Streaming utilities |

#### 4c. Interface Patterns

**Pattern 1: Interface + Single Implementation (Export interface, hide struct)**

Used by: `mqttx`, `wsx`

```go
// client.go
type Client interface {
    Send(ctx context.Context, msg []byte) error
    Close() error
    State() ConnState
}

type client struct { ... } // unexported

// Constructor returns the interface, not the concrete type:
func NewClient(cfg Config, opts ...ClientOption) (Client, error) { ... }
func MustNewClient(cfg Config, opts ...ClientOption) Client { ... }
```

**Pattern 2: Interface + Multiple Implementations via Category**

Used by: `ossx`

```go
// ossx.go
type OssTemplate interface {
    MakeBucket(ctx context.Context, tenantId, bucketName string) error
    PutFile(...) (*File, error)
    // ...more methods
}

// Factory function dispatches by category:
func NewTemplate(config *Config, ossRule OssRule) (OssTemplate, error) {
    switch config.Category {
    case Category_Minio:
        return NewMinioTemplate(config, ossRule)
    default:
        return nil, fmt.Errorf("unsupported oss category: %d", config.Category)
    }
}
```

MinioTemplate is the exported concrete struct implementing OssTemplate. The interface `_ = OssTemplate(*MinioTemplate)` compile-time check is at `ossx.go:41`.

**Pattern 3: Small functional interfaces (Go-composition style)**

```go
// mqttx/dispatcher.go
type ConsumeHandler interface { ... }

// wsx/config.go
type MessageHandler interface {
    HandleMessage(ctx context.Context, msg []byte) error
}
type MessageHandlerFunc func(ctx context.Context, msg []byte) error  // adapter
```

**Pattern 4: Engine abstraction (Dependency injection interface)**

Used by: `netx`

```go
// netx/transport.go
type Engine interface {
    Do(ctx context.Context, req *Request) (*Response, error)
}
// DefaultEngine implements it; callers inject WithEngine
```

#### 4d. Constructor Patterns (Canonical)

All constructors follow the same pattern across the codebase:

```go
// 1. Options struct (separate from runtime struct)
type XxxOptions struct {
    Field1 Type1
    Field2 Type2
}

// 2. Option function type
type XxxOption func(*XxxOptions)

// 3. New constructor: options → runtime struct
func NewClient(cfg Config, opts ...XxxOption) (*Client, error) {
    o := defaultXxxOptions()  // or &XxxOptions{}
    for _, opt := range opts { opt(o) }
    // map options to client fields
}

// 4. Must variant (go-zero style)
func MustNewClient(cfg Config, opts ...XxxOption) Client {
    c, err := NewClient(cfg, opts...)
    logx.Must(err)
    proc.AddShutdownListener(func() { c.Close() })
    return c
}
```

**Key rule from spec** (`coding-standards.md` line 96-125): Options must write to `XxxOptions` struct, NOT directly to the runtime `Client` struct. This is the "Good/Base/Bad" contract:

- ✅ Good: `type ClientOption func(*ClientOptions)` 
- ❌ Bad: `type ClientOption func(*Client)` 

#### 4e. Error Handling

Two-tier approach:

1. **Sentinel errors** for package-level conditions (in `errors.go`):
   ```go
   var ErrNotConnected = errors.New("[wsx] not connected to server")
   var ErrNilDecoder = errors.New("mqttx: reply decoder cannot be nil")
   ```
   Convention: prefix with package name `[wsx]` or `mqttx:`.

2. **Project error codes** for service-level (Logic layer) errors:
   ```go
   tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
   tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "action desc")
   ```

**Critical rule** from `error-handling.md` line 132-133: Logic layer must NOT log error/warn — only return error. The `LoggerInterceptor` handles all logging uniformly with `%+v`.

#### 4f. Testing Conventions

- Test files co-located with source: `*_test.go`
- Table-driven tests are the norm
- External dependencies mocked via interfaces (e.g., `Clock` injection in `flowx`)

#### 4g. Dependencies

- Common packages depend only on: standard library, other `common/` packages (same level or sub-level), and third-party libraries already in `go.mod`
- Common packages must NOT depend on `app/` or `model/`

### 5. Key Spec Guidelines for Creating a New common/ Package

From `.trellis/spec/backend/coding-standards.md` and `.trellis/spec/backend/directory-structure.md`:

1. **Reuse first**: Search for existing similar functionality in `common/` before creating a new package (`code-reuse-thinking-guide.md`)
2. **Only for cross-service use**: A new `common/` package must be used by multiple `app/*` services. Single-service helpers stay in `internal/` (`directory-structure.md` line 147-151)
3. **client Option pattern**: `ClientOption func(*ClientOptions)` — never `func(*Client)` (`coding-standards.md` `Convention: Client Option 构造配置边界`)
4. **Generic for repeated patterns only**: Use Go generics to eliminate repetition, not for cleverness (`coding-standards.md` line 83-87)
5. **No Java style**: No getters/setters, no exception-style error handling, no Result wrappers, no Builder patterns unless Go-idiomatic
6. **Sentinel errors**: Package errors as `var ErrXxx = errors.New("[pkg] description")` in a separate `errors.go`
7. **Follow adjacent patterns**: Read existing packages like `mqttx`, `wsx`, `ossx` for file layout and constructor conventions
8. **No common/ dependency on app/ or model/**: Strict one-way dependency (`directory-structure.md` line 156)
9. **go-zero `Must*` style**: Provide `MustNewClient` that panics on error + registers shutdown listener when appropriate
10. **Validate early**: In `gen.sh` / compiled contracts, validate before running. Run `go build ./...` and `go mod tidy` after changes

## Caveats / Not Found

- No `common/` package currently has tests for the full interface-contract compliance across implementations (only `ossx` approaches this with the compile-time check). This may be expected given most packages have only one implementation.
- The `rrule-go` library at `v1.8.2` appears to be a direct dependency (not indirect), though grep did not find it imported in any `.go` file in the repo currently — it may be used transitively or recently added but not yet referenced.
- Some packages (e.g., `sscx`, `asynqx`, `dockerx`) have no README.md — documentation conventions are inconsistent across packages.
