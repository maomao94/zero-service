# go-zero 服务约定

> 修改 RPC/API 服务结构、Handler/Server、Logic、`ServiceContext` 或新增服务方法时读取。

## 分层职责速查

| 层 | 职责 | 不做 | 依据 |
|----|------|------|------|
| Handler/Server | 接收传输参数、调用 Logic、返回结果 | 业务编排、数据库事务、外部协议转换 | `app/trigger/internal/server` |
| Logic | 请求级业务流程、校验、多依赖协调 | 复制 SQL/topic/帧操作 | `app/trigger/internal/logic` |
| ServiceContext | 创建共享 client/store/scheduler/producer | 保存请求级状态 | `app/trigger/internal/svc/servicecontext.go` |
| Model/store/SDK | 数据或外部系统边界 | — | — |

- 一个服务内沿用相邻 Logic 的构造与日志方式
- 不同服务的 `NewServiceContext` 可能返回单值或 `(*ServiceContext, error)`，不为表面一致性改动无关服务

## 开发顺序

1. 从 `.proto` 或 `.api` 确认输入、输出、校验和兼容性
2. 运行服务自己的 `gen.sh`，不从其他服务复制生成命令
3. 在 Logic 中实现业务，在 `ServiceContext` 中装配新增共享依赖
4. 把可复用且传输中立的能力留在现有 `common/` 包或窄接口后方
5. 运行目标服务测试/编译并检查生成 diff

## 依赖方向

```
传输层 → Logic → store/SDK/公共包
                ↓
         common/ (不反向依赖 internal/)
```

- 跨服务调用使用生成 client 或现有 facade，不直接导入另一个服务的 `internal/`
- 多个入口共享业务时提取服务私有组件或公共领域接口，避免 Handler 互相调用

## trace ID 获取

```go
// ✓ 正确
traceID := trace.TraceIDFromContext(ctx)

// ✗ 错误 - 过于冗长
traceID := oteltrace.SpanFromContext(ctx).SpanContext().TraceID().String()
```

- 未接入 OTEL 时返回空字符串，调用方无需做 nil span 判断
- 依据：`zerorpc/internal/task/deferforwardtask.go`、`common/crontask/crontask.go`

## 网关鉴权模式

> 修改 HTTP 网关（gtw）的路由鉴权、JWT 中间件或 token claim 转换时读取。

### 路由分类

| 路由类型 | 鉴权方式 | 中间件 | 示例 |
|---------|---------|--------|------|
| 业务 API | JWT + 自定义中间件 | `rest.WithJwt` + `rest.WithMiddlewares` | `/live/v1/meeting/*` |
| Webhook | 自有签名验签 | 无全局中间件 | `/webhook/livekit` |
| 测试页/静态资源 | 免认证 | 无 | `/test/meeting`、`/static/*` |

### 中间件链执行顺序

```
native middlewares → rest.WithJwt → server.Use → rest.WithMiddlewares → handler
```

`rest.WithMiddlewares` 直接包裹 `Route.Handler`，在 `rest.WithJwt` 之后执行，此时 JWT claims 已写入 context。

### 标准实现模式

```go
// 1. routes.go：rest.WithJwt 做验证，rest.WithMiddlewares 做 claim 桥接
server.AddRoutes(
    rest.WithMiddlewares(
        []rest.Middleware{middlewares.MeetingAuth},
        []rest.Route{...}...,
    ),
    rest.WithJwt(serverCtx.Config.JwtAuth.AccessSecret),
    rest.WithPrefix("/live/v1/meeting"),
)

// 2. middleware.go：在 JWT 验证之后运行，设置 auth-type + 桥接 claims
func NewMeetingAuthMiddleware(claimMapping map[string]string) *MeetingAuthMiddleware {
    return &MeetingAuthMiddleware{
        MeetingAuth: func(next http.HandlerFunc) http.HandlerFunc {
            return func(w http.ResponseWriter, r *http.Request) {
                ctx := r.Context()
                if auth := r.Header.Get("Authorization"); auth != "" {
                    ctx = authctx.WithAuthType(ctx, "user")
                    ctx = authctx.WithAuthorization(ctx, auth)
                }
                ctx = authctx.BridgeJWTClaims(ctx, claimMapping)
                next(w, r.WithContext(ctx))
            }
        },
    }
}

// 3. livegtw.go：路由注册时传入中间件实例
meetingAuth := handler.NewMeetingAuthMiddleware(c.JwtAuth.ClaimMapping)
handler.RegisterHandlers(server, ctx, meetingAuth)
```

### ClaimMapping 配置

Java 侧 JWT token 的 claim key 使用下划线（`user_id`），authctx 标准 key 使用短横线（`user-id`）：

```yaml
JwtAuth:
  AccessSecret: ""
  PrevAccessSecret: ""  # token 轮转，可选
  ClaimMapping:
    user-id: "user_id"
    user-name: "user_name"
    dept-code: "dept_code"
```

`BridgeJWTClaims(ctx, mapping)` 先拷贝短横线 wire 名（go-zero 默认），再按 mapping 拷贝下划线外部名。已存在的 typed value 不被覆盖（幂等）。

