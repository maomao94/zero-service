# design.md — oryxgtw / oryxserver

## 架构与边界

```
Oryx (Docker)                          zero-service
+------------------+   HTTP callback   +---------------------+
| SRS http_hooks   |------------------>| oryxgtw (REST :11004)|
|   -> Oryx callback|                  |  /v1/hook (action 分发)|
+------------------+                   +---------------------+
                                            | gRPC (记录生命周期)
                                            v
Java 服务 ---Nacos---> oryxserver (zRPC :21016) ---HTTP(Bearer)---> Oryx /terraform/v1/*
```

- `oryxgtw` 被动接收 Oryx 回调；`oryxserver` 主动代理 Oryx OpenAPI。两者独立，功能不重叠（对齐 lalhook/lalproxy 边界）。
- 命名：HTTP 服务 `oryxgtw`（gateway），gRPC 服务 `oryxserver`（业务主入口）。

## 数据流与契约

### oryxgtw（HTTP，goctl api）
- `.api` 定义 types（字段 snake_case，`json:"..."` tag 与 Oryx 回调一致）：
  - `HookRequest`（公共：`request_id/action/opaque/vhost/app/stream/param` + 可选 `uuid/artifact_code/artifact_path/artifact_url/prompt/result`）
  - `HookReply`：`code int`
- 路由：`POST /v1/hook` → 按 `req.Action` 分发到 5 个 handler（onPublish/onUnpublish/onRecordBegin/onRecordEnd/onOcr）。
- `on_publish` 放行由返回 `{"code":0}` 实现；其余忽略响应（同样返回 code 0）。
- 处理逻辑：日志记录 + 空实现 stub（预留落库/转发）。
- **回调凭证校验**：`.api` 声明 `middleware: OryxHookAuth`，`internal/middleware` 校验请求体 `opaque` 字段（Oryx 回调透传的凭证）与配置 `HookOpaque` 一致，否则返回 `{"code":1}` 拒绝；`HookOpaque` 为空时跳过校验。

### oryxserver（gRPC，goctl rpc protoc）
- `oryxserver.proto`：字段 snake_case + 全量 `json_name`（camelCase，遵循契约规范）；响应**无 `code` 字段**（错误统一走 gRPC error）。
- RPC 分组：
  - **平台级接口**（业务服务经 Nacos 调用）：`Versions`、`RecordQuery`、`RecordApply`、`RecordEnd`、`RecordRemove`、`RecordFiles`、`RecordGlobs`、`RecordPostProcessing`、`DvrQuery`、`DvrApply`、`DvrFiles`、`HooksApply`、`HooksQuery`、`RecordList`、`RecordDelete`。
  - **⚠️ 内部 Hook**（proto 中显式分组 + 注释标注，**仅 oryxgtw 调用**）：`RecordBeginHook`、`RecordEndHook`（gtw 收到 Oryx 回调后的落库通道，业务服务请勿调用）。
- 错误语义：
  - Oryx 调用失败（网络/协议/业务错误）→ 统一返回 gRPC error `extproto.Code__1_06_THIRD_PARTY`，错误信息（含 Oryx 返回的 code/message）在 gRPC error message 中透传，Java 只处理 error 通道。
  - 本地参数校验错误 → `extproto.Code__1_01_PARAM_MISSING` 等。

### record 生命周期（gtw → server 落库）
- **数据源**：Oryx 回调 `on_record_begin` / `on_record_end` 经 oryxgtw 调 `RecordBeginHook` / `RecordEndHook`（同步，失败仅记日志，不影响回调响应），oryxserver 落库 `model/gormmodel/Record`（gormx + LegacyStringBaseModel，Dev/Test 自动迁移）。
- **状态机**：`Status` 1-录制中（begin 写入）→ 2-已完成 / 3-失败（end 写入，`artifact_code != 0` 为失败）；begin 幂等（uuid 存在则更新回录制中），end 若因 begin 丢失未命中则**补插**完整记录。
- **删查 grpc**：`RecordList`（分页，支持 uuid/vhost/app 精确 + stream 模糊 + status 多选过滤，按 begin_time 倒序）、`RecordDelete`（仅删除本地生命周期记录，不影响 Oryx 文件）。
- **播放地址**：约定拼接 `{host}/terraform/v1/hooks/record/hls/{uuid}/index.mp4`（m3u8 为 `.m3u8`），m3u8 由 Oryx 内建路由动态生成；数据存于 `Record.ArtifactURL`（来自 on_record_end 回调）。

