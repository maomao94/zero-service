# Authorization 传播链路审计报告

- **Task**: `08-14-audit-authorization-propagation`
- **Type**: 只读安全审计（无代码/配置/运行行为变更）
- **Date**: 2026-08-14
- **基线**: 用户已确认默认拒绝基线——原始用户 token 默认不跨服务传播，仅显式 RPC/MCP 委托 allowlist 可分类为 `user-token`；本报告不给任何当前链路擅自加入 allowlist。

## 1. 执行摘要

### 1.1 三处已确认的完整 token 值日志

| # | 位置 | 级别 | 内容 |
|---|---|---|---|
| L1 | `facade/streamevent/internal/logic/upsocketmessagelogic.go:30-31` | Info | Socket.IO JWT/raw Authorization 经 gRPC 传播后在 `UpSocketMessage` 中 `Infof("token: %s", token)` 完整打印 |
| L2 | `aiapp/mcpserver/internal/tools/echo.go:25-28` | Debug | MCP echo 工具从 tool context 读取 `authctx.GetAuthorization` 并 `Debugf("token: %s,...")` 完整打印 |
| L3 | `common/mcpx/auth.go:45-65` | Debug | `extra` map 在 line 61 写入 `extra["authorization"] = token`，line 65 用 `%v` 打印整个 `extra` map，含完整 raw token |

L1 为 Info 级，L2/L3 为 Debug 级；源码可确认完整值会被交给对应日志调用，但生产是否实际写入、可被谁读取及保留多久取决于部署日志级别、sink ACL 和保留策略（U10）。三者均为独立可部署的日志脱敏修复项，回滚时不得被其他策略改动覆盖。

### 1.2 raw token 跨服务转发姿态

- **gRPC**：`common/grpcx/metadata.go:45-59` 的 `InjectToGrpcMD` 把所有非空标准 context 字符串（`authorization` 在首位，`metadata.go:27-34`）经 `md.Set` 注入出站 metadata。`UnaryMetadataInterceptor` / `StreamTracingInterceptor`（`client_interceptor.go:11-20`）被广泛安装在网关、MCP server、Trigger 动态调用等客户端（证据见 §5）。接收端 `ExtractFromGrpcMD`（`metadata.go:62-74`）无 token 验证、无 claims 一致性校验（`server_interceptor.go:10-31`）。
- **MCP**：MCP HTTP transport 用配置的 service token 认证连接（`common/mcpx/client.go:1192-1201`，组装于 `client.go:579`），但每次 tool call 又通过 `CollectFromCtx`（`common/mcpx/context_meta.go:12-23`）把调用方 raw Authorization 写入 `params._meta` 序列化到外部 MCP server（`client.go:774-796,825-827`）。server wrapper 将 raw `_meta` 原样存入 context（`wrapper.go:241-245`），启用 `WithExtractUserCtx` 的工具（echo/progress/modbus，`echo.go:44`、`testprogress.go:89`、`modbus.go:52,75`）可把 raw token 恢复到 context 并继续经嵌套 gRPC 客户端（`aiapp/mcpserver/internal/svc/servicecontext.go:30-35`）重传。
- **Socket.IO**：握手 token 验证后，raw token 被写入 connection/每个事件的 context（`common/socketiox/server.go:537,558,579,594,610,673,698,730,754`），socket 网关的 connect/disconnect/join-room hook 用该 context 调 `StreamEventCli.UpSocketMessage`（`socketapp/socketgtw/internal/svc/servicecontext.go:75-134`），经 `UnaryMetadataInterceptor` 转发到 StreamEvent；下游直接用途仅发现 L1 日志。

结论：当前 raw 用户凭证被通用拦截器机制跨服务转发，但仓库内未发现任何需要接收端用户委托授权的业务消费者（无 allowlist 证据）。按默认拒绝基线，本报告不将任何链路分类为 `user-token`。

## 2. 证据复核结果

对 research 的传播矩阵、身份 claims 矩阵、gRPC metadata 契约、MCP `_meta`/泄漏、policy 输入逐项抽查，**所有被抽查的关键主张均可由代码证实**。补充两处细化（非纠错）：

### R1（细化）token 签发方 claim 命名与映射配置不一致