### 为什么不使用全局 server.Use

全局 `server.Use()` 对所有路由生效，webhook 路由的 Authorization 是签名 token 而非用户 JWT，会被错误标记为 `auth-type=user`。使用 `@server middleware` 注解 + `rest.WithMiddlewares` 实现路由组级别隔离。

依据：`aiapp/aigtw/aigtw.go`、`app/livegtw/livegtw.go`、`app/livegtw/internal/handler/middleware.go`、`common/authctx/context.go`。

## 反模式

| 错误做法 | 正确做法 | 原因 |
|---------|---------|------|
| 手写或长期修改生成的 Server/Handler/Routes/Types | 只改手写 Logic，生成文件不碰 | 生成文件会被覆盖 |
| 把输入校验、事务、重试和外部调用全部塞进 Server 方法 | 业务逻辑放在 Logic 层 | Server 只做传输适配 |
| 为单个 Logic 创建全局单例或把请求状态放进 ServiceContext | 请求状态在 Logic 内创建 | ServiceContext 是共享依赖 |
| 从另一个服务复制配置和生成脚本 | 核对本服务插件与输出目录 | 不同服务配置不同 |
| 使用 `logx.Logger.Warnf` | 用 `Errorf` 或 `Infof` | `logx.Logger` 没有 `Warnf` |
| 在全局 `server.Use()` 中设置 `auth-type=user` | 用 `rest.WithMiddlewares` 路由组隔离 | webhook 路由会被错误标记 |
| 把 JWT 验证放在自定义中间件内手动调用 `handler.Authorize` | 用 `rest.WithJwt()` 路由选项 | go-zero chain 统一管理 |
| `ClaimMapping` 配置缺失 | 配置完整的 claim 映射 | Java 侧 token 无法桥接 |

## 网关响应格式（go-zero-x BaseResponse）

> 修改 HTTP 网关 Handler 或新增 API 接口时读取。

### 标准响应格式

所有网关接口统一使用 `xhttp.JsonBaseResponseCtx` 返回，响应格式为 go-zero-x 的 `BaseResponse`：

```json
// 成功
{"code": 0, "msg": "ok", "data": {...}}

// 成功（无数据）
{"code": 0, "msg": "ok"}

// 错误
{"code": 5, "msg": "会议不存在"}
```

### Handler 模板

```go
import (
    "net/http"

    "github.com/zeromicro/go-zero/rest/httpx"
    xhttp "github.com/zeromicro/x/http"
)

func XxxHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req types.XxxRequest
        // 解析用 httpx.Parse
        if err := httpx.Parse(r, &req); err != nil {
            xhttp.JsonBaseResponseCtx(r.Context(), w, err)
            return
        }

        l := xxx.NewXxxLogic(r.Context(), svcCtx)
        resp, err := l.Xxx(&req)
        if err != nil {
            xhttp.JsonBaseResponseCtx(r.Context(), w, err)
        } else {
            xhttp.JsonBaseResponseCtx(r.Context(), w, resp)
        }
    }
}
```

### 关键规则

| 规则 | 说明 |
|------|------|
| 解析请求 | 用 `httpx.Parse(r, &req)` |
| 返回成功 | 用 `xhttp.JsonBaseResponseCtx(r.Context(), w, resp)` |
| 返回成功（无数据） | 用 `xhttp.JsonBaseResponseCtx(r.Context(), w, nil)` |
| 返回错误 | 用 `xhttp.JsonBaseResponseCtx(r.Context(), w, err)` |
| 不使用 | `httpx.OkJsonCtx`、`httpx.ErrorCtx`、`httpx.Ok` |

### 为什么不直接用 httpx

- `httpx.OkJsonCtx` 返回原始数据，不包装 `BaseResponse`
- `httpx.ErrorCtx` 依赖 `SetErrorHandlerCtx`，格式不统一
- `xhttp.JsonBaseResponseCtx` 自动处理成功和错误，统一格式

### 错误码来源

`xhttp.JsonBaseResponseCtx` 内部 `wrapBaseResponse` 处理逻辑：

| 输入类型 | Code | Msg |
|---------|------|-----|
| `*errors.CodeMsg` | 自定义 Code | 自定义 Msg |
| `*status.Status` | gRPC 状态码 | gRPC 消息 |
| `error` | -1 | err.Error() |
| 其他 | 0 | "ok" |

依据：`app/livegtw/internal/handler/meeting/`、`common/gtwx/errorhandler.go`。

## 网关 Logic 实现模式（livegtw）

> 新增 HTTP 网关 API 或实现 goctl 生成的空 Logic 时读取。

### 核心职责

网关 Logic 是传输适配层：接收 HTTP 请求参数，调用后端 gRPC 服务，转换 proto 响应为 HTTP 响应。

### 分类与签名

根据 `.api` 定义是否有返回类型，Logic 函数签名分为两类：

| 类型 | `.api` 定义 | Logic 签名 | Handler 使用 |
|------|------------|-----------|-------------|
| 有返回 | `post /xxx (Req) returns (Reply)` | `func (l *XxxLogic) Xxx(req *types.XxxRequest) (resp *types.XxxReply, err error)` | `resp, err := l.Xxx(&req)` |
| 无返回 | `post /xxx (Req)` | `func (l *XxxLogic) Xxx(req *types.XxxRequest) error` | `err := l.Xxx(&req)` |

