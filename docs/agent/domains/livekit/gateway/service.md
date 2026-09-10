# LiveKit HTTP 网关

适用：HTTP 网关 `app/livegtw`，转发请求到 `app/live` gRPC 服务。

### 1. 开发流程

```
1. 编写 livegtw.api 定义接口（类型定义 + 路由 + 鉴权组）
2. 执行 gen.sh 生成 handler / logic / types / routes
3. 在 logic 文件中实现业务逻辑（调用 gRPC client）
4. 特性钩子类（webhook、测试页）不走 api 定义，直接在 livegtw.go 配置路由
```

### 2. 命名规范（api 层 vs gRPC 层）

| 层 | 请求 | 响应 | 说明 |
|----|------|------|------|
| `livegtw.api` | `XxxRequest` | `XxxReply` | HTTP 网关类型，goctl 生成到 `types` 包 |
| `live.proto` | `XxxReq` | `XxxRes` | gRPC 类型，protoc 生成到 `live` 包 |

- 两层类型名**不同**（`Request/Reply` vs `Req/Res`），避免同包同名冲突
- logic 中做映射：`types.XxxRequest` → `live.XxxReq`，`live.XxxRes` → `types.XxxReply`

### 3. 文件结构与职责

```
app/livegtw/
├── livegtw.api              # API 定义文件
├── gen.sh                   # 代码生成脚本
├── livegtw.go               # 主入口
├── internal/
│   ├── config/config.go     # 配置结构
│   ├── svc/servicecontext.go # ServiceContext（持有 gRPC client、中间件）
│   ├── handler/
│   │   ├── routes.go        # [生成] 路由注册
│   │   ├── meeting/         # [生成] meeting 组 handler（httpx 默认格式）
│   │   ├── ticket/          # [生成] 免鉴权组 handler
│   │   ├── webhook/         # [手写] webhook handler（不走 api 定义）
│   │   └── testpage/        # [手写] 测试页 handler
│   ├── logic/
│   │   ├── meeting/         # [生成+手写] meeting 组 logic（含 meeting_helper.go）
│   │   ├── ticket/          # [生成+手写] 免鉴权组 logic
│   │   └── webhook/         # [手写] webhook logic
│   ├── middleware/           # [生成+手写] 中间件
│   │   └── meetingauthmiddleware.go
│   └── types/               # [生成] 请求/响应类型（XxxRequest/XxxReply）
```

### 4. API 定义规范（livegtw.api）

```go
// 类型定义：请求用 XxxRequest，响应用 XxxReply
type CreateMeetingRequest {
    Title string `json:"title"`
}

type CreateMeetingReply {
    Meeting MeetingInfo `json:"meeting"`
}

// 业务接口组：需要 JWT + 中间件
@server (
    prefix:     live/v1
    group:      live
    jwt:        JwtAuth
    middleware: MeetingAuth
)
service livegtw {
    @doc "创建会议"
    @handler createMeeting
    post /createMeeting (CreateMeetingRequest) returns (CreateMeetingReply)
}

// 免鉴权组：不声明 jwt/middleware（如票据加入会议）
@server (
    prefix: live/v1
    group:  ticket
)
service livegtw {
    @doc "根据票据加入会议（无需JWT）"
    @handler joinMeetingByTicket
    get /joinMeetingByTicket (JoinMeetingByTicketRequest) returns (JoinMeetingByTicketReply)
}
```

- 业务接口必须声明 `jwt: JwtAuth` 和 `middleware: MeetingAuth`
- 免鉴权接口单列一个 `@server` 块，不写 `jwt`/`middleware`
- 查询类用 `get`，写入类用 `post`；get 的请求参数 tag 用 `form`, post 用 `json`
- webhook、测试页等特性钩子不走 api 定义，直接在 `livegtw.go` 配置路由
- **路由命名与 gRPC 接口保持一致**：handler 名和路径都使用小驼峰，与 gRPC 方法名对应（如 `CreateMeeting` → `createMeeting` → `/createMeeting`）
- **例外：组合业务路由**：如果网关接口是多个 gRPC 调用组合的业务逻辑（非直接转发），则按前端业务语义命名，不必与 gRPC 一致
- **例外：同一 gRPC 多个前端路由**：如"全部会议列表"和"我的会议列表"都调用 `ListMeetings`，但前端路由应分别命名为 `/listMeetings` 和 `/myMeetings`（后者在 logic 中自动注入 identity 参数）
- **网关自有接口**：如 `/getCurrentUser`（从 authctx 获取当前用户信息），不调用 gRPC，直接在网关 logic 实现

