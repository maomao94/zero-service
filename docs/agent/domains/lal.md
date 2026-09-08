# LAL 流媒体服务规范

## 适用范围

修改 `app/lalhook`、`app/lalproxy`、`common/lalx`、`common/mediax` 或 LAL 服务间的交互时读取。

## 服务定位与边界

- **lalhook** — REST 服务，接收 lalserver HTTP Notify Webhook 回调，DB 持久化 TS 文件记录，查询 TS 列表。
- **lalproxy** — zRPC 服务，将 lalserver 的 HTTP API 包装为 gRPC 接口，供其他服务调用。
- **lalx** — LAL API 响应类型定义（`GroupData`、`PubSessionInfo`、`LalServerData` 等），被 lalproxy 使用。
- **mediax** — FFmpeg 截图工具，用于从视频流或本地文件截取单帧 JPEG。

注意：**lalhook 和 lalproxy 是两个独立服务**，功能不重叠。lalhook 被动接收 Webhook；lalproxy 主动代理 LAL API。

依据：`app/lalhook/internal`、`app/lalproxy/internal`、`common/lalx/laltype.go`、`common/mediax/mediax.go`。

## lalhook — REST Webhook 服务

### 入口与配置

- `lalhook.go` 标准 go-zero REST 入口：`conf.MustLoad` → `svc.NewServiceContext(c)` → `handler.RegisterHandlers(server, ctx)` → `server.Start()`
- CORS 配置使用 `rest.WithCustomCors`，允许动态 Origin、带凭证、指定允许头。
- `config.Config` 嵌入 `rest.RestConf`，包含 `DB.DataSource` 字段。
- 所有路由超时 2 小时 (`rest.WithTimeout(7200000*time.Millisecond)`)。

依据：`app/lalhook/lalhook.go`、`app/lalhook/internal/config/config.go`。

### 路由分组

两套路由组 (`internal/handler/routes.go`)：
- `/v1/api` — 单端点 `POST /ts/list`：查询 DB 中的 TS 文件记录
- `/v1/hook` — 10 个 Webhook 端点：`onHlsMakeTs`、`onPubStart`、`onPubStop`、`onRelayPullStart`、`onRelayPullStop`、`onRtmpConnect`、`onServerStart`、`onSubStart`、`onSubStop`、`onUpdate`

### Webhook Handler 约定

- 所有 Webhook Handler 接收 lalserver 的 HTTP Notify POST 请求，返回 `EmptyReply`。
- **大多数 Webhook Handler 是桩实现** — 仅打印日志并返回空响应，业务逻辑标注 `// todo`。新增逻辑前确认对应 Handler 是否已有实现。
- `ListTsFilesLogic` 是唯一完整实现的 Handler：使用 `squirrel.SelectBuilder` 构建 SQL，支持按流名称、时间范围、事件类型过滤，查询 `HlsTsFilesModel` 并映射为 `ApiTsFile` 列表。

依据：`app/lalhook/internal/logic/webhook/`、`app/lalhook/internal/logic/api/listtsfileslogic.go`。

### 类型体系

- `types.go` 为 goctl 生成：Webhook 事件类型（`OnPubStartRequest` 等）、API 类型（`ApiListTsRequest`、`ApiListTsReply`、`ApiTsFile`）、聚合类型（`GroupInfo`、`PubInfo`、`SubInfo`、`PullInfo`、`FpsInfo`）、统一响应 `EmptyReply`。
- **lalhook 定义了自己的 LAL 类型，不导入 `common/lalx`**。lalx 提供更强类型（如 `Pushs` 为 `[]*PushSessionInfo` 而非 `[]interface{}`），但 lalhook 选择独立维护。新增 Webhook 业务逻辑时继续使用 `types.go` 中已有类型；如果需要引入 lalx 需要评估向后兼容。

### DB 查询

- Model 层使用 `github.com/Masterminds/squirrel` 构建 SQL 查询。
- `ServiceContext` 持有 `HlsTsFilesModel`，通过 squirrel 查询 TS 文件记录。

依据：`app/lalhook/internal/model/`、`app/lalhook/internal/svc/servicecontext.go`。

## lalproxy — gRPC 代理服务

### 入口与配置

- `lalproxy.go` 标准 go-zero zRPC 入口：`conf.MustLoad` → `svc.NewServiceContext(c)` → `zrpc.MustNewServer`，注册 `lalproxy.RegisterLalProxyServer`。
- `config.Config` 嵌入 `zrpc.RpcServerConf`，额外包含 `NacosConfig`（可选）和 `LalServer` 配置（IP、Port、Timeout）。
- `ServiceContext` 持有 `LalBaseUrl`（格式 `http://{ip}:{port}`）和 `LalClient httpc.Service`（go-zero HTTP 客户端，带超时）。
- gRPC reflection 在 DevMode/TestMode 时启用。

