# zero-service 网关标准开发流程（go-zero）

面向新增或改造 HTTP 网关、网关接口的落地规范。错误码语义以 [`error-codes.md`](./error-codes.md) 为准，本文只回答“代码怎么写、错误从哪里出、怎么验收”。

已按 `feature/openapp-opengtw` 分支源码核对；涉及 go-zero 内置行为的地方标注了出处，未实现的能力不写入本文。

## 1. 适用范围与分层

### 1.1 三层职责

| 层 | 职责 | 目录示例 | 对外形态 |
| --- | --- | --- | --- |
| HTTP 网关 | 协议适配、鉴权、参数解析、转发、统一信封 | `app/livegtw`、`openapp/opengtw` | `HTTP 200 + {code,msg,data}` |
| 业务服务 | 领域校验、事务、错误码构造 | `app/live`、`app/podengine` | gRPC status + `ErrorInfo.reason` |
| gRPC 契约 | 字段与错误码语义 | `app/live/live.proto`、`facade/openapi/openapi.proto` | 六位码写入 `reason` |

调用链示例：

```text
浏览器/第三方 ──HTTP──▶ livegtw / opengtw ──gRPC──▶ app/live / facade/openapi ──▶ 下游服务
```

### 1.2 分层规则

1. **网关不写业务判断**。网关 logic 只做“请求 → RPC → 响应”，参考 `app/livegtw/internal/logic/live/createmeetinglogic.go`：解析参数后直接调用 `LiveRpcCli.CreateMeeting`，失败原样返回。
2. **业务服务不依赖网关包**。业务错误用 `common/tool` 构造，不 `import common/gtwx`。
3. **错误码只在业务服务定义语义**。网关不新增错误码，只做透传、兜底和折叠。
4. **生成代码可改但契约优先**。修改 `.api` 后必须执行对应目录的 `gen.sh`（`goctl api format` + `goctl api go`），不要只改 `internal/types` 或 `routes.go`。

## 2. 路由、前缀与鉴权

### 2.1 前缀约定

| 网关 | 业务前缀 | 管理/特殊前缀 | 鉴权方式 |
| --- | --- | --- | --- |
| `app/livegtw` | `/live/v1` | `/webhook/livekit` | JWT + `MeetingAuth`；webhook 验签 |
| `openapp/opengtw` | `/v1` | `/open-admin/v1` | Basic；管理面 JWT + `UserAuth` |
| `gtw`（存量） | `gtw/v1`、`app/user/v1`、`file/v1` 等 | — | JWT（未按本文迁移） |

规则：

- 新网关使用 `/<业务域>/v<版本>`，例如 `/live/v1`；同一网关内不要混用有无版本段。
- 管理面单独前缀（如 `/open-admin/v1`），部署时由 nginx 限制为内网访问，见 `docs/open-api/README.md`。
- 健康检查保持公开，例如 `GET /v1/health`，返回 `{"code":0,"msg":"ok","data":{"status":"ok"}}`。

### 2.2 JWT 路由

在 `.api` 的路由组声明 `jwt: JwtAuth`，由 goctl 生成 `rest.WithJwt(serverCtx.Config.JwtAuth.AccessSecret)`：

```text
@server (
	prefix:     live/v1
	group:      live
	jwt:        JwtAuth
	middleware: MeetingAuth
)
service livegtw {
	@handler listMeetings
	get /listMeetings (ListMeetingsRequest) returns (ListMeetingsReply)
}
```

对应生成结果见 `app/livegtw/internal/handler/routes.go`（`rest.WithMiddlewares(...)` + `rest.WithJwt(...)` + `rest.WithPrefix("/live/v1")`）。

JWT 校验失败不会进入 handler，必须由 main 统一接管：

```go
server := rest.MustNewServer(c.RestConf,
	gtwx.CorsOption(),
	rest.WithUnauthorizedCallback(gtwx.UnauthorizedHandler),
	rest.WithNotFoundHandler(http.HandlerFunc(gtwx.NotFoundHandler)),
	rest.WithNotAllowedHandler(http.HandlerFunc(gtwx.MethodNotAllowedHandler)),
)
```