### 调研补充
- Oryx `record/files` 无分页/筛选参数，全量返回；SRS `/api/v1/` 仅 `streams`/`clients` 支持 `start/count`。分页能力由 oryxserver 落库后自建查询（`RecordList`）补全。

## SRS HTTP API 代理（Oryx 已代理 /api/v1/*）

- **Oryx 鉴权**（源码 `ossrs/oryx` platform/service.go L374-394 + utils.go）：仅 `/api/v1/versions` 免鉴权（健康检查）；其余 `/api/*` 必须 `Authorization: Bearer <SRS_PLATFORM_SECRET>`（或 `?token=<JWT>` 同签名）。Oryx 纯透传（path/query 不改写，仅去 Server/CORS 头）；SRS 自身 http_api.auth 默认 off，Bearer 校验只在 Oryx 层 → SDK 复用现有 `OryxConfig.Secret` 即可覆盖。
- **分页**：仅 `streams`/`clients` 支持 `start/count`；SRS v5/v6.0 `count = max(10, atoi(count))`——**下限 10**（count=0/1/-1 均返回 10），无"全量"语义、**无 total**；vhosts 无分页全量；summaries 单对象。SDK 对非正值参数不携带，交由 SRS 默认值（start=0, count=10）。
- **端点映射**（增加 6 个 RPC，平台级分组，字段强类型）：
  - `SrsVersions` → GET `/api/v1/versions`（data: major/minor/revision/version）
  - `SrsStreams(SrsStreamsReq{start,count})` → GET `/api/v1/streams/`（top-level `streams` + server/service/pid；元素含 kbps/publish/video/audio）
  - `SrsClients({start,count})` → GET `/api/v1/clients/`（top-level `clients`）
  - `SrsVhosts` → GET `/api/v1/vhosts/`（top-level `vhosts`，无分页）
  - `SrsSummaries` → GET `/api/v1/summaries`（data: ok/now_ms/self/system，`ilde_time` 为 SRS 源码拼写原样透传）
  - `SrsRequests` → GET `/api/v1/tests/requests`（调试；注意 SRS 实际注册在 `tests/requests`，`/api/v1/requests` 命中前缀导航）
- **响应差异**：SRS 响应顶层字段因端点而异（data / streams / clients / vhosts），`common/oryxx/srs.go` 的 `callSrs` 按各自结构解析；SRS 错误为 **HTTP 200 + code 非 0**（mux 未匹配才是真 404）→ 统一 `OryxError`，逻辑层转 `extproto.Code__1_06_THIRD_PARTY` 与 Oryx 组一致。

## 契约与字段对齐（已从 Oryx 源码核对）

- Oryx 回调 JSON 字段是 snake_case，`.api` 直接使用相同 json tag，不二次转换。
- Oryx OpenAPI 请求/响应字段同样 snake_case；oryxserver 侧 HTTP 请求体/响应体使用本地 snake_case struct 解析（参照 lalproxy 的 `reqData`/`lalx` 模式）。
- gRPC 对外字段遵循项目规范：proto 字段 snake_case + `json_name` camelCase。
- 已核对字段（来源 `platform/dvr-local-disk.go`、`platform/callback.go`、`platform/srs-hooks.go`）：
  - `record/query` data：`all`、`home`、`globs`、`processCpDir`
  - `record/apply` 请求 `all`；`record/end`、`record/remove` 请求 `uuid`；三者 data 为 null
  - `record/files` data 为数组，元素含 `uuid/vhost/app/stream/progress/update/nn/duration/size`
  - `hooks/apply` 请求 `target/opaque/all/host`；`hooks/query` data 含 `req/res/target/opaque/all/host`
  - `mgmt/versions` data：`version`
- **字段整改结论（Oryx 接口检核）**：13 个代理接口请求字段全部齐全，无缺失（Oryx handler 实际解析字段 = token/all/globs/postProcess/postCpDir/uuid/target/opaque/all/host，SDK 均已覆盖）；`?action=start/stop` 参数不存在（Oryx 仅按 body 字段判断）；`app/stream/vhost` 仅出现在 files/回调**响应**中，无请求过滤参数。`RecordList` 为本地落库查询（非 Oryx 代理），分页+uuid 已支持；Oryx `record/files` 本身无分页能力（单次 HScan）。详见 `research/oryx-api-fields.md`。

