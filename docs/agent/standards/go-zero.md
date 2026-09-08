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