### 5. 中间件规范

```go
// internal/middleware/meetingauthmiddleware.go
type MeetingAuthMiddleware struct {
    claimMapping map[string]string
}

func NewMeetingAuthMiddleware(claimMapping map[string]string) *MeetingAuthMiddleware {
    return &MeetingAuthMiddleware{claimMapping: claimMapping}
}

func (m *MeetingAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        if auth := r.Header.Get("Authorization"); auth != "" {
            ctx = authctx.WithAuthorization(ctx, auth)
        }
        ctx = authctx.BridgeJWTClaims(ctx, m.claimMapping)
        next(w, r.WithContext(ctx))
    }
}
```

- 中间件不再硬编码 `auth-type`，由 `BridgeJWTClaims` 从 JWT claims 中提取（设备 token 为 `device`，用户 token 为 `user`）
- `routes.go` 通过 `serverCtx.MeetingAuth` 引用
- **用户身份**通过 `authctx.GetUserId(ctx)` / `authctx.GetUserName(ctx)` 获取，写入 gRPC metadata 透传

### 6. Logic 实现规范

```go
// internal/logic/meeting/createmeetinglogic.go
type CreateMeetingLogic struct {
    logx.Logger
    ctx    context.Context
    svcCtx *svc.ServiceContext
}

func NewCreateMeetingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateMeetingLogic {
    return &CreateMeetingLogic{
        Logger: logx.WithContext(ctx),
        ctx:    ctx,
        svcCtx: svcCtx,
    }
}

func (l *CreateMeetingLogic) CreateMeeting(req *types.CreateMeetingRequest) (resp *types.CreateMeetingReply, err error) {
    r, err := l.svcCtx.LiveRpcCli.CreateMeeting(l.ctx, &live.CreateMeetingReq{Title: req.Title})
    if err != nil {
        return nil, err
    }
    return &types.CreateMeetingReply{Meeting: toMeetingInfo(r.GetMeeting())}, nil
}
```

- 每个接口一个单独的 logic 文件（goctl 标准拆分模式）
- 公共转换函数（`toMeetingInfo`/`toParticipantInfo`）放在 `meeting_helper.go`，可以被同组 logic 复用
- **logic 只做请求转发和类型转换**，不写业务编排/单测
- 依赖登录用户的接口（join、listMyMeetings）用 `authctx.GetUserId(l.ctx)` 取身份，不使用请求体传

#### Logic 函数签名分类

根据 `.api` 定义是否有返回类型，Logic 函数签名分为两类：

| 类型 | `.api` 定义 | Logic 签名 | Handler 使用 |
|------|------------|-----------|-------------|
| 有返回 | `post /xxx (Req) returns (Reply)` | `func (l *XxxLogic) Xxx(req *types.XxxRequest) (resp *types.XxxReply, err error)` | `resp, err := l.Xxx(&req)` |
| 无返回 | `post /xxx (Req)` | `func (l *XxxLogic) Xxx(req *types.XxxRequest) error` | `err := l.Xxx(&req)` |

```go
// 有返回的 Logic
func (l *EndMeetingLogic) EndMeeting(req *types.EndMeetingRequest) (resp *types.EndMeetingReply, err error) {
    r, err := l.svcCtx.LiveRpcCli.EndMeeting(l.ctx, &live.EndMeetingReq{MeetingNo: req.MeetingNo})
    if err != nil {
        return nil, err
    }
    return &types.EndMeetingReply{}, nil
}

// 无返回的 Logic（void 操作）
func (l *EndMeetingLogic) EndMeeting(req *types.EndMeetingRequest) error {
    _, err := l.svcCtx.LiveRpcCli.EndMeeting(l.ctx, &live.EndMeetingReq{MeetingNo: req.MeetingNo})
    return err
}
```

