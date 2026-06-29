# netx HTTP 客户端规范

> `common/netx/` 包是基于 go-zero 生态的 HTTP 客户端，支持可插拔 Engine、`Request` 链式 Builder、流式上传下载、大小限制和 go-zero httpc 熔断/Otel 追踪集成。

## When to read

- 创建或修改外部 HTTP API 调用（REST/文件上传/下载）
- 使用 `Client` 默认引擎还是 go-zero `httpc.Service`（熔断/Otel 追踪）的选择
- 排查 `ErrResponseTooLarge` / `ErrUploadTooLarge` 溢出、`OptionError` 静默或 Engine 选择不当的性能问题
- 如涉及 MQTT 消息收发请改读 [`mqttx-guidelines.md`](./mqttx-guidelines.md)

## 包结构

```
common/netx/
├── client.go         # Client 结构体 + functional options 构造 + Do/Get/Post 等方法
├── client_pkg.go     # 包级别便捷函数 Get/Post/Put… 和 SendRequest
├── request.go        # Request 结构体 + RequestOption + 链式 Builder 方法
├── response.go       # Response 结构体 + JSON/XML/Text/Decode 解析 + 哨兵错误
├── upload.go         # Upload/UploadFile/UploadBytes + countingWriter 限流
├── download.go       # Download/DownloadFile/DownloadBytes + Range 支持 + 默认 32MB 限制
├── encode.go         # ValidateAndFlatten / EncodeURLEncoded / EncodeMultipart 编码工具
└── transport.go      # Engine 接口 + DefaultEngine / HTTPCEngine + NewTransport/NewHTTPCService
```

## 构造方式

```go
// 默认引擎（标准库 http.Client），10MB 响应上限、32MB 上传/下载上限
cli := netx.NewClient()

// 自定义限制
cli := netx.NewClient(
    netx.WithMaxResponseBytes(1 << 20),      // 1MB 响应上限
    netx.WithUploadBytesLimit(64 << 20),      // 64MB 上传上限
    netx.WithDownloadBytesLimit(0),            // 下载不限制
    netx.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}),
    netx.WithDefaultHeaders(http.Header{"Authorization": {"Bearer token"}}),
    netx.WithHTTPClientOption(func(hc *http.Client) {
        hc.CheckRedirect = func(req *http.Request, via []*http.Request) error {
            return http.ErrUseLastResponse
        }
    }),
)
```

`WithHTTPClientOption` 仅在**未自定义 Engine** 时生效。Engine 自定义后由调用方接管 `http.Client` 的全部配置。

### go-zero httpc 引擎（推荐生产环境）

```go
// servicecontext.go
httpcSvc := netx.NewHTTPCService("my-svc")   // 复用 NewTransport 连接池配置
cli := netx.NewClient(
    netx.WithEngine(netx.NewHTTPEngine(httpcSvc)),
)
```

`HTTPCEngine` 通过 `httpc.Service` 获得 go-zero 内置的 Otel 追踪、熔断、指标收集。无特殊原因（如需要裸 `http.Client` 的 cookie jar 或 redirect 控制）优先使用此模式。

## Engine 接口

```go
type Engine interface {
    Do(req *http.Request) (*http.Response, error)
}
```

- `DefaultEngine` —— 标准 `http.Client` 封装，Otel 追踪需自行通过 Transport 包装。
- `HTTPCEngine` —— `httpc.Service` 封装，自带 Otel 追踪/熔断/日志。
- 自定义 —— 实现 `Do` 方法即可注入 mock 或第三方客户端。

参考文件：`common/netx/transport.go`

## 请求构建

### RequestOption 函数式

```go
req := netx.NewRequest(url, http.MethodPost,
    netx.WithJSONBody(payload),                        // 自动 JSON 序列化 + Content-Type
    netx.WithHeader("Authorization", "Bearer token"),
    netx.WithQueryParams(url.Values{"page": {"1"}}),
)
```

### Request 链式 Builder

```go
req := netx.NewRequest(url, http.MethodPost).
    JSON(map[string]string{"name": "zero"}).           // 同 WithJSONBody
    Header("X-Trace-Id", traceId).
    Query("page", "1")

resp, err := cli.Do(ctx, req)
```

Builder 方法 `.Header()` / `.Query()` / `.JSON()` / `.Form()` / `.Raw()` / `.Reader()` 返回 `*Request` 自身，支持无限链式调用。内部委托对应的 `RequestOption`。

参考文件：`common/netx/request.go`

### Body 来源优先级

`buildBody` 选择顺序（`client.go`）：
1. `FormData` / `bodyKindForm` → `application/x-www-form-urlencoded`
2. `BodyReader` / `bodyKindReader` → 流式，调用方负责关闭
3. `bodyKindJSON` → 自动 `application/json`
4. 裸 `Body` + `Content-Type` 探测 → 若 Content-Type 含 `x-www-form-urlencoded` 则走 `EncodeURLEncodedIfNeeded`

## 响应解析

```go
resp, err := cli.Get(ctx, url)
if err != nil {
    return err                                    // 构造级错误
}
if !resp.Success {
    return fmt.Errorf("status %d: %s", resp.StatusCode, resp.Err)  // 业务错误
}

var result MyResponse
if err := resp.JSON(&result); err != nil { ... }   // 或 resp.XML / resp.Text / resp.Decode
```

- `resp.Err` 携带网络错误或 `ErrResponseTooLarge`
- `resp.CostMs` / `resp.CostFormatted` 记录完整请求耗时
- `classifyNetErr` 将网络错误映射为 HTTP 语义状态码（`response.go`）
- 网络错误不返回 `error` 而是放到 `resp.Err`——调用方需检查 `resp.Success`