- 仓库内 token 签发方把 claim 写成 `user-id`（短横线）：`zerorpc/internal/logic/generatetokenlogic.go:49`（int64 值）、`socketapp/socketpush/internal/logic/gentokenlogic.go:61`（string 值）。
- 而 `aigtw.yaml:15-18`、`mcpserver.yaml:22-25` 的 ClaimMapping 是 internal `user-id` ← external `user_id`（下划线）。
- go-zero v1.10.3 `rest/handler/authhandler.go:72-78` 按 claim 名（`k`）把非标准 claims 原样写入 HTTP context，即 token 里是 `user_id` 才有 context key `user_id`。
- 因此 mapping 只在 token 实际携带 `user_id`（下划线）时生效；若生产 token 携带 `user-id`（短横线），`ApplyClaimMappingToCtx`（`common/authctx/claims.go:31-38`）为 no-op。
- 同时 `authctx.GetUserId`（`common/authctx/context.go:22-27`）对 `user-id` 做 string 断言，int64 值（zerorpc 签发）会取到空串，导致 `createsessionlogic.go:33-35` 等依赖点返回 unauthenticated。
- 影响：`normalize-auth-claims` 必须把“生产 token 的 claim key 命名（`user-id` vs `user_id`）与类型（int64 vs string）”作为输入决策，并在迁移前确认实际签发链路。

### R2（细化）aigtw raw-header middleware 不区分路由是否受 JWT 保护

- `aigtw.go:82-85` 对**所有**请求（包括未受 `rest.WithJwt` 保护的 `/health`，`routes.go:18-26`）只要有 `Authorization` 头就写入 context。受保护路由组（`routes.go:37,50,83,198,211,226`）在 JWT 验证通过后才会执行 handler，因此 raw token 只有在验证成功后才会经 gRPC 转发；`/health` 不发起 gRPC 调用。当前代码路径未发现未验证 token 的下游转发。`typed-auth-context-keys` 只负责把该位置纳入写入点清单并保持现有行为；是否按路由验证状态限制写入/传播属于 `enforce-authorization-policy`，不得在 typed-key 迁移中顺带改变。

## 3. 最终传播分类表（默认拒绝基线）

> 模式词汇：`user-token`（原始用户凭证，仅显式委托 allowlist）、`claims-only`（仅规范化身份，信任已认证服务边界）、`service-token`（独立服务凭证）、`none`（两者都不传）、`unresolved`（缺少业务/deployment 证据）。