#### Proto 与 HTTP 类型转换

proto 和 HTTP 类型可能不一致，需要手动转换：

| Proto 类型 | HTTP 类型 | 转换方式 |
|-----------|----------|---------|
| `uint32` | `int32` | `uint32(req.ExpireSeconds)` |
| `[]byte` | `string` | `[]byte(req.Payload)` |
| `int64` | `int64` | 直接赋值 |
| `string` | `string` | 直接赋值 |

```go
// 示例：uint32 vs int32
r, err := l.svcCtx.LiveRpcCli.GenerateMeetingTicket(l.ctx, &live.GenerateMeetingTicketReq{
    ExpireSeconds: uint32(req.ExpireSeconds), // proto 用 uint32，HTTP 用 int32
})

// 示例：[]byte vs string
r, err := l.svcCtx.LiveRpcCli.SendMeetingData(l.ctx, &live.SendMeetingDataReq{
    Payload: []byte(req.Payload), // proto 用 []byte，HTTP 用 string
})
```

#### 列表响应转换

proto 返回 `[]*live.XxxInfo`，HTTP 返回 `[]types.XxxInfo`，需要逐个转换：

```go
items := make([]types.XxxInfo, 0, len(r.GetItems()))
for _, item := range r.GetItems() {
    items = append(items, toXxxInfo(item)) // 使用 helper 函数
}
return &types.XxxReply{Items: items, Total: r.GetTotal()}, nil
```

### 7. Handler 规范

handler 使用 `xhttp.JsonBaseResponseCtx`（统一响应格式 `{code, msg, data}`，gRPC 错误自动转换）：

```go
import xhttp "github.com/zeromicro/x/http"

var req types.CreateMeetingRequest
if err := httpx.Parse(r, &req); err != nil {
    xhttp.JsonBaseResponseCtx(r.Context(), w, err)
    return
}
l := meeting.NewCreateMeetingLogic(r.Context(), svcCtx)
resp, err := l.CreateMeeting(&req)
if err != nil {
    xhttp.JsonBaseResponseCtx(r.Context(), w, err)
} else {
    xhttp.JsonBaseResponseCtx(r.Context(), w, resp)
}
```

- `xhttp.JsonBaseResponseCtx` 内部调用 `wrapBaseResponse`，自动处理 gRPC status error（提取 code + message）和普通 error（code=-1）
- 成功响应：`code=0, msg="ok", data=<resp>`
- 错误响应：HTTP 200 + `{code: <grpc-code或-1>, msg: "<错误信息>"}`
- **标准网关写法**：统一使用 `xhttp.JsonBaseResponseCtx`，除非用户特殊要求返回其他格式

### 8. ServiceContext 规范

```go
type ServiceContext struct {
    Config     config.Config
    LiveRpcCli live.LiveRpcClient    // gRPC 客户端
    MeetingAuth rest.Middleware       // 中间件
}

func NewServiceContext(c config.Config) *ServiceContext {
    logx.Must(logx.SetUp(c.Log))
    m := middleware.NewMeetingAuthMiddleware(c.JwtAuth.ClaimMapping)
    return &ServiceContext{
        Config: c,
        LiveRpcCli: live.NewLiveRpcClient(zrpc.MustNewClient(c.LiveRpcConf,
            zrpc.WithUnaryClientInterceptor(grpcx.UnaryMetadataInterceptor)).Conn()),
        MeetingAuth: m.Handle,
    }
}
```

### 9. 主入口（livegtw.go）规范