## 上传

```go
// 流式上传（内存友好）
resp, err := cli.Upload(ctx, url, []netx.FileUpload{
    {FieldName: "file", FileName: "photo.jpg", Content: fileReader},
    {FieldName: "thumb", FileName: "thumb.jpg", Content: thumbReader},
}, map[string]string{"desc": "avatar"})

// 便捷方法
resp, err := cli.UploadFile(ctx, url, "/tmp/photo.jpg", "file", nil)
resp, err := cli.UploadBytes(ctx, url, "attachment", "data.bin", data, nil)

// 支持 RequestOption（如鉴权头）
resp, err := cli.Upload(ctx, url, files, fields,
    netx.WithHeader("Authorization", "Bearer upload-token"),
)
```

- 使用 `io.Pipe` + goroutine 流式写入，**不缓冲全量**到内存
- `uploadBytesLimit` 通过 `countingWriter` 在流写入时拦截（`upload.go`）
- stream goroutine panic 会被 recover 并传播到 resp.Err（`upload.go`）

## 下载

```go
// 流式下载 reader
body, err := cli.Download(ctx, url)
defer body.Close()
io.Copy(writer, body)                              // 受 client downloadBytesLimit 约束

// 保存文件（原子写入：tmp + rename）
err := cli.DownloadFile(ctx, url, "/tmp/output.bin",
    netx.WithDownloadMaxBytes(100<<20),             // 覆盖 client 级限制
)

// 全量 bytes
data, err := cli.DownloadBytes(ctx, url)

// 断点续传
data, err := cli.DownloadBytes(ctx, url,
    netx.WithDownloadRange(0, 1023),
)
```

- `DownloadFile` 先写 `.tmp`，成功后 `os.Rename`，失败自动清理（`download.go`）
- 调用方**必须 `Close`** `Download` 返回的 `io.ReadCloser`，否则连接泄漏
- `WithDownloadRange(start, end)` 设置 `Range` 请求头（`download.go`）

## 大小限制

| 常量 | 默认值 | 适用方法 | 设为 0 |
| --- | --- | --- | --- |
| `DefaultMaxResponseBytes` | 10MB | `Client.Do` | 不限制 |
| `DefaultDownloadBytesLimit` | 32MB | `Download` / `DownloadBytes` / `DownloadFile` | 不限制 |
| `DefaultUploadBytesLimit` | 32MB | `Upload` / `UploadFile` / `UploadBytes` | 不限制 |

限制触发时返回哨兵错误 `ErrResponseTooLarge` / `ErrUploadTooLarge`，可配合 `errors.Is` 判断。

## 包级别便捷函数

```go
resp, err := netx.Get(ctx, url)
resp, err := netx.Post(ctx, url, netx.WithJSONBody(payload))
data, err := netx.DownloadBytes(ctx, url)

// 临时自定义 Client 配置（注意：每次新建 Client 和 Transport，不共享连接池）
resp, err := netx.SendRequest(ctx, req, netx.WithEngine(customEngine))
```

`SendRequest` 不带 `opts` 时复用全局 `defaultClient` 的连接池；带 `opts` 时每次新建 `Client`，高频调用应自行持有 `Client` 实例。参考：`client_pkg.go`

## 常见反模式

### 忽略 `resp.Err` 只判 `err`

```go
resp, err := cli.Get(ctx, url)
if err != nil { return err }                    // ❌ 网络错误在 resp.Err 里
if !resp.Success { ... }                        // ✅ 统一检查
```

`Client.Do` 在网络错误时不返回 error，而是填充到 `resp.Err`。`resp.Err != nil` 时 `resp.Success == false`，`resp.StatusCode` 为 `classifyNetErr` 映射值。

### 高频调用用 `SendRequest` 带 opts

```go
for i := 0; i < 1000; i++ {                     // ❌ 每次新建 Transport/连接池
    resp, err := netx.SendRequest(ctx, req, netx.WithMaxResponseBytes(1<<20))
}
cli := netx.NewClient(netx.WithMaxResponseBytes(1<<20))  // ✅ 复用连接池
```

### 流式 reader 不关闭

```go
body, _ := cli.Download(ctx, url)
// ... 忘记 body.Close()                        // ❌ 连接泄漏
defer body.Close()                              // ✅
```

### 下载文件后忘记处理 `.tmp` 残留

`DownloadFile` 在 `io.Copy` 或 `Close` 失败时会自动移除 `.tmp`，但在 `os.Rename` 失败且 `Remove` 也失败时可能残留。调用方可周期性清理 `*.tmp`。

### custom Engine 下用 `WithHTTPClientOption`

`WithHTTPClientOption` 仅在 `buildEngine` 内生效，自定义 Engine 时被跳过。自定义 Engine 的 `http.Client` 配置需在 Engine 构造时完成。

## 参考文件

- `common/netx/client.go` — 默认常量定义、Client 结构体、ClientOption 列表、Do 方法完整流程
- `common/netx/request.go` — Request 结构体、链式 Builder 方法
- `common/netx/response.go` — Response 结构体
- `common/netx/upload.go` — 流式上传实现
- `common/netx/download.go` — 原子下载文件
- `common/netx/transport.go` — Engine 接口、httpc 集成
- `app/trigger/internal/svc/servicecontext.go` — 生产环境 httpc Engine 使用示例
- `app/file/internal/svc/servicecontext.go` — 文件服务 httpc Engine 使用示例
