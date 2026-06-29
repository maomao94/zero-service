# 上下文传播规范

> `common/ctxprop/` 包负责跨边界 context 传播：gRPC metadata ↔ context、JWT claims ↔ context、MCP `_meta` ↔ context。字段定义在 `common/ctxdata/`。

## When to read

- 添加新的跨服务传播字段（修改 `ctxdata.PropFields` 后需要同步此包）
- 修改 gRPC 拦截器中 md 编码或解码逻辑
- 修改 JWT 拦截器中 claims 到 context 的映射
- 调试 MCP 请求中用户上下文丢失或 trace 断裂

## 包结构

```
common/ctxprop/
├── ctx.go    # MCP _meta ↔ context（CollectFromCtx, ExtractFromMeta）
├── claims.go # JWT claims ↔ context（ExtractFromClaims, ApplyClaimMapping, ClaimString）
└── grpc.go   # gRPC metadata ↔ context（InjectToGrpcMD, ExtractFromGrpcMD）
```

## gRPC 传播

字段通过 `ctxdata.PropFields` 驱动，在 gRPC 客户端和服务端拦截器中成对使用。

```go
// 客户端拦截器：发送前注入
func clientInterceptor(ctx context.Context, ...) (context.Context, error) {
    return ctxprop.InjectToGrpcMD(ctx), nil
}

// 服务端拦截器：接收后提取
func serverInterceptor(ctx context.Context, ...) (context.Context, error) {
    return ctxprop.ExtractFromGrpcMD(ctx), nil
}
```

### gRPC 非 ASCII 编码

非 ASCII 可打印字符（如中文字段值）自动 `base64` 编码并添加 `b64:` 前缀。`ExtractFromGrpcMD` 仅在检测到 `b64:` 前缀时才解码。

### 仅传播 string 类型

`InjectToGrpcMD` 只提取 `string` 类型的 context value。非 string 类型被静默跳过。

## JWT 传播

### 标准路径：claims key 一致

```go
ctx = ctxprop.ExtractFromClaims(ctx, claims) // claims["user-id"] → ctx.Value("user-id")
```

适用于 JWT claims key 与 `ctxdata.PropFields[*].CtxKey` 一致的情况。

### Claim 映射：外部 key 与内部 key 不同

```go
// 修改 claims 本身（外部 → 内部 key 覆盖写入）
ctxprop.ApplyClaimMapping(claims, map[string]string{
    "user-id": "user_id", // 外部 claim 的 key 是 user_id，写入内部 key user-id
})

// 或从上下文值复制（go-zero WithJwt 场景）
ctx = ctxprop.ApplyClaimMappingToCtx(ctx, map[string]string{
    "user-id": "user_id",
})
```

### ClaimString 类型处理

`ClaimString` 自动处理 JWT 常见类型：`string` 直接返回，`float64`（JSON number 解码）转为整数字符串。

## MCP _meta 传播

### 服务端：从 _meta 恢复上下文

```go
// MCP 服务端 handler 入口
ctx := ctxprop.ExtractFromMeta(ctx, meta) // _meta → context values
ctx = ctxprop.ExtractTraceFromMeta(ctx, meta) // traceparent → OTel context
```

### 客户端：发送上下文到 _meta

```go
meta := ctxprop.CollectFromCtx(ctx) // context values → map[string]any（nil 表示无可用字段）
ExtractTraceFromMeta ←→ trace.Extract (common/trace) 用于 W3C traceparent 传播
```

## 注意事项

### 添加新字段必须同步 ctxdata 和 ctxprop

`ctxdata.PropFields` 是唯一数据源。`ctxprop` 的所有函数都读取 `PropFields`，不需要为新字段增加 `ctxprop` 代码。但需确保 `ctxdata` 中新添加的 `PropField` 的 `GrpcHeader` 在 gRPC metadata 中不与已有字段冲突。

### gRPC header 必须小写

gRPC metadata key 强制小写。`ctxdata.PropField.GrpcHeader` 应使用 `kebab-case`（如 `"user-id"`），gRPC 自动转为小写。

### 参考文件

- `common/ctxprop/grpc.go` — gRPC 注入/提取实现
- `common/ctxprop/claims.go` — JWT claims 处理
- `common/ctxprop/ctx.go` — MCP _meta 处理
- `common/ctxdata/ctxData.go` — PropFields 定义
- `common/trace/carrier.go` — OTel trace 载体
- `app/djicloud/internal/svc/servicecontext.go:68` — gRPC 拦截器实际使用示例
