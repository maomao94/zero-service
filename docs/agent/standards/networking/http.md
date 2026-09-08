# HTTP Client

修改 `common/netx` 或服务的 HTTP client、请求编码、响应解码与 Transport 时读取。

## netx — HTTP 客户端

### 核心抽象

- **`Engine` 接口** — `Do(req *http.Request) (*http.Response, error)`，允许替换底层执行引擎。
- **`DefaultEngine`** — 包装 `*http.Client`。
- **`HTTPCEngine`** — 包装 go-zero `httpc.Service`（支持熔断、中间件）。
- **`Client`** — 主 HTTP 客户端，功能选项模式构建。

依据：`common/netx/client.go`、`common/netx/transport.go`。

### Client 构建

```go
c := netx.NewClient(
    netx.WithEngine(httpcEngine),       // 使用 go-zero httpc.Service
    netx.WithTLSConfig(tlsConfig),       // TLS 配置
    netx.WithHeaders(headers),            // 全局请求头
    netx.WithMaxResponseBytes(10<<20),   // 响应体限制
    netx.WithHTTPClientOption(func(c *http.Client) { ... }), // 额外 HTTP 客户端配置
)
```

- 默认限制: 响应 10MB，上传 32MB。
- `WithHTTPClientOption` 是灵活性兜底，允许直接修改 `*http.Client`。

依据：`common/netx/client.go`。

### Request/Response

- **`Request`** — 链式构建器 + 功能选项模式：
  - 方法: `Get`、`Post`、`Put`、`Delete`、`Patch`、`Head`、`Options`
  - Body 类型: raw bytes、JSON struct、form map、io.Reader
  - Query 参数、Header、FormData 均可配置
- **`Response`** — 永远不会返回 error：
  - 网络错误或超时都捕获在 `Response.Err` 字段中
  - 状态码: 504 (超时)、503 (网络错误)
  - 自动检测 Content-Type 并解码: `JSON(target)`、`XML(target)`、`Text()`
- 包级便捷函数 (`netx.Get()`、`netx.Post()` 等) 使用内部 `defaultClient`

**关键约定**: 调用方检查 `resp.Err` 而非 `err`。`Do()` 返回的 `error` 只表示请求构建失败，不表示网络执行失败。

依据：`common/netx/request.go`、`common/netx/response.go`。

### Transport

- `NewTransport(opts ...TransportOption)` — 创建带默认参数的 `http.Transport`（30s dial, 10s TLS, 90s idle, 100 max idle, HTTP/2 支持）。
- `NewHTTPClient(opts ...TransportOption)` — 使用上述 Transport 创建 `*http.Client`。
- `NewHTTPCService(name, opts ...TransportOption)` — 创建 go-zero `httpc.Service`。
- `NewHTTPEngine(svc httpc.Service)` — 包装 go-zero service 为 Engine。

依据：`common/netx/transport.go`。

### 编码工具

- `ValidateAndFlatten` — JSON → 扁平键值 map
- `EncodeURLEncoded` — JSON → URL-encoded 字符串
- `EncodeURLEncodedIfNeeded` — 智能判断：已编码则不处理
- `EncodeMultipart` — 键值 map → multipart/form-data

依据：`common/netx/encode.go`。

验证 Engine 切换、错误进入 `Response.Err`、超时状态码和编解码完整性。
