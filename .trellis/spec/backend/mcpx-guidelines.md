# MCPx 协议规范

> `common/mcpx` 是 MCP (Model Context Protocol) 客户端和服务端封装。提供多服务连接管理、工具/提示模板/资源路由、鉴权、进度通知和异步任务执行。

## When to read

- 新增 MCP Server 注册，或修改 Server Config 结构。
- 在 `aiapp/mcpserver` 或 `aiapp/aigtw` 中集成 MCP 工具调用。
- 对接新的 MCP 后端，配置鉴权和传输协议。
- 实现带进度通知的长时间工具调用。

## 包结构

| 文件 | 职责 |
|------|------|
| `client.go` | MCP Client：多连接管理、工具/提示/资源路由、同步/异步/流式调用、进度通知 | 
| `server.go` | MCP Server：REST 路由注册、SSE/Streamable transport、JWT + ServiceToken 鉴权 |
| `config.go` | Client/Server 配置结构 | 
| `auth.go` | 双模式 Token 验证器（ServiceToken 常量比较 + JWT 解析） |
| `wrapper.go` | 工具调用包装器（CallToolWrapper）：trace 传播、_meta 透传、进度发送 |
| `async_result.go` | 异步结果存储接口和任务观察者 | 
| `memory_handler.go` | 内存异步结果存储实现 | 
| `logger.go` | MCP SDK 日志适配 | 

## Client

### 构造

`mcpx/client.go:139-200`：

```go
cfg := mcpx.Config{
    Servers: []mcpx.ServerConfig{
        {Name: "search", Endpoint: "http://localhost:8090/sse", UseStreamable: true},
        {Name: "tools",  Endpoint: "http://localhost:8091/messages", ServiceToken: "xxx"},
    },
}
client := mcpx.NewClient(cfg,
    mcpx.WithOptions(&mcp.ClientOptions{
        Capabilities:     &mcp.ClientCapabilities{...},
        CreateMessageHandler: samplingHandler,
    }),
    mcpx.WithAsyncResultStore(mcpx.NewMemoryAsyncResultStore()),
)
defer client.Close()
```

- 服务名冲突检测：重复 name 跳过并打 error 日志
- 服务名未设时自动生成 `mcp0, mcp1...`
- 每连接独立 context，`RefreshInterval` 控制重连间隔（默认 30s）

### 工具路由

`client.go:482-504` — 工具名加 server 前缀 `serverName__toolName`，重名安全：

```go
client.CallTool(ctx, "search__web_search", map[string]any{"query": "..."})
```

### 三种调用模式

| 模式 | 方法 | 返回值 | source |
|------|------|--------|--------|
| 同步 | `CallTool(ctx, name, args)` | `(string, error)` | `client.go:347` |
| 流式（带进度） | `CallToolWithProgress(ctx, req)` | `(string, error)` | `client.go:370` |
| 异步（fire-and-forget） | `CallToolAsync(ctx, req)` | `(taskID, error)` | `client.go:962` |
| 异步（可等待） | `CallToolAsyncAwait(ctx, req)` | `(taskID, Promise, error)` | `client.go:1072` |

`CallToolAsync` 通过 `context.WithoutCancel(ctx)` 创建独立 context，不继承原 ctx 的 cancel。`CallToolAsyncAwait` 使用 `antsx.Reactor` goroutine 池复用。

### 进度通知

`client.go:290-302` — 通过 `antsx.EventEmitter[ProgressInfo]` 按 progressToken 分发进度事件。`CallToolWithProgress` 内部订阅进度并回调 `OnProgress`。

### Context 注入

每次工具调用时，从 ctx 收集上下文并注入 `_meta`：
```go
meta := ctxprop.CollectFromCtx(ctx)
trace.Inject(ctx, trace.NewAnyCarrier(meta))
params.SetMeta(meta)
```

## Server

### 构造

`server.go:42-79`：

```go
srv := mcpx.NewMcpServer(mcpx.McpServerConf{
    RestConf: rest.RestConf{Host: "0.0.0.0", Port: 8090},
    McpConf: mcp.McpConf{
        Name:           "my-tools",
        Version:        "1.0.0",
        SseEndpoint:    "/sse",
        MessageEndpoint: "/messages",
        UseStreamable:  true,
    },
    Auth: struct {
        JwtSecrets   []string
        ServiceToken string
        ClaimMapping map[string]string
    }{
        JwtSecrets: []string{"my-secret"},
    },
})

mcp.AddTool(srv.Server(), myTool, mcpx.CallToolWrapper(handler))
srv.Start()
```

### Transport

`server.go:102-119`：
- **SSE** (2024-11-05)：`sdkmcp.NewSSEHandler`，注册 GET 到 `SseEndpoint`
- **Streamable HTTP** (2025-03-26)：`sdkmcp.NewStreamableHTTPHandler`，注册 GET/POST/DELETE 到 `MessageEndpoint`

### 鉴权

`auth.go:22-71` — `NewDualTokenVerifier` 双模式验证：
1. ServiceToken 常量时间比较 → `CtxAuthType: "service"`
2. JWT 解析 → `CtxAuthType: "user"`，从 claims 提取 `user_id` 等字段

验证器包装为 `auth.RequireBearerToken` 中间件（`server.go:122-128`）。

### CallToolWrapper

`wrapper.go:208-285` — 泛型工具调用包装器，处理：
1. `_meta` → trace context 提取（`ctxprop.ExtractTraceFromMeta`）
2. `_meta` → ctx values（`ctxdata.CtxMetaKey`）
3. 可选用户上下文提取（`WithExtractUserCtx`）→ 供 gRPC 调用透传
4. `progressToken` → `ProgressSender` 注入 ctx（`GetProgressSender(ctx)`）
5. 调用结束后自动 `Resolve`/`Reject` + `Stop` 进度发送器

## 异步结果存储

`async_result.go` — `AsyncResultStore` 接口，二开可替换持久化实现：

```go
type AsyncResultStore interface {
    Save(ctx, *AsyncToolResult) error
    Get(ctx, taskID) (*AsyncToolResult, error)
    UpdateProgress(ctx, taskID, progress, total, message) error
    Exists(ctx, taskID) bool
    List(ctx, *ListAsyncResultsReq) (*ListAsyncResultsResp, error)
    Stats(ctx) (*AsyncResultStats, error)
}
```

内存实现在 `memory_handler.go`（单机开发/测试），生产建议用 Redis/MySQL 实现。

## 信任边界

| 层 | 职责 |
|----|------|
| MCP Server ServiceToken | 验证调用方是可信 Client |
| JWT Token | 验证用户身份，提取 user_id |
| 业务层 | 从 `_meta` 或 `ctxdata` 解析用户身份，自行鉴权 |

MCP 层不做用户权限校验，只做 trace 传播和 _meta 透传。

## 反模式

- 不要在 CallToolWrapper 内做用户鉴权——应在业务 handler 内自行处理。
- 不要在生产用内存 `AsyncResultStore`——任务重启即丢失。
- 不要在 MCP Server 端跳过 `CallToolWrapper` 直接注册 handler——会丢失 trace 传播和进度功能。
- 不要混用 SSE 和 Streamable HTTP 在同一 Server——用 `UseStreamable` 统一控制。