### 实现模板

```go
// 有返回的 Logic
func (l *XxxLogic) Xxx(req *types.XxxRequest) (resp *types.XxxReply, err error) {
    r, err := l.svcCtx.LiveRpcCli.Xxx(l.ctx, &live.XxxReq{
        Field: req.Field, // 直接映射
    })
    if err != nil {
        return nil, err
    }
    return &types.XxxReply{
        Field: r.GetField(), // proto getter 安全取值
    }, nil
}

// 无返回的 Logic（void 操作）
func (l *XxxLogic) Xxx(req *types.XxxRequest) error {
    _, err := l.svcCtx.LiveRpcCli.Xxx(l.ctx, &live.XxxReq{
        Field: req.Field,
    })
    return err
}
```

### Proto 与 HTTP 类型转换

proto 和 HTTP 类型可能不一致，需要手动转换：

| Proto 类型 | HTTP 类型 | 转换方式 |
|-----------|----------|---------|
| `uint32` | `int32` | `uint32(req.ExpireSeconds)` |
| `[]byte` | `string` | `[]byte(req.Payload)` |
| `int64` | `int64` | 直接赋值 |
| `string` | `string` | 直接赋值 |

### 列表响应转换

proto 返回 `[]*live.XxxInfo`，HTTP 返回 `[]types.XxxInfo`，需要逐个转换：

```go
items := make([]types.XxxInfo, 0, len(r.GetItems()))
for _, item := range r.GetItems() {
    items = append(items, toXxxInfo(item)) // 使用 helper 函数
}
return &types.XxxReply{Items: items, Total: r.GetTotal()}, nil
```

### 常见错误

| 错误 | 原因 | 修复 |
|------|------|------|
| `cannot use int32 as uint32` | proto 和 HTTP 类型不匹配 | 手动类型转换 |
| `cannot use string as []byte` | proto 用 `[]byte`，HTTP 用 `string` | `[]byte(req.Payload)` |
| 未使用的 import | 从 authctx 获取身份但未使用 | 删除 import |
| 返回类型不匹配 | 有返回 vs 无返回签名搞混 | 检查 `.api` 定义 |

### 开发流程

1. 检查 `.api` 定义确认请求/响应类型
2. 检查 `.proto` 确认 gRPC 方法签名和字段类型
3. 实现 Logic：接收 HTTP 参数 → 调用 gRPC → 转换响应
4. 注意类型转换（uint32/int32, []byte/string）
5. 构建验证：`go build ./app/<service>/...`

### 网关增加字段标准流程

当需要给网关接口增加新字段时，必须遵循以下顺序：

```
1. 修改 .proto（gRPC 定义）
2. 执行服务自己的 gen.sh 重新生成 gRPC 代码
3. 检查生成的导入路径是否正确（goctl 可能生成错误路径）
4. 修改 .api（网关 API 定义）
5. 执行服务自己的 gen.sh 重新生成网关代码
6. 修改 logic 文件，补充字段映射
7. 编译验证 go build ./app/<service>/...
```

**禁止顺序**：
- ❌ 先写 logic 再改 api（会导致编译失败，types 包缺少字段）
- ❌ 只改 proto 不改 api（网关 types 与 gRPC 不一致）
- ❌ 只改 api 不改 proto（gRPC 层不识别新字段）
- ❌ 手动修改 pb.go 文件（应通过 gen.sh 重新生成）

**字段映射示例**：

```go
// .api 类型定义
type GenerateMeetingTicketRequest {
    MeetingNo         string   `json:"meetingNo"`
    CanPublishSources []string `json:"canPublishSources,optional"`
}

// logic 中映射到 gRPC
r, err := l.svcCtx.LiveRpcCli.GenerateMeetingTicket(l.ctx, &live.GenerateMeetingTicketReq{
    MeetingNo:         req.MeetingNo,
    CanPublishSources: req.CanPublishSources,
})
```

#### goctl 代码生成导入路径问题

**问题**：goctl 生成的代码可能包含错误的导入路径（如 `zero-service/app/live/app/live`），而正确路径应为 `zero-service/app/live/live`。

**原因**：goctl 的路径解析逻辑与项目的目录结构不匹配。

**解决方案**：在 gen.sh 运行后，检查并修复生成的导入路径：

```bash
# 检查是否有错误的导入路径
grep -rn "zero-service/app/<service>/app/<service>" app/<service>/

# 修复错误的导入路径
sed -i '' 's|zero-service/app/<service>/app/<service>|zero-service/app/<service>/<service>|g' app/<service>/<service>rpc/<service>rpc.go
```

**预防**：在 gen.sh 中添加后处理步骤，或使用 goctl 的 `--module` 参数。

## 验证

```bash
cd <service-directory>
./gen.sh
go test ./...
git diff --check
```

- 如果只改手写 Logic 且契约未变，不应制造生成文件 diff
- 至少运行目标服务或受影响包测试