## 鉴权

- Oryx API 鉴权为 header `Authorization: Bearer <apiSecret>`，其中 `apiSecret = SRS_PLATFORM_SECRET`（Oryx 环境变量，即「系统配置 > OpenAPI」页下发的 Bearer token）。
- `oryxserver` 配置 `OryxConfig.Secret`，写入 `etc/oryxserver.yaml`（占位值，不提交真实凭据）。

## 服务装配（对齐 lal 模式）

- `oryxgtw`：`rest.RestConf` + CORS（`rest.WithCustomCors`）+ `handler.RegisterHandlers`；`ServiceContext` 注入 `OryxHookAuth rest.Middleware`（routes.go 通过 `rest.WithMiddlewares` 挂载）。
- `oryxserver`：`zrpc.RpcServerConf` + `NacosConfig`（可选注册）+ `OryxConfig{Ip,Port,Timeout,Secret}`；`ServiceContext` 持有 `OryxClient *oryxx.Client`（`common/oryxx` 封装，统一 baseUrl + Bearer 注入）；gRPC reflection 在 Dev/Test 模式启用；`s.AddUnaryInterceptors(grpcx.LoggerInterceptor)`。

## 公共包 `common/oryxx`

- 封装 Oryx HTTP API 为**标准 SDK**，`oryxx.NewClient(oryxx.Config{Ip,Port,Timeout,Secret})` 返回 `*Client`。
- 底层 `call(ctx, method, apiPath, body)` 拼接 baseUrl、统一注入 `Authorization: Bearer <Secret>`、解析统一响应 `{code,data}`。
- DTO 定义于 `oryxtype.go`（对齐 `lalx/laltype.go`）：`RecordQueryData`、`RecordFile`、`HooksQueryData`、`HooksApplyReq`、`DvrQueryData`、`DvrFile`。
- SDK 不依赖任何服务的 `internal/` 或生成 pb，也不依赖 `extproto`。

### 错误处理（SDK 结构化错误 + 服务层统一 gRPC error）

- SDK 层（`common/oryxx`）保持结构化错误，但**服务层只识别 error，不再透传业务码**：
  - `OryxError{Code int32, Message string}` + `IsOryxError(err)`（`errors.As` 解包）——保留以防未来 Oryx 返回「200 + code != 0 + message」的场景；当前 Oryx 业务错误常态是非 200（500）+ 文本信息。
  - `call` 返回 `(data json.RawMessage, err error)`：
    - 195非 200：响应为 JSON `{code,data}` 时取 `OryxError{Code, Message}`；否则 `OryxError{Code: HTTP 状态码, Message: 响应文本}`。
    - 200 且 `code != 0`：`OryxError{Code, Message}`（防御未来）。
    - 网络/解析错误：普通 `error`。
- 服务层（logic）：调用 SDK 业务方法，任何 error（`OryxError` 或普通 error）统一返回 gRPC error `tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, ...)`；不把错误码塞回响应（proto 已无 `code` 字段），错误信息经 gRPC error message 透传给 Java。

## Trade-offs

- **透传 Oryx 错误**：不把 Oryx 业务错误转成 gRPC error，避免信息丢失（对齐 lalproxy）。
- **暂不落库/不做工单关联**：最小实现，回调仅记录日志，后续扩展。
- **POST 空 body**：无参 POST（query/files/hooks query）请求体为 nil，Oryx `ParseBody` 对空 body 容忍；鉴权靠 header Bearer。
- **go-zero httpc body 约束**：`mapping.Marshal` 仅支持 struct，故请求体统一用本地 struct（带 snake_case json tag），不用 `map[string]any`。
- **回调凭证**：Oryx 回调透传 `opaque` 字段作为凭证；`HookOpaque` 为空时不做校验（开发期免配）。
- **端口**：oryxgtw 11004、oryxserver 21016（登记于 `docs/service-ports.md`）。

## 运维与回滚

- 两个服务独立编译、独立部署，互不影响；回滚按服务单独回滚。
- 生成文件（pb.go/routes/types）由 gen.sh 生成，不手工修改。