| # | 路径 | 来源 → 验证点 → context 写入 | 传播载体 | 目标/消费者 | 当前用途 | 推荐模式 | 证据 | 需 owner 确认 |
|---|---|---|---|---|---|---|---|---|
| P1 | Browser → AI gateway → gRPC clients | HTTP `Authorization`；`rest.WithJwt`（routes.go:37,50,83,198,211,226）；global middleware 写 `auth-type=user` + raw header（`aigtw.go:82-85`）；claim mapping（`aigtw.go:90-98`） | gRPC metadata `authorization` + `x-user-*`（`aigtw/internal/svc/servicecontext.go:69-74`） | aichat / aisolo / 知识库 | aigtw 把 user-id 放入 AISolo 请求字段做数据隔离（`createsessionlogic.go:33-46`、`chatlogic.go:37-48`）；raw token 未发现业务消费 | `claims-only` | user-id 隔离在请求字段完成，无需 raw token；接收端无 token 重验 | 确认 aisolo/aichat 无用户委托需求（默认视为无） |
| P2 | Browser → generic gtw → gRPC clients | HTTP Authorization；`rest.WithJwt`（`gtw/internal/handler/routes.go:151`）；gtw 全局 middleware 只写 `auth-type=user`，不写 raw header（`gtw/gtw.go:57-63`） | gRPC metadata claims（`gtw/internal/svc/servicecontext.go:59-62`） | zerorpc / file RPC | `GetCurrentUser` 读 `user-id`（`getcurrentuserlogic.go:34`）；raw token 未从本网关写 context | `claims-only` | 本网关未传播 raw token | raw token 需求未发现 → 视为 `none`（如未来要传播需单独批准） |
| P3 | Socket.IO handshake/event → StreamEvent gRPC | `socket.Handshake.Auth.Token`；`OnAuthentication`（`socketiox/server.go:496-506`）+ socket 网关 JWT/claims validator（`socketgtw/internal/svc/servicecontext.go:41-74`） | 事件 context raw token（`server.go:537,558,...`）→ gRPC metadata（`servicecontext.go:24-33`） | StreamEvent `UpSocketMessage` | 仅 L1 日志 | `none` | 下游用途仅日志；若未来需要身份，改为 `claims-only` | 若 StreamEvent 未来做身份鉴权，须确认合法 claims 来源 |
| P4 | MCP client → MCP server HTTP 连接 | 配置 ServiceToken；服务端 dual verifier 常量时间比较（`common/mcpx/auth.go:22-31`） | 配置后发送 `Authorization: Bearer <serviceToken>`（`client.go:1192-1201`，组装 `client.go:579`） | MCP HTTP server（SSE / Streamable） | 建立/认证 client 边界 | `service-token` | 配置启用时是独立服务凭证模式；实际配置与外部 server 信任仍属部署证据 | 确认所有部署均配置并校验 service token；无需用户 token |
| P5 | 用户 context → MCP tool `_meta` | 已有 raw Authorization；其先前验证取决于来源（HTTP JWT 可已验证，gRPC/MCP metadata 也可仅由调用方断言）；`CollectFromCtx` 不验证（`context_meta.go:12-23`） | `params._meta` → 外部 MCP server（`client.go:774-796,825-827`） | 外部 MCP server；wrapper 可恢复并再入 gRPC（`wrapper.go:231-250`） | 无业务读取 raw meta 的消费者；`GetMeta` 仅扩展点（`context_meta.go:44-50`） | `claims-only`（默认）；`user-token` 仅对显式批准委托的 tool/server | 默认拒绝基线；echo/progress/modbus 无委托证据；不得把 context 中存在 token 等同于已验证 | 每个来源边界、外部 MCP server / tool 的委托与数据处置由 owner 确认 |
| P6 | 外部用户 JWT → MCP server HTTP auth → SDK/tool 边界 | HTTP bearer；`NewDualTokenVerifier` JWT 解析 + claim mapping（`auth.go:34-63`，解析器 `tool/authutil.go:18-47`） | verifier 返回 `TokenInfo.Extra`（含 raw token）；SDK 是否及如何把 Extra 转入 `req.Params._meta`/tool context 属外部依赖行为，仓库源码未证实 | MCP SDK handler；仓库 wrapper 仅从 `req.Params.GetMeta()` 提取（`wrapper.go:231-250`） | echo 是 tool-context token/用户名的日志与展示 sink（`echo.go:25-44`），但“外部 JWT Extra → echo context”的完整连接仍需 SDK/集成验证；modbus 在 context 有值时可经 gRPC 转发 | `claims-only` | HTTP JWT 在 verifier 边界已验证；其后 SDK context transfer 为未知，不据此声称已完成端到端传播 | 验证 SDK transfer/retention；若工具必须委托给下游接收端，再逐工具批准 |
| P7 | gRPC incoming → nested gRPC outgoing | incoming `authorization` metadata；`LoggerInterceptor` 仅提取不验证（`server_interceptor.go:10-31`；`metadata.go:62-74`） | gRPC metadata 重传（任意安装 `UnaryMetadataInterceptor`/`StreamTracingInterceptor` 的客户端，如 `aigtw/internal/svc/servicecontext.go:69-74`、`app/trigger/internal/invoke/grpc_invoker.go:35`） | 各 RPC 接收端 | 接收端特定逻辑；直接 raw 消费者仅 StreamEvent（L1）与 MCP echo（L2） | `unresolved`（全局）；按 receiver 分别 `claims-only`/`service-token`/`none` | 无方法级策略注册表；授权/冲突规则待 `enforce-authorization-policy` | 方法/工具委托表 + 内部 gRPC 对等鉴权机制 |
| P8 | Provider 集成 | 静态 API key（非用户 Authorization） | 出站 Authorization（`aiapp/aichat/internal/provider/openai.go:99`；`common/einox/knowledge/embedder.go:96`） | 外部模型/embedding API | 服务凭据 | `service-token` | 与用户传播无关 | 无 |

### 3.1 不允许的 `user-token` 归类

按默认拒绝基线，P1–P8 **均无**显式用户委托证据，本报告不将任何路径标记为 `user-token`。任何未来的 `user-token` 归类必须由业务 owner 提供：该 RPC/MCP 工具确实需要接收端执行用户授权/委托，且接收端具备用户 token 验证职责。