```go
func main() {
    // ... 配置加载 ...
    server := rest.MustNewServer(c.RestConf, gtwx.CorsOption())

    // 请求日志中间件（method/path/start time 写入 context）
    server.Use(gtwx.RequestLogMiddleware)

    // 响应日志（仅记录业务错误，成功由 go-zero 标准日志覆盖）
    gtwx.SetLogOkHandler()

    ctx := svc.NewServiceContext(c)

    // 业务 API 路由（通过 routes.go 注册）
    handler.RegisterHandlers(server, ctx)

    // 特性钩子路由（直接配置，不走 api 定义）
    server.AddRoute(rest.Route{
        Method:  http.MethodPost,
        Path:    "/webhook/livekit",
        Handler: webhook.LiveKitWebhookHandler(ctx),
    })

    // 测试页（可选）
    if c.EnableTestPage {
        server.AddRoute(rest.Route{
            Method:  http.MethodGet,
            Path:    "/test/meeting",
            Handler: testpage.MeetingTestPageHandler(),
        })
    }
}
```

#### 网关日志模式

- **不要调用 `gtwx.SetGrpcErrorHandler()`**（deprecated）：handlers 统一走 `xhttp.JsonBaseResponseCtx`，gRPC 错误已由 `wrapBaseResponse` 内置转换为 `{code, msg}` 响应体。
- **`RequestLogMiddleware`**：把 `method`、`path`、`start time` 写入 context，供 ok handler 读取。
- **`SetLogOkHandler`**：只在业务错误（`code != 0`）时打印 error 日志（含 method、path、duration、code、msg），成功请求由 go-zero 标准 `LogHandler` 覆盖，不重复打 info。
- 日志效果：
  ```
  [HTTP] POST /live/v1/live/createMeeting  duration=12ms  code=102102  msg="meeting not found"
  ```

### 10. gRPC 层约定（app/live）

- **按表字段简单检索**，RPC 层不做复杂业务编排。例如会议列表查询就是按 `status`/`create_user`/`title`/`identity` 等字段条件过滤，逻辑放在 repo 的 where 子句。
- 新增的检索条件（如"查某用户相关的会议"）通过给 `ListMeetingsReq` 加一个 `identity` 字段实现，**不要单开一个 `ListMyMeetings` RPC**。
- 网关 /myList 就是调用同一个 `ListMeetings`，传当前用户 identity。**避免为同一查询开多个 RPC。**

### 11. 代码生成注意事项

- `gen.sh` 会用 goctl 生成 scaffold（skeleton），需要手动填 logic 业务逻辑
- **从 `.api` 生成时只保留最新结构**：如果之前手工架过 logic，重新生成前先 `rm -rf internal/handler internal/logic internal/types`，避免新旧命名文件（驼峰 vs 下划线）共存导致重复定义
- **不要**把多个 logic 合并进一个 `meetinglogic.go`（合并文件模式和 goctl 的拆分模式冲突，会让后续 `gen.sh` 生成重复定义）。旧合并文件应删掉，改成拆分文件。
- 生成文件不要手工改结构（struct 定义/函数签名），只填 logic 方法体
- **必须用 `gen.sh` 生成**：`gen.sh` 内部调用 `goctl api format` + `goctl api go`。不能直接 `goctl api go`（会跳过 format 导致 api 文件格式异常）。proto 用 `goctl rpc protoc` + `gen.sh`
- **`liverpc` 包已废弃**：直接用 proto 生成的 `live.LiveRpcClient`（在 `app/live/live/live_grpc.pb.go`）。ServiceContext 字段名是 `LiveRpcCli live.LiveRpcClient`
- **`--client=false`**：`app/live/gen.sh` 使用此参数，`liverpc.go` 的 client 接口不会自动更新。新增 RPC 后需要手动更新 `liverpc.go`（如果还用的话）或直接用 `live.LiveRpcClient`
- **webhook 测试**：`fakeLiveRpcCli`（webhook 测试用）必须实现 `live.LiveRpcClient` 的**全部**方法。gRPC 接口一旦新增/删除方法，`helpers_test.go` 里的 fake 需同步补齐/删除对应方法，否则 `go test ./...` 构建失败。