见 `app/livegtw/livegtw.go`。`gtwx.UnauthorizedHandler` 先写 `200 + 100401`；go-zero 随后补写的 401 会被 `HeaderOnceResponseWriter` 丢弃，因此响应仍是信封（实现注释见 `common/gtwx/response.go`）。

JWT 通过后，业务中间件负责把 claims 桥接到 `authctx` 并校验身份：

- `app/livegtw/internal/middleware/meetingauthmiddleware.go`：没有 `userId` 时返回 `100401`，文案 `缺少用户身份`。
- `openapp/opengtw/internal/middleware/userauthmiddleware.go`：只做 `authctx.BridgeJWTClaims`，管理面身份由下游 logic 使用。

### 2.3 Basic 鉴权

只有开放平台使用 Basic。实现见 `openapp/opengtw/internal/middleware/openauthmiddleware.go`：

- 缺少/非法凭据 → `130102`；应用不存在或已停用 → `130101`（两者不区分，避免泄露应用状态）。
- 密钥只存 `sha256`，用常数时间比较（`internal/security`）；凭据缓存 60s，管理面变更后立即刷新。
- 鉴权失败同样走 `gtwx.WriteError`，HTTP 状态为 200。

### 2.4 公开路由

必须显式声明并说明原因，当前只有四类：

| 路由 | 原因 | 保护手段 |
| --- | --- | --- |
| `GET /v1/health` | 存活探针 | 无 |
| `GET /live/v1/joinMeetingByTicket` | 访客票据入会 | 票据本身校验 |
| `POST /live/v1/generateToken` | 内部签发 token | `TokenSign.SignKey` 校验 |
| `POST /webhook/livekit` | LiveKit 回调 | webhook 签名验签 |

不要把“内部调用”当作公开路由的理由；内部调用应走 gRPC 或内网 + 鉴权。

## 3. 统一响应与 0 成功

### 3.1 契约

```json
{"code": 0, "msg": "ok", "data": {}}
```

- 业务响应统一 **HTTP 200**，客户端以 `code == 0` 判断成功。
- `code` 为六位业务码，`0` 是唯一成功码（`errx.SuccessCode`，见 `common/errx/normalize.go`）。
- 失败时省略 `data`，`msg` 可展示；基础设施错误折叠为 `系统繁忙`。
- 已接入网关的业务响应带 `X-Request-Id`；框架级 `413/503`、panic 与 nginx 错误不保证带该头（见 §5）。

### 3.2 handler 唯一写法

```go
func GetMeetingHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetMeetingRequest
		if err := httpx.Parse(r, &req); err != nil {
			gtwx.WriteParseError(r.Context(), w, err)
			return
		}

		resp, err := live.NewGetMeetingLogic(r.Context(), svcCtx).GetMeeting(&req)
		if err != nil {
			gtwx.WriteError(r.Context(), w, err)
			return
		}
		gtwx.WriteOK(r.Context(), w, resp)
	}
}
```

三个出口的职责：

| 函数 | 用途 | 实际结果 |
| --- | --- | --- |
| `gtwx.WriteOK(ctx, w, data)` | 成功 | `200 {"code":0,"msg":"ok","data":...}` |
| `gtwx.WriteError(ctx, w, err)` | 业务/下游错误 | `200 {"code":<六位>,"msg":"..."}` |
| `gtwx.WriteParseError(ctx, w, err)` | 参数解析失败 | `200 {"code":100101,"msg":<解析错误>}` |

不要直接调用 `xhttp.JsonBaseResponseCtx`：它只套信封，遇到 gRPC 错误时取的是 gRPC 数字码（3/5/16），遇到普通 error 时是 -1。`gtwx.WriteError` 会先经 `gtwx.NormalizeError` 归一化并校验：码必须在 `extproto` 枚举中，否则降级为 `100999` 并写日志。

### 3.3 X-Request-Id

全局注册 `gtwx.RequestIDMiddleware`（`common/gtwx/requestid.go`）：

```go
server.Use(gtwx.RequestIDMiddleware)
```