## 4. gRPC 重复/冲突契约与建议处理

### 4.1 当前契约（已复核，与 archived `08-13-extract-grpc-raw-codec/prd.md:65-80` 一致）

| 场景 | 当前行为 | 证据 |
|---|---|---|
| 出站已有 metadata + context Authorization | `md.Copy()` 后 `md.Set` 整体替换该 key；context 值胜出，未覆盖的 key 保留 | `metadata.go:45-59`；`metadata_test.go:58-71` |
| 入站重复值 | 只读 `values[0]`，不拒绝、不比较重复值 | `metadata.go:66`；`metadata_test.go:73-80` |
| 空首值 + 非空后续值 | 整字段跳过，后续值被忽略 | `metadata.go:66`；`metadata_test.go:73-83` |
| Authorization 与 `x-user-*` 冲突 | 各自独立拷贝进 context，不做 token 解析或交叉校验 | `metadata.go:27-34,62-74` |
| 既有进程 context + 空入站 metadata | 空/缺省不覆盖，外层旧值保留 | `metadata.go:63-72` |
| 空/非字符串出站 context | 不发出；已有出站 metadata 因未调用 `Set` 而保留 | `metadata.go:49-58`；`metadata_test.go:86-101` |
| 非可打印值 / 非 ASCII | 整串 `b64:` + 标准 Base64 编码；接收端对任何 `b64:` 前缀解码，解码错误不上抛 | `metadata.go:36-56,68-70` |

### 4.2 建议的未来处理（report-only 先行）

每个冲突类别都需独立批准，`enforce-authorization-policy` 不得自行选择：

1. 拒绝重复值 / 仅接受单个非空值 / 要求重复值相等——三选一由安全 owner 决定。
2. 定义 token 派生 claims 是否覆盖 metadata `x-user-*`，以及无 token 时的行为。
3. 冲突识别需要信号而非丢弃：当前提取丢弃后续值，无法区分「无重复」「有重复且一致」「有重复且冲突」。迁移第一步应加 content-free 观测（见 §7）统计 duplicate count、empty-first、conflict，再决定策略。

## 5. MCP `_meta` 边界与泄漏

### 5.1 边界与生命周期

- MCP HTTP transport 在配置 ServiceToken 时以该凭证认证（`client.go:1192-1201`）——这是 P4 `service-token` 层；所有部署是否配置并被服务端校验需环境确认。
- 每次 tool call，`CollectFromCtx` 把调用方 authctx 非空字符串（含 raw Authorization）拷入新 map，注入 W3C trace 后 `params.SetMeta` 序列化（`context_meta.go:12-23`；`client.go:774-796,825-827`）。
- server wrapper 读 `req.Params.GetMeta()`，`WithMeta` 原样存 map 到 context（`wrapper.go:241-245`），`WithExtractUserCtx` 工具再恢复到标准 context keys（`wrapper.go:247-251`）。
- `WithMeta` 无克隆、无脱敏、无过期；仓库内无业务 `GetMeta` 消费者（`context_meta.go:44-50` 仅测试外无调用）。

### 5.2 泄漏面核查结论

| Sink | 证据 | 状态 |
|---|---|---|
| StreamEvent Info 日志 | `upsocketmessagelogic.go:30-31` | 确认完整值日志调用；生产写入/可见性取决于 U10 |
| MCP echo Debug 日志 | `echo.go:25-28` | 确认完整值日志调用（context 有 token 时）；生产启用取决于 U10 |
| MCP auth Debug extra map | `auth.go:45-65` | 确认完整值日志调用（Debug 级）；生产启用取决于 U10 |
| MCP `_meta` wire 序列化 | `context_meta.go:12-23`、`client.go:787-790` | 传输必要暴露面；策略需决定是否携带 raw token |
| gRPC metadata | `metadata.go:27-59` | 传输必要暴露面；策略需决定 |
| MCP call 错误/进度日志 | `wrapper.go:215-229`、`client.go:791-794` 不含 `_meta`；progress `token` 为关联 ID 非 Authorization（`client.go:808-829,831-840`） | 未发现 raw token |
| trace span/attributes | `trace/carrier.go:41-49` 只注入 OTel span context；`carrier.go:51-84` `AnyCarrier` 仅作为 `_meta` 的 TextMapCarrier 适配（MCP 场景） | 未发现 raw token |
| DB/cache/event/file 持久化 | 负面搜索无命中 | 未发现 |
| metrics label | 负面搜索无命中 | 未发现 |

