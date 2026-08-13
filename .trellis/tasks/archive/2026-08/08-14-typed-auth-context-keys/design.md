# Typed Context Key 迁移设计

## 边界

本任务仅迁移**进程内**认证身份的存储方式，从公开 string context key 迁移到 authctx 包私有的 typed key + 统一 setter/getter。不改变：
- wire key（JWT claim、gRPC metadata、MCP `_meta`）与字段顺序；
- claim 类型接受规则（`ClaimString` 行为由 `normalize-auth-claims` 处理）；
- raw Authorization 传播、`b64:`、metadata 重复/冲突行为；
- 方法策略（`enforce-authorization-policy` 处理）。

## 目标 API（authctx 包）

包私有 typed key（每个字段一个 struct{} 类型）：

```go
type userIDKey struct{}
type userNameKey struct{}
type deptCodeKey struct{}
type authorizationKey struct{}
type authTypeKey struct{}
```

统一 setter / getter（getter 保持 string 断言 + 空串 fallback 语义）：

```go
func WithUserID(ctx context.Context, v string) context.Context
func WithUserName(ctx context.Context, v string) context.Context
func WithDeptCode(ctx context.Context, v string) context.Context
func WithAuthorization(ctx context.Context, v string) context.Context
func WithAuthType(ctx context.Context, v string) context.Context

func GetUserID(ctx context.Context) string
func GetUserName(ctx context.Context) string
func GetDeptCode(ctx context.Context) string
func GetAuthorization(ctx context.Context) string
func GetAuthType(ctx context.Context) string
```

保留 5 个公开 string 常量与 `ContextKeys` 顺序不变（wire 契约，由 grpcx/mcpx/签发方使用，且被契约测试锁定）。这两个对象**仅**作为 wire/claim 命名表，不再直接用于进程 context 读写。

## 迁移策略

### 阶段 0：无兼容回退，采用显式桥接中间件

go-zero v1.10.3 `rest/handler/authhandler.go:72-78` 在 JWT 中间件中把非标准 claim 用**原始 claim 名 string key** 写入进程 context（out-of-repo 写入方）。本仓库任何地方都没有为 `user-id`/`user-name`/`dept-code` 调用 typed setter——网关身份全部来自该中间件。

设计决策（用户确认）：**getter 不做 string-key 回退，只读 typed key**。go-zero 的 string-key 写入属于它自己的命名空间，我们通过网关层**显式桥接中间件**（在 JWT 之后执行）把 claim 额外写入 typed key 命名空间。两个命名空间互不污染，桥接是唯一翻译点，外部无法绕过 authctx 读写。

go-zero 中间件执行顺序已确认（`rest/engine.go` `bindRoute`：先 `appendAuthHandler` 追加 JWT，再 append `ng.middlewares` 全局 `server.Use`）：**全局 `server.Use` 中间件在 JWT 之后运行**，能看到 JWT 写入的 string claims context——桥接插入点即为网关的 `server.Use`。

新增桥接函数（authctx 包）：

```go
// BridgeJWTClaims 将 go-zero JWT 中间件以原始 claim 名写入的 string context key
// 转换为 typed key。必须在 JWT 验证之后调用（网关注册于 server.Use）。
// 先直接拷贝内部 wire 名（短横线 user-id/user-name/dept-code），再应用
// ClaimMapping 映射（下划线 user_id -> user-id）。
func BridgeJWTClaims(ctx context.Context, mapping map[string]string) context.Context
```

桥接必须同时覆盖两种 claim 命名：
- **短横线** `user-id`/`user-name`/`dept-code`：仓库内签发方（`zerorpc/generatetokenlogic.go:49`、`socketpush/gentokenlogic.go:61`）写入的正是 `user-id`，go-zero 原样写入 string key。桥接遍历 `ContextKeys`，`ctx.Value(key).(string)` 非空则 `WithKey` 写 typed。
- **下划线** `user_id`/`user_name`/`dept_code`：外部 token，由现有 `ApplyClaimMappingToCtx` 按配置（`aigtw.yaml`、`mcpserver.yaml`）映射到内部 typed key。

仅字符串值被桥接（`. (string)` 断言）；`zerorpc` 签发的 int64 `user-id` 仍返回 `""`，与现状一致（audit R1，类型规范化归 `normalize-auth-claims`）。

> **设计修订（用户确认 2026-08-14）**：`user-id` 同时接受 **int64 或 string**。桥接转换不再仅做 string 断言，而是把 `int64`/整数值 `float64` 也转为 string（`fmt.Sprintf` 语义，与 `ClaimString` 的数值处理一致），再写入 typed key。`ApplyClaimMappingToCtx` 的下划线映射同样接受数值。其它非 string/数值类型（bool/数组/对象）仍跳过。此决定缩小了 `normalize-auth-claims` 的必需范围（该任务仍负责完整的类型白名单与缺失/非法值语义）。

桥接落地位置：
- `aigtw`：已有 `server.Use`（auth-type/authorization + `ApplyClaimMappingToCtx`），扩展为 `BridgeJWTClaims`。
- `gtw`：有 `WithJwt` 路由（`routes.go:151`）但无 ClaimMapping，新增 `server.Use` 调 `BridgeJWTClaims(ctx, nil)`。
- `socketgtw`：HTTP 路由 `WithJwt` 为注释状态（`routes.go:21`），Socket.IO 路径已直接写 typed key，无需桥接。

### 阶段 1：仓库内写入方迁移到 typed setter