- 优先复用请求头 `X-Request-Id`；为空、超过 128 字符或含控制字符时重新生成。
- 写入响应头和 context，handler 用 `gtwx.RequestIDFromContext(ctx)` 读取。
- `openapp/opengtw` 通过 `internal/middleware/requestid.go` 别名复用同一实现；`GatewayHandler` 在 body 带 `tid` 时以 `tid` 覆盖响应头，缺失时使用中间件生成的 ID。
- 鉴权/404/405 等绕过业务中间件的兜底出口由 `ensureRequestID` 补 ID（`common/gtwx/response.go`）。

## 4. 错误码使用

### 4.1 extproto v2

错误码为 `ABCDEF` 六位：`A` 来源、`BC` 功能模块、`DEF` 模块内序号。定义在 `third_party/extproto.proto`，语义说明见 `docs/error-codes.md`。

- `A`：`0` 成功、`1` 平台、`3` 设备、`5` 云端，其余预留。
- `BC`：当前只启用 `00` 通用、`30` 开放平台；新增功能域必须先申请，不抢段。
- `00` 内按 DEF 分段：`100-199` 参数、`200-299` 数据、`300-399` 缓存、`400-499` 认证、`500-599` 业务、`600-699` 外部、`700-799` 限流、`800-899` 网关、`900-999` 兜底。
- `0` 只能表示成功，禁止用于构造错误。

### 4.2 业务服务：构造错误

```go
return nil, tool.NewErrorByPbCode(extproto.Code__1_00_PARAM_MISSING, "meetingNo 不能为空")
```

需要保留底层原因时：

```go
return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询会议失败")
```

- 六位码写入 gRPC `ErrorInfo.reason`，gRPC Code 按 `http_code` 推导。
- `NewErrorByPbCode(extproto.Code__0_OK)` 会直接返回内部错误，防止“成功码当错误”。
- 实现见 `common/tool/errorutil.go`；业务侧示例见 `app/live/internal/logic/startmeetingrecordlogic.go`。

> `http_code=429` 已映射为 `ResourceExhausted`；`_1_00_RATE_LIMITED (100701)` 会以限流语义返回，不会被误判为 500。

### 4.3 网关：错误归一化与折叠

`errx.PublicError` 的优先级（`common/errx/`）：

```text
NormalizeError
  → PublicError
     → xerrors.CodeMsg
     → CodeFromStatus（gRPC ErrorInfo.reason，仅 extproto 已登记码）
     → gRPC Code 兜底
  → IsDefinedCode 校验（不在枚举内 → 100999 + 日志）
```

- `CodeFromStatus` 只在 `reason` 是 extproto 中已登记的码时采用；未登记的数字码按内部错误折叠，非数字 reason 走 gRPC Code 映射（`common/errx/codec.go`）。
- `FoldPublic` 把 `100201`、`100301`、`100302`、`100303`、`100601`、`100602`、`100998` 折叠为 `100999`，对外文案 `系统繁忙`；真实原因写日志。
- 普通 `error`（未结构化、也未携带 `CodeMsg`）会被折叠为 `100999 系统繁忙`，原始文本只写服务端日志。因此业务与下游必须先转换成 extproto 错误，不能把 SQL、地址、堆栈直接抛给网关。

### 4.4 设备/云端原始码（DJI）

`extproto` 是唯一的平台信封码表。DJI 的 `dji_error_code.proto` 是独立的厂商错误码体系，与平台码不互通、不映射、不透传，不进入信封，也不写入 `ErrorInfo.reason`：

- 设备执行结果走业务字段：`CommonRes{code:-1, reasonCode:312014}` → HTTP `data.reasonCode`，信封 `code` 保持 `0`。
- 网关 `NormalizeError` 只认 extproto 中已登记的码；未登记的原始码一律折叠为 `100999 系统繁忙`，原文进日志。
- 第三方平台的故障（超时、鉴权、限流、5xx）由厂商适配层转换成 extproto 码后再以 error 返回，见 `docs/error-codes.md` §5.2。

## 5. 统一出口