## 6. Claims 安全属性分类

| Claim | 安全属性 | 证据 |
|---|---|---|
| `user-id` | **安全属性（授权/数据隔离）** | aigtw 从 context 取 user-id 放入 AISolo 请求字段（`createsessionlogic.go:33-46`、`chatlogic.go:37-48`、`listsessionslogic.go:27-34`）；AISolo 校验请求 user-id 与记录 owner 一致（`getinterruptlogic.go:48`），session/knowledge store 以 user_id 条件过滤（`aisolo/internal/session/gormx_store.go`、`common/einox/knowledge/store_*.go`）。注意：隔离判断消费的是**请求字段** user-id（由 aigtw 从已验证 context 派生），不是接收端 gRPC metadata 直接消费；gRPC 直连调用方可自行伪造请求字段，因此隔离有效性依赖「aigtw 是唯一受信入口」这一部署假设。 |
| `user-name` | 信息属性（展示/审计） | MCP echo 展示（`echo.go:25-40`）；GORM audit helper 暴露 name（`common/gormx/user_context.go:85-91`）；未发现授权决策。 |
| `dept-code` | 潜在数据范围属性，**未证实**用于授权 | `tool/userutil.go:63-96` 回退首个部门；Trigger proto 有显式 dept_code 字段与校验，但未发现 authctx→Trigger 的授权决策。显式请求字段与 verified context metadata 不得互相替代。 |
| `auth-type` | 当前无消费者；可被 gRPC/MCP metadata 伪造 | 无 getter/业务消费者；`gtw/gtw.go:57-63`、`aigtw/aigtw.go:82-85`、`socketgtw/socketgtw.go:65-71`、`mcpx/auth.go:30,47` 写入；接收端不重验。 |

信任等级：JWT 经 go-zero middleware 或 `tool.ParseToken` 验证后属已验证；MCP service-token 请求的 `_meta` 身份为调用方提供 metadata；gRPC `x-user-*`/`x-auth-type` 为调用方 metadata，接收端不校验、不与 Authorization 调和。

## 7. Receiver-first 迁移、无内容观测与回滚

### 7.1 迁移顺序（实施任务 `enforce-authorization-policy` 的 PRD 要求）

1. 冻结并批准方法/工具策略表，identify owners；**不改 sender**。
2. Receiver 观测：presence、mode、duplicates、conflicts、validation outcome、caller service，不记录 token 值。
3. Receiver 兼容：同时接受 legacy raw-token metadata 与目标 claims/service credential；冲突处理先 report-only。
4. 按服务/方法在配置或 allowlist 后开启 enforcement，保留 legacy 兼容开关。
5. 按窄 route/tool cohort 改 sender：`none`/`claims-only` 停止 raw token；批准处加 service credential；仅批准委托路径保留 user token。
6. 验证混合版本行为；legacy 流量低于批准阈值/窗口后才移除兼容路径。

### 7.2 无 token 内容观测字段

transport（http/socketio/grpc/mcp）、caller service、receiver service、method/tool、selected mode、credential-present boolean、claim-presence bitset、duplicate count、empty-first boolean、conflict boolean、validation result/reason category、policy version。**禁止**记录 token、前缀/后缀、可指纹 issuer 的长度、JWT claims payload 整体、`_meta`、headers，以及可跨系统跟踪的 hash（除非安全方批准短期 keyed correlation）。

### 7.3 回滚策略

- Receiver 双模式接受保持，sender 按 cohort 切换；配置/策略与代码独立版本化，telemetry 记录 policy version。
- 先回滚 sender cohort，再关 receiver 兼容。
- 本任务期间保持 wire key 与 `b64:` 契约。
- 三处 token 日志脱敏为独立可部署修复项，回滚其他改动不得恢复。

## 8. 后续子任务输入与禁止混合项

### 8.1 `typed-auth-context-keys`