依据：`app/lalproxy/lalproxy.go`、`app/lalproxy/internal/config/config.go`、`app/lalproxy/internal/svc/servicecontext.go`。

### RPC 方法

9 个 RPC 方法，全部代理 lalserver HTTP API：

| RPC | HTTP 方法 | LAL API |
|-----|----------|---------|
| GetGroupInfo | GET | `/api/stat/group?stream_name=` |
| GetAllGroups | GET | `/api/stat/all_group` |
| GetLalInfo | GET | `/api/stat/lal_info` |
| StartRelayPull | POST | `/api/ctrl/start_relay_pull` |
| StopRelayPull | GET | `/api/ctrl/stop_relay_pull` |
| KickSession | POST | `/api/ctrl/kick_session` |
| StartRtpPub | POST | `/api/ctrl/start_rtp_pub` |
| StopRtpPub | (使用 KickSession) | - |
| AddIpBlacklist | POST | `/api/ctrl/add_ip_blacklist` |

### Logic 层约定

- 每个 Logic struct 嵌入 `logx.Logger`，持有 `ctx context.Context` + `svcCtx *svc.ServiceContext`。
- 方法签名: `(in *lalproxy.XxxReq) (*lalproxy.XxxRes, error)`
- 流程: 参数校验 → `svcCtx.LalClient.Do()` HTTP 调用 → JSON 反序列化到 `lalx.*` 类型 → `copier.Copy()` 映射到 protobuf 类型 → 返回。
- 错误处理: 使用 `tool.NewErrorByPbCode(extproto.Code__1_06_THIRD_PARTY, ...)` 报告第三方服务错误。
- LAL API 自身的错误（`ErrorCode` + `Desp`）在响应结构体中返回，不作为 gRPC error。

依据：`app/lalproxy/internal/server/lalproxyserver.go`、`app/lalproxy/internal/logic/`。

### 类型映射

- lalproxy 依赖 `common/lalx` 的类型（`GroupData`、`LalServerData` 等）。
- 使用 `copier.Copy()` 将 `lalx.*` 映射到 protobuf 生成类型。
- LAL 响应 JSON 解析依赖精确的 JSON tag（全部 snake_case）。

## lalx — 类型定义

- 单文件 `laltype.go`，定义 LAL HTTP API JSON 响应的 Go 结构体。
- 核心类型: `GroupData`（聚合所有会话信息）、`PubSessionInfo`、`SubSessionInfo`、`PullSessionInfo`、`PushSessionInfo`（预留）、`FrameData`（FPS 数据点）、`LalServerData`（服务器信息）。
- 所有 struct 字段有显式 JSON tag（snake_case），注释注明对应 LAL API 端点。
- 被 `app/lalproxy` 使用；`app/lalhook` 未导入（使用自己的 types.go）。

依据：`common/lalx/laltype.go`。

## mediax — FFmpeg 截图

- `Screenshotter` 封装 `github.com/u2takey/ffmpeg-go`，支持按时间点截图和按帧索引截图。
- `CaptureFrameToFile(ctx, timePoint, localFilePath)` — 按秒级时间点截图；实时流传 `-1` 取当前帧。
- `CaptureFrameByIndexToFile(ctx, frameIndex, localFilePath)` — 使用 `select=eq(n,{frameIndex})` 滤镜。
- 输出固定为 JPEG (`mjpeg`)，质量 `q:v = 2` (1-31，1=最优)。
- **原子性模式**: 截图后验证文件存在且非空，失败时自动清理无效文件。
- `GenerateTempFilePath(baseDir, ext)` — 生成 `{baseDir}/YYYYMMDD/{uuid}{ext}` 临时路径。
- 详细 debug 日志记录 FFmpeg stderr 输出。

依据：`common/mediax/mediax.go`。

## 反模式

- 在 lalhook Webhook Handler 中直接调用外部服务（Webhook 应快速确认并返回）。
- 在 lalproxy 中将 LAL API 错误映射为 gRPC error（LAL 错误在响应 struct 中）。
- 绕过 ServiceContext 的 `LalClient` 创建新的 HTTP 客户端。
- 在 mediax 截图后不验证文件有效性就直接使用。
- 在 lalhook 中引入 lalx 类型而未评估 lalhook types.go 的类型体系。

## 验证

- lalhook: 验证 Webhook 路由注册和 TS 查询的参数过滤/分页。
- lalproxy: 验证所有 9 个 RPC 方法、HTTP 超时、LAL 错误传递、copier 映射完整性。
- lalx/lalproxy: 确保 JSON tag 与 LAL API 实际返回字段一致。
- mediax: 验证截图成功/失败/空文件/路径不存在场景，确保文件清理逻辑正确执行。