| 场景 | 统一出口 | `code` | HTTP |
| --- | --- | --- | --- |
| JWT 缺失/过期/签名错误 | `rest.WithUnauthorizedCallback(gtwx.UnauthorizedHandler)` | `100401` | 200 |
| JWT 有效但无用户身份（livegtw） | `MeetingAuth` → `gtwx.WriteError` | `100401`（`缺少用户身份`） | 200 |
| Basic 凭据缺失/错误 | `OpenAuthMiddleware` → `gtwx.WriteError` | `130102` | 200 |
| 应用不存在/已停用 | `OpenAuthMiddleware` → `gtwx.WriteError` | `130101` | 200 |
| 参数解析失败 | `gtwx.WriteParseError` | `100101` | 200 |
| 下游 gRPC 业务错误 | `gtwx.WriteError` | 原始六位码 | 200 |
| 下游基础设施错误 | `gtwx.WriteError` | 折叠为 `100999` | 200 |
| 未在 extproto 登记的码 | `gtwx.WriteError` → `NormalizeError` | `100999`（原文进日志） | 200 |
| 未转换的普通 `error` | `gtwx.WriteError` → `NormalizeError` | `100999 系统繁忙` | 200 |
| 路由不存在 | `rest.WithNotFoundHandler(gtwx.NotFoundHandler)` | `100801` | 200 |
| 方法不允许 | `rest.WithNotAllowedHandler(gtwx.MethodNotAllowedHandler)` | `100802` | 200 |

以下不是信封，属于框架/部署层行为，排查时不要误判为漏接：

| 场景 | 响应 | 来源 |
| --- | --- | --- |
| 请求体超过 `MaxBytes` | `413`，空 body | go-zero 内置 |
| 请求超时 | `503`，纯文本 `Request Timeout` | go-zero 内置 |
| panic | `500`，空 body | go-zero 恢复中间件 |
| nginx 不可达 | `502/504` | nginx |
| gRPC 直连 | gRPC status + `reason` | 无 HTTP 信封 |

以上内置行为的现有记录见 `docs/error-codes.md` §6.4；本文未新增对这些响应的包装。

## 6. 明确禁止事项

| 禁止 | 原因 | 应改为 |
| --- | --- | --- |
| `httpx.ErrorCtx` / `httpx.OkJsonCtx` | 输出裸 JSON/纯文本，HTTP 状态随错误变化，客户端无法只认 `code` | `gtwx.WriteError` / `gtwx.WriteOK` |
| 直接 `xhttp.JsonBaseResponseCtx` | 不解析 extproto，错误码会变成 gRPC 数字码或 `-1` | 同上 |
| 裸 `http.Error` | 纯文本 + 4xx/5xx，破坏信封 | `gtwx.WriteError`；仅 webhook 等协议例外可用 |
| 把 gRPC 数字码（3/5/16）直接返回 | 客户端无法区分业务语义 | `gtwx.WriteError` 或 `tool.NewErrorByPbCode` |
| 用 `0` 构造错误 | 成功与失败歧义 | 选一个非 0 的六位码 |
| 在 `msg` 中放 SQL、连接串、堆栈 | 泄露内部信息 | 真实原因写日志，对外用固定文案 |
| 用 HTTP 4xx/5xx 表达业务失败 | 与契约冲突 | HTTP 200 + 非 0 `code` |
| 网关 logic 里写业务规则或直接查库 | 破坏分层，错误码归属混乱 | 下沉到业务服务 |
| 随手新增公开路由 | 绕过鉴权面 | 先声明并说明保护手段 |
| 在新网关复制 `oryxgtw`/`bridgegtw` 的 handler | 它们是未迁移实现 | 以 `livegtw`/`gtw`/`opengtw` 为模板 |

当前仓库中仍有这些写法的存量位置：`app/oryxgtw/internal/handler/hook/hookhandler.go`、`app/bridgegtw/internal/handler/bridgeGtw/pinghandler.go`、`aiapp/ssegtw/internal/handler/sse/*`。只作为迁移对象，不作为新代码参考。

## 7. 新增网关/接口检查清单