- **输入**：直接 string-key writer/getter 清单——HTTP（`aigtw.go:82-85,90-98`，`gtw/gtw.go:60`，`socketgtw/socketgtw.go:68`）、go-zero claims→context（go-zero v1.10.3 `authhandler.go:72-78`）、Socket.IO 事件 context 写入（`server.go:537,558,579,594,610,673,698,730,754`）、authctx claims（`claims.go:9-18,31-38`）、grpcx 提取（`metadata.go:62-74`）、mcpx 提取/raw-meta（`context_meta.go:27-50`、`wrapper.go:231-250`）。保留 5 个 wire 字符串与顺序（`authctx/context.go:14-20`、`grpcx/metadata.go:27-34` 契约测试锁定）。
- **禁止混合**：不改 claim 接受类型、metadata duplicate 行为、raw-token 传播、`b64:`、方法策略。

### 8.2 `normalize-auth-claims`

- **输入**：`ClaimString` 宽松转换（`claims.go:41-56`，测试 `claims_test.go:9-28` 锁定 float64 整数值、分数 `%g`、bool/array/map `%v`、nil `<nil>`）、映射方向 internalKey→externalKey（`claims.go:22-38`）、JWT 来源、MCP verifier（`auth.go:34-63`）、Socket 仅 string 拷贝（`server.go:517-527`）、必需身份消费者（尤其 AI 数据隔离 aigtw→aisolo 链路）。**必须纳入本报告 R1**：token claim key 命名（`user-id` vs `user_id`）与类型（int64 vs string）的产物一致性验证。
- **禁止混合**：不同时做 typed-key 切换；不收缩 Authorization；不强制 transport 冲突。

### 8.3 `enforce-authorization-policy`

- **输入**：本报告 §3 分类表（经业务 owner 批准）、每方法/工具模式表、receiver 信任/鉴权机制、duplicate/conflict 规则（§4.2）、raw-meta 策略、泄漏修复（L1–L3）、content-free 指标/配置/灰度、回滚窗口。Receiver-first 兼容为 PRD 要求。
- **禁止混合**：不升级 `b64:`；不捆绑 claim 规范化或 typed-key 语义变更；不在无逐边界证据下全局删除 Authorization。

### 8.4 顺序约束

父任务 PRD（`08-14-auth-context-hardening/prd.md:9-20`）要求 audit → typed keys → claim normalization → policy enforcement，各子任务开发前/后均有独立人工门禁；本审计完成并经用户确认前不进入下一子任务。

## 9. 已知未知 / 待 owner 确认

| # | 事项 | 需要谁确认 |
|---|---|---|
| U1 | 哪些 RPC 方法与 MCP 工具被授权接收端用户 token（委托 allowlist） | 业务 owner（默认：无） |
| U2 | 下游服务是否独立重验用户 JWT，还是信任已验证内部调用方的 claims | 安全/deployment owner |
| U3 | 内部 gRPC 对等端认证机制：网络隔离、TLS/mTLS、service token 或其他 | deployment owner |
| U4 | 重复/冲突 metadata 规则（拒绝/相等/单值）与 token 派生身份是否覆盖 `x-user-*` | 安全 owner |
| U5 | 外部 MCP server 的数据处置/保留保证，以及是否允许接收 raw 用户凭证 | 每个外部 MCP server owner |
| U6 | `dept-code` 与 `auth-type` 是安全属性还是信息属性；缺失时的要求 | 产品/安全 owner |
| U7 | 生产 token 签发方实际 claim key（`user-id` 还是 `user_id`）与类型；`normalize-auth-claims` 的兼容窗口 | 业务/部署 owner（R1 细化） |
| U8 | 生产 Socket metadata 实际配置的 claim keys（`SocketMetaData` 配置驱动） | deployment owner |
| U9 | 混合版本部署兼容窗口与回滚时长 | deployment owner |
| U10 | 生产日志级别、sink ACL、保留期与脱敏策略（决定 L1–L3 实际可达性） | deployment owner |
| U11 | 仓库外服务是否重验被转发的用户 token | 外部服务 owner |

## 10. 交付与门禁状态

- 本任务为只读审计，未修改任何 Go/配置/proto/生成/部署文件；唯一产物为本报告及 research 既有文档。
- 按 implement.md 门禁，本报告与分类结论需经用户确认后，才可提交、归档并规划 `typed-auth-context-keys`。
- 复用的基础证据文档：`research/propagation-matrix.md`、`research/identity-claims-matrix.md`、`research/grpc-metadata-conflicts.md`、`research/mcp-meta-and-leakage.md`、`research/policy-migration-inputs.md`、`research/unknowns-and-negative-searches.md`。