| 写入方 | 现状 | 迁移 |
|---|---|---|
| `aiapp/aigtw/aigtw.go:83-84` | `context.WithValue(ctx, CtxAuthTypeKey, "user")`、`CtxAuthorizationKey` | `WithAuthType` / `WithAuthorization` |
| `gtw/gtw.go:60` | `CtxAuthTypeKey` | `WithAuthType` |
| `socketapp/socketgtw/socketgtw.go:68` | `CtxAuthTypeKey` | `WithAuthType` |
| `common/socketiox/server.go:537,558,579,594,610,673,698,730,754`（9 处） | `CtxAuthorizationKey` = token | `WithAuthorization` |
| `common/authctx/claims.go:15` `ExtractFromClaims` | `context.WithValue(ctx, key, v)` | `With` 系列（按 key 分发） |
| `common/authctx/claims.go:34` `ApplyClaimMappingToCtx` | 拷贝外部 string key 值到内部 string key | 读外部原 key（string），写内部 typed setter |
| `common/grpcx/metadata.go:71` `ExtractFromGrpcMD` | `context.WithValue(ctx, f.contextKey, val)` | 按 `f.contextKey` 分发到 typed setter |
| `common/mcpx/context_meta.go:33` `ExtractFromMeta` | `context.WithValue(ctx, key, v)` | 按 key 分发到 typed setter |

### 阶段 2：进程内读取方迁移

`GetUserID`/`GetUserName`/`GetDeptCode`/`GetAuthorization` 本身改为 typed getter + 旧 key 回退（阶段 0），因此所有 `authctx.GetX` 调用方（16×aigtw solo、gtw current-user、`common/tool/userutil.go`、MCP echo、StreamEvent）无需改动即受益。

传输边界读取（grpcx `InjectToGrpcMD`、mcpx `CollectFromCtx`）需从 `ctx.Value(f.contextKey).(string)` 改为通过 authctx getter 读取（保持 string 断言与空值过滤语义），确保 typed key 值能上 wire。

### 阶段 3：wire 命名表收口

- `ContextKeys` 保留为 string 切片（wire 顺序契约），但 grpcx/mcpx 对它的使用改为「通过 getter 读进程、通过 wire key 写线上」，不再直接 `ctx.Value(key)`。
- 新增（或保留）一个「typed key ↔ wire 名」映射函数，供 `ExtractFromGrpcMD`/`ExtractFromMeta` 按 wire 名分发到对应 typed setter。

### 阶段 4：测试更新与移除条件

- `authctx/context_test.go`：getter 改为**只读 typed** 的契约测试；移除 string-key 回退测试；新增 `BridgeJWTClaims` 桥接契约测试（短横线直拷、下划线映射、非 string 跳过）。
- `grpcx/metadata_test.go`、`mcpx/context_meta_test.go`：改为通过 setter 写入 typed key 后再验证 wire 行为；保留 wire key 名断言。
- aigtw 三个 validation 测试文件：把 `context.WithValue(ctx, authctx.CtxUserIdKey, "user-1")` 改为 `authctx.WithUserID(...)`。
- 旧 string key 不再作为 getter 读取源（已移除回退）；`BridgeJWTClaims` 是唯一承接 go-zero string-key 写入的入口。string 常量仍保留为 wire/claim 命名表，不做进程读写。

## 风险

| # | 风险 | 缓解 |
|---|---|---|
| R1 | go-zero 中间件写 string key，去掉 getter 回退后身份丢失 | 网关 `server.Use` 桥接中间件（JWT 之后执行）用 `BridgeJWTClaims` 承接；aigtw/gtw 落地，socketgtw 无需 |
| R2 | int64 `user-id`（zerorpc 签发）typed getter string 断言仍返回 `""` | 保持现状；类型规范化属 `normalize-auth-claims`，本任务不改 |
| R3 | 9 处 socketiox 写入漏迁造成事件身份不对称 | 全部迁移 + 契约测试覆盖 |
| R4 | `auth-type` 无读取方，迁移属纯写入侧 | 仅改 setter；wire key `x-auth-type` 不变 |
| R5 | MCP `CollectFromCtx` 继续把 raw Authorization 写入 `_meta` | 本任务不改变；策略归 `enforce-authorization-policy` |
| R6 | trigger 中继链路（`InjectToGrpcMD` 读 typed key）漏迁丢身份 | 传输边界统一走 getter/setter；`grpc_invoker.go` 复核 |
| R7 | 契约测试用 `string("user-id")` 字面量锁定旧 key | 测试改为 typed API + 保留 wire 名断言 |
| R8 | gormx 自己的 typed key（`gormx:user`）与 authctx 混淆 | 不合并；authctx 迁移不触碰 gormx |
| R9 | 外部 token 用下划线 claim（`user_id`），桥接漏映射丢身份 | `BridgeJWTClaims` 先拷内部 wire 名再调 `ApplyClaimMappingToCtx`（aigtw/mcpserver 配置已含 `user_id`） |
| R10 | 桥接中间件在无 JWT 路由上执行 | 无害：无 JWT 时 ctx 无 string claim，桥接为空操作 |

## 兼容与回滚

- 桥接中间件保证 go-zero string-key 写入被承接，线上无需灰度即兼容；无 JWT 路由桥接为空操作。
- 若迁移后出现身份丢失，回滚为在桥接基础上暂时保留 getter string-key 回退（或整体还原）。每阶段独立可回滚。
- wire key、`b64:`、字段顺序全程不变。

## 禁止混合（audit §8.1）

- 不改 claim 接受类型（`ClaimString` 不动）。
- 不改 metadata 重复/冲突行为。
- 不改 raw-token 传播。
- 不改 `b64:`。
- 不实施方法策略。