### 7.1 契约与代码

- [ ] 新增/修改 `.api`，声明 `prefix`、`group`、`jwt`、`middleware`；执行对应目录 `gen.sh`。
- [ ] main 注册完整启动模板（参考 `app/livegtw/livegtw.go`）：
  - `gtwx.CorsOption()`
  - `rest.WithUnauthorizedCallback(gtwx.UnauthorizedHandler)`
  - `rest.WithNotFoundHandler(http.HandlerFunc(gtwx.NotFoundHandler))`
  - `rest.WithNotAllowedHandler(http.HandlerFunc(gtwx.MethodNotAllowedHandler))`
  - `server.Use(gtwx.RequestIDMiddleware)`
  - `server.Use(gtwx.RequestLogMiddleware)`
  - `gtwx.SetLogOkHandler()`
- [ ] handler 只使用 `gtwx.WriteParseError` / `gtwx.WriteError` / `gtwx.WriteOK` 三个出口。
- [ ] 业务错误由业务服务用 `tool.NewErrorByPbCode` / `NewErrorByPbCodeWrap` 构造，码来自 `extproto`，不使用 `0`。
- [ ] 鉴权失败、身份缺失、Basic 失败都有明确错误码，不返回裸状态码。
- [ ] 下游错误依赖 `ErrorInfo.reason`；不要自己解析 gRPC 数字码。

### 7.2 测试

至少覆盖：

- [ ] 成功信封：HTTP 200、`code=0`、`data` 正确（参考 `openapp/opengtw/internal/handler/gateway/gatewayhandler_test.go` 的 `TestGatewaySuccessEnvelope`）。
- [ ] 参数错误：非法 JSON → `100101`（`TestGatewayInvalidJSONReturnsParseError`）。
- [ ] 鉴权失败：JWT 缺失/身份缺失 → `100401`（`app/livegtw/internal/middleware/meetingauthmiddleware_test.go`）；Basic 失败 → `130101/130102`（`openapp/opengtw/internal/middleware/openauthmiddleware_test.go`）。
- [ ] 404/405/未认证兜底 → `100801/100802/100401`（`common/gtwx/response_test.go` 的 `TestGatewayFallbackHandlers`）。
- [ ] 下游基础设施错误折叠为 `100999 系统繁忙`（`TestGatewayFoldsInfrastructureError`、`TestWriteErrorAlwaysHTTP200`）。
- [ ] 未登记码/设备原始码折叠为 `100999 系统繁忙`（`TestWriteErrorFoldsVendorCode`、`...OverGRPC`）。

执行：

```bash
gofmt -l common/errx common/gtwx gtw app/livegtw openapp/opengtw aiapp/ssegtw aiapp/aigtw
go test ./common/errx/... ./common/gtwx/... ./gtw/... ./app/livegtw/... ./openapp/opengtw/... ./aiapp/ssegtw/... ./aiapp/aigtw/...
go vet ./common/errx/... ./common/gtwx/... ./gtw/... ./app/livegtw/... ./openapp/opengtw/... ./aiapp/ssegtw/... ./aiapp/aigtw/...
```

### 7.3 curl 验收

以下命令使用占位地址与占位凭据，不包含真实密钥：

```bash
BASE_URL=http://127.0.0.1:11002   # livegtw；opengtw 用 11005，路径换成 /v1/...

# 1) 公开探针：期望 200 + code=0
curl -sS -i "$BASE_URL/v1/health"

# 2) 未认证访问受保护接口：期望 200 + code=100401
curl -sS -i "$BASE_URL/live/v1/listMeetings"

# 3) 路由不存在：期望 200 + code=100801
curl -sS -i "$BASE_URL/live/v1/not-exist"

# 4) 方法不允许（对公开 GET 路由发 POST）：期望 200 + code=100802
curl -sS -i -X POST "$BASE_URL/live/v1/joinMeetingByTicket"

# 5) 参数解析失败（公开接口 + 非法 JSON）：期望 200 + code=100101
curl -sS -i -X POST "$BASE_URL/live/v1/generateToken" \
  -H 'Content-Type: application/json' \
  -d 'not-json'

# 6) 请求 ID 回显：响应头应包含 X-Request-Id
curl -sS -i "$BASE_URL/live/v1/listMeetings" -H 'X-Request-Id: demo-req-1'
```

判定标准：除明确例外外，HTTP 状态必须是 200，body 必须是 `{code,msg,data}`；失败响应不带 `data`，且带 `X-Request-Id`。

## 8. 迁移状态与例外

| 服务/场景 | 状态 | 说明 |
| --- | --- | --- |
| `app/livegtw` | 已接入 | 业务、鉴权、参数、404/405、下游错误均走 `gtwx` |
| `openapp/opengtw` | 已接入 | 开放面 Basic + 管理面 JWT；`tid` 回显 |
| `gtw` | 已接入 | 中间件/兜底 handler 已补齐；微信支付回调为协议例外 |
| `app/oryxgtw` | 未迁移 | `httpx.ErrorCtx` / `OkJsonCtx` |
| `app/bridgegtw` | 未迁移 | grpc-gateway + `httpx`，无信封 |
| `aiapp/aigtw` | 协议例外 | OpenAI 兼容错误结构（`gtwx.SetOpenAIErrorHandler`），不套 `{code,msg,data}`；401/404/405 也保持 `{"error":{...}}` |
| `aiapp/ssegtw` | 部分/例外 | 启动模板已补齐；`/ping` 走信封，SSE 流式响应写流后不套信封 |
| `POST /webhook/livekit` | 协议例外 | 验签失败 `401`，处理失败 `500`，成功走 `gtwx.WriteOK`；LiveKit 依靠状态码决定重推 |
| `413 / 503 / panic` | 框架例外 | go-zero 内置响应，不是信封，见 §5 |
| gRPC 直连 | 协议例外 | 使用 gRPC status + `ErrorInfo.reason`，无 HTTP 信封 |

迁移顺序建议：先补测试，再替换 handler 出口，最后在 main 打开 404/405/未认证回调；不要一次性改动业务逻辑。

## 9. 关键文件索引

| 文件 | 作用 |
| --- | --- |
| `third_party/extproto.proto` | 六位错误码定义与 `http_code` |
| `docs/error-codes.md` | 错误码语义、v1→v2 映射、网关契约 |
| `common/gtwx/response.go` | `SuccessCode`、`PublicError`、三个写出口、兜底 handler |
| `common/errx/codec.go` | `CodeFromStatus`、`FoldPublic`、`PublicMessage` |
| `common/gtwx/requestid.go` | `X-Request-Id` 中间件与 context 读取 |
| `common/gtwx/cors.go` | 统一 CORS，暴露 `X-Request-Id` |
| `common/tool/errorutil.go` | `NewErrorByPbCode` / `NewErrorByPbCodeWrap` |
| `common/djisdk/error.go` | DJI 原始码 `PublicCode` / `GRPCStatus` |
| `app/livegtw/livegtw.go` | livegtw 统一出口接线 |
| `app/livegtw/internal/handler/routes.go` | JWT、中间件、前缀的实际生成结果 |
| `app/livegtw/internal/middleware/meetingauthmiddleware.go` | 身份缺失 → `100401` |
| `app/livegtw/internal/handler/webhook/webhookhandler.go` | webhook 协议例外 |
| `openapp/opengtw/opengtw.go` | opengtw 统一出口接线 |
| `openapp/opengtw/internal/middleware/openauthmiddleware.go` | Basic → `130101/130102` |
| `openapp/opengtw/internal/handler/gateway/gatewayhandler.go` | 开放面转发与 `tid` 回显 |
| `openapp/opengtw/internal/logic/gateway/gatewaylogic.go` | 下游错误折叠 |
| `common/gtwx/response_test.go`、`common/gtwx/codec_test.go` | 公共出口与错误码测试 |
| `openapp/opengtw/internal/handler/gateway/gatewayhandler_test.go` | 网关信封/解析/折叠测试 |
| `gtw/gtw.go` | 未迁移存量参考（反面示例） |
