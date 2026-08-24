# Oryx 网关与 SRS API 代理（oryxgtw/oryxserver/oryxx）

## 适用范围

修改 `app/oryxgtw`（REST 回调网关）、`app/oryxserver`（gRPC 代理 + StreamRelay 转推）、`common/oryxx`（SDK）时读取。

## 1. Scope / Trigger

- 本规范覆盖三层链路：Oryx 回调 → `oryxgtw`（HTTP 11004）→ gRPC → `oryxserver`（21016）→ Oryx HTTP API（`/terraform/v1/*` 与 `/api/v1/*`）。
- `oryxserver` 另有 **StreamRelay** 链路：业务 gRPC → 内部 FFmpeg 进程（拉外部流转推 SRS，动态启停，第一版仅 copy 转推）。
- 涉及跨层契约（Oryx/SRS 第三方 HTTP、gRPC proto、DB record 表、FFmpeg 进程生命周期），深度契约必须清晰。
- **关键前提（为何自管 FFmpeg）**：SRS/Oryx 全家族无动态中继拉流接口、无按流转码接口——Oryx `/terraform/v1/ffmpeg/transcode/*` 为**全局单任务**（同一时刻只转 1 条、自动选最新流、输入必须为 SRS 本地流）；SRS Ingest 配置驱动；Dynamic Forward 方向相反（relay push）。详见 `.trellis/tasks/08-25-stream-relay/research/oryx-relay-transcode.md`。

## 2. Signatures

- Oryx 平台 API（`/terraform/v1/*`）：13 个端点，SDK 方法见 `common/oryxx/oryx.go`；除 `/terraform/v1/mgmt/versions` 外全部需要 Bearer 鉴权。
- SRS API（`/api/v1/*`，Oryx 已透传代理）：6 个核心端点：

```go
func (c *Client) SrsVersions(ctx) (SrsVersionsData, error)      // GET /api/v1/versions
func (c *Client) SrsStreams(ctx, start, count int32) (SrsStreamsData, error) // GET /api/v1/streams/
func (c *Client) SrsClients(ctx, start, count int32) (SrsClientsData, error) // GET /api/v1/clients/
func (c *Client) SrsVhosts(ctx) (SrsVhostsData, error)         // GET /api/v1/vhosts/
func (c *Client) SrsSummaries(ctx) (SrsSummariesData, error)   // GET /api/v1/summaries
func (c *Client) SrsRequests(ctx) (SrsRequestsData, error)     // GET /api/v1/tests/requests
```

- gRPC：`OryxServer` 服务，平台级 17 个 RPC + `RecordList`/`RecordDelete`（本地落库）+ `RecordBeginHook`/`RecordEndHook`（⚠️ 内部钩子，仅 `oryxgtw` 调用）+ `StreamRelay`/`StreamRelayStop`（转推）。

### Stream Relay（FFmpeg 转推）— 接口与进程契约

```protobuf
message StreamRelayReq {
  string source_url = 1;   // 源流地址（RTMP/RTSP/HTTP-FLV/HLS 等 FFmpeg 支持的协议）
  string app = 2;          // 目标应用名，默认 "live"（RelayConfig.DefaultApp）
  string stream = 3;       // 目标流名，不填则自动生成 UUID（tool.SimpleUUID）
}
message StreamRelayRes {
  string task_id = 1;      // 转推任务 ID（停止转推时使用）
  string app = 2;          // 目标应用名（实际使用值）
  string stream = 3;       // 目标流名（自动生成时返回生成值）
}
message StreamRelayStopReq { string task_id = 1; }
message StreamRelayStopRes {}
```

- **播放地址约定**（业务侧按协议自行拼接，Oryx 无获取播放地址接口）：
  `RTMP: rtmp://<host>:<rtmp_port>/{app}/{stream}`；`HTTP-FLV: http://<host>:<http_port>/{app}/{stream}.flv`；`HLS: http://<host>:<http_port>/{app}/{stream}.m3u8`；`WebRTC: webrtc://<host>:<rtc_port>/{app}/{stream}`。
- **任务管理**（`app/oryxserver/internal/relay`）：`Manager.Start(source, target) (taskID, error)`（**无 ctx 参数**——绑定 gRPC 请求 ctx 会在 RPC 返回时杀进程，内部必须用 `context.Background()` 派生可取消 ctx）、`Stop(taskID)`、`TaskIDs()`、**`StopAll()`**；任务存 `sync.Map`，仅内存、不落库。
- **FFmpeg 命令**：`ffmpeg -i <source_url> -c copy -f flv rtmp://<srs>/<app>/<stream>`；构建方式 `ffmpeg.Input(src)` → `ffmpeg.OutputContext(ctx, ..., {"c":"copy","f":"flv"})` → `WithErrorOutput(stderrBuf)` → `Compile()`（内部 `exec.CommandContext`）→ `cmd.Start()`（非阻塞）+ goroutine `cmd.Wait()`（退出打 error 日志：task_id/source/stderr 并清理 map；`ctx.Err()!=nil` 为主动停止则 Info）。
- **鉴权查询参数在 target 尾部**：`authQuery`（`?secret=xxx` / `?sign=md5(pushkey)`）由 `buildAuthQuery` 拼在**整个 target URL 后**（`target += "?" + authQuery`），最终命令为 `-f flv rtmp://<srs>/<app>/<stream>?secret=...`。SRS/Oryx 用 `?secret=`、WVP/ZLM 用 `?sign=`，authStyle 配置切换。
- **进程生命周期契约**（服务退场必须停掉 ffmpeg，否则孤儿进程继续推流）：
  - Manager 内部持有 `sync.WaitGroup`，`Start` 成功即 `wg.Add(1)`，`watch` 的 `Wait()` goroutine 里 `defer wg.Done()`。
  - `StopAll()` 遍历所有任务 `Cancel()` 后 `wg.Wait()` 阻塞到**全部** ffmpeg 真正退出（`exec.CommandContext` 的 cancel → SIGKILL，ms 级）。
  - 接入 go-zero：`main` 里 `waitRelayStop := proc.AddShutdownListener(relayMgr.StopAll); defer waitRelayStop()`（**必须** defer 返回的 `waitForCalled`，否则 `s.Start()` 返回后 main 退出、监听器 goroutine 的 StopAll 未跑完）。
  - `proc.SetTimeToForceQuit(c.GracePeriod)`（config 里 `GracePeriod time.Duration`，默认 10s）必须在 `s.Start()` 前调用。
- **容器编排兜底**：容器停止（`docker stop`/k8s 优雅终止 → PID 1 收到 SIGTERM → follow 上述优雅路径；`kill -9 容器` → 内核销毁整个 cgroup，ffmpeg 同死）。**无需** `PR_SET_PDEATHSIG`/进程组；macOS 本机开发无 PDEATHSIG，需手动 `kill`。
- **配置**：`DeployMode`（`standalone`/`cluster`，默认 standalone，对齐 ieccaller）；`RelayConfig{SrsRtmpAddr, DefaultApp}`；`MqttConfig`（可选，仅 cluster 用；cluster 且无 broker → `logx.Must` 快速失败）。
- **分布式停止（cluster 可选）**：本地未命中 → 经 `common/mqttx/broadcast` 广播（prefix `oryx/server`，topic `oryx/server/broadcast`，reply `oryx/server/broadcast_reply/{instanceId}`，`BroadcastReply` 默认 TTL 10s）；仅**持有任务的节点**回 ack（成功/失败都回），其余节点返回 `broadcast.ErrSkipAck` 保持沉默（避免假成功）；忽略自环（`AckTopic == 本机 reply topic`）。
- **单节点（standalone）行为**：零 MQTT 依赖；`StreamRelayStop` 未命中本地任务 → 明确错误「任务不在当前节点」，不 panic。

## 3. Contracts

- **鉴权**：请求头 `Authorization: Bearer <SRS_PLATFORM_SECRET>`；`OryxConfig.Secret` 对应 `OryxConfig.Secret`（yaml `OryxConfig.Secret`，env 来源 `SRS_PLATFORM_SECRET`）由 `common/oryxx` 的 `NewClient` 统一注入，服务层不重复加头。`versions`（mgmt 与 SRS 两侧）免鉴权。
- **SRS 重复推流（同 `(app, stream)`）行为**：已存在活跃发布者时，SRS **保留旧连接、拒绝新连接**（`Stream: existing or busy`/code `1028`），把新连接拒绝映射为 ffmpeg 的 `Error opening output ... Input/output error` + `exit status 251`。**不是**"新连者踢旧连者"。因此同一 target 并发推流，旧任务存活、新任务 ffmpeg 立刻退出。若要踢旧改用 `?replace=on`（SRS 3.0+，含鉴权时拼 `?secret=...&replace=on`）且播放端需重连；或改不同 stream key。
- **响应结构**：
  - Oryx mgmt：`{"code":0,"data":...}`，业务错误常为 **HTTP 500 + 纯文本**（无 `message` 字段；`SystemComplexError/AppError` 为 `{code:N,data:"msg"}`）。
  - SRS：`{"code":0,"server":...,"service":...,"pid":...,data|streams|clients|vhosts}`；错误为 **HTTP 200 + `{code:非0}`**（mux 未匹配才是真 404）。
- **分页语义（SRS v5/v6）**：仅 `/api/v1/streams/`、`/api/v1/clients/` 支持 `start/count`；`count = max(10, atoi(count))`（**下限 10**，count=0/1/-1 都返回 10，无全量、无 total）；vhosts 无分页全量。翻页需自累加 start。
- **端点命名陷阱**：`/api/v1/summaries`（复数）；`/api/v1/tests/requests`（`/api/v1/requests` 命中前缀导航不是请求 dump）；无 `/api/v1/hosts`。
- **SRS vhosts `id` 是字符串**：`/api/v1/vhosts` 元素的 `id` 返回数字字符串（如 `"8286"`），`SrsVhost.ID` 类型必须是 `string`；proto `SrsVhostItem.id` 为 int64，映射时 `strconv.ParseInt`（失败容错为 0）。streams/clients 的 `id` 为数字，保持 int64。
- **DB record 表**（`model/gormmodel/Record`）：表名 `oryx_record`（`TableName()` 显式声明）；索引 `uq_oryx_record_uuid` + `idx_oryx_record_{vhost,app,stream,status,begin_time,end_time}`（前缀随表名）；begin_time/end_time 为 `time.Time` + `timestamp`（非 int64，逻辑层 `time.Now()` 显式赋值，不加 autoCreateTime）；gormx 自动迁移仅 Dev/Test；`Status` 1-录制中/2-已完成/3-失败（`artifact_code != 0` → 失败）；begin/end 按 `uuid` 幂等（end 未命中时**补插**完整记录）。
- **回调协议**：gtw http 回调（`/v1/hook`）与 Oryx 同 payload，`HookRequest` 已覆盖全部 action 字段：公共字段 `request_id/action/opaque/vhost/app/stream`；`on_record_begin` + `uuid`；`on_record_end` + `uuid/artifact_code/artifact_path/artifact_url`；`on_ocr` + `uuid/prompt/result`（Oryx AI OCR 回调，忽略响应错误）；`on_publish` + `param`（返回 `{"code":0}` 放行推流）。`HookOpaque` 为空跳过校验。响应统一 `{"code":0}`。
- **on_hls 不属于 HTTP 回调**：`seqno/duration/size` 的 on_hls 消息由 Oryx 内部消费（`record consume msg`），不会 POST 到 gtw；因此 `RecordEndHookReq` 的 `duration/size/nn` 字段位已预留但 Oryx 回调不携带，落库保持 0。
- **gRPC 字段**：snake_case + `json_name` camelCase；响应无 `code` 字段。
- **端口**：oryxgtw 11004、oryxserver 21016（`docs/service-ports.md`）。
- **生成规范**：proto 变更后运行 `app/oryxserver/gen.sh`（goctl rpc protoc + --client=false）；生成的 logic/server/svc 为标准 camelCase 命名（`recordendhooklogic.go`/`oryxserverserver.go`/`servicecontext.go`），不要混入 snake_case 副本（会与 goctl 产物并存错乱）；logic 实现由 `git show HEAD:` 参考或 copy 旧实现后再填空。

## 4. Validation & Error Matrix

| 条件 | 错误行为 |
| --- | --- |
| Oryx 非 200 响应 | SDK 返回 `*OryxError{Code: HTTP状态码, Message: 文本}`（JSON 时取 `{code,data}`） |
| Oryx/SRS 200 且 `code != 0` | `*OryxError{Code: N, Message...}`（SRS Message 固定 `"SRS API 业务错误"`） |
| 网络/JSON 解析失败 | 普通 `error`（带 cause） |
| 以上任一 error 上抛到 RPC 层 | gRPC error `tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 Oryx API 失败")`（SRS 用同码，message "调用 SRS API 失败"） |
| `postProcess != "post-cp-file"` | Oryx 返回 "invalid post process"（默认 500） |
| `record/end` 指向非 live 任务 / `record/remove` uuid 不存在 | Oryx 500 错误（正常业务错误，不可重试语义按调用方处理） |
| `StreamRelay` 缺 source_url | gRPC error `extproto.Code__1_01_PARAM_MISSING` |
| `StreamRelayStop` 本地未命中且非 cluster | gRPC error（明确文案「任务不在当前节点」）；cluster 广播超时（10s 无 ack）→ 同样失败错误 |
| FFmpeg 拉流失败/流断开 | 后台 `Wait()` goroutine 打 error 日志（task_id/source/err/stderr），任务自动清理；RPC 无同步错误（启动即返回） |
| 同一 `(app,stream)` 二次推流（并发） | 新任务 ffmpeg 立即退出：`Error opening output ... Input/output error` + `exit status 251`（SRS 拒绝新连接，见"SRS 重复推流行为"）；旧任务存活 |
| 服务退场（SIGTERM/SIGINT） | `proc.AddShutdownListener` 触发 `StopAll()` → `wg.Wait()`，全部 ffmpeg 退出后再 `waitForCalled` 返回，进程退场；超 `GracePeriod`（10s 默认）proc 强杀 |
| 服务被 `kill -9`/崩溃 | 无 cleanup 可跑，ffmpeg 成孤儿继续推流；容器部署由 cgroup 兜底（进程随容器死）；裸进程需 PDEATHSIG/编排兜底 |

## 5. Good/Base/Bad Cases

- Good：`SrsStreams(0, 20)` → 前 20 条；`count=0`（缺省）→ 10 条。
- Base：`SrsVersions` 无鉴权可用（健康检查）。
- Bad：SRS 返回 200 + code 1061（raw api 未启用）——SDK 已归 `OryxError`，不会 panic；调用方不能期待 HTTP 错误码。

## 6. Tests Required

- SDK：mock `httpc`（httpc.Service）单测：鉴权头注入、非 200 → `OryxError{Code: HTTP状态码}`、200+code 非0 → OryxError、data/streams 顶层字段解析。
- Logic：err 上抛时断言 gRPC status code == `extproto.Code__1_06_THIRD_PARTY`。
- proto 生成后：`go test ./app/oryxserver/...` + `git diff --check`。

## 7. Wrong vs Correct

#### Wrong
```go
// proto 响应带 code 字段 + gRPC error 也返回业务码 —— 双通道冲突
return &RecordQueryRes{Code: err.Code}, nil // 错误：Java 只见 code 不见原因

// SRS 分页全量语义错误假设
count = -1 // 期望全量 —— SRS v6 实际返回 10 条
```

#### Correct
```go
// proto 响应不含 code；错误统一 gRPC error，message 透传 Oryx 原文
return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 Oryx API 失败")

// 全量需循环累加 start（每次 10 条直到返回不足 count）
```

- StreamRelay 进程退场——注册 ShutdownListener 并守护等待，而非裸 defer。同时停止 FFmpeg 包的 std log 打印（`LogCompiledCommand` 是包级全局，需 `.Silent(true)`）：

```go
// Correct: stop-all 接入 go-zero 优雅关闭，且保证等待全部退出
waitRelayStop := proc.AddShutdownListener(ctx.RelayManager.StopAll)
defer waitRelayStop()
defer s.Stop()

// Correct: 关掉 ffmpeg-go 包内 std log，改由 logx 记录
cmd := out.Silent(true).Compile()
logx.WithContext(ctx).Infof("compiled command: ffmpeg %s", strings.Join(cmd.Args[1:], " "))
```

## 反模式

- 用 `logx.Logger.Warnf`（logx 无该 API，编译失败）——用 `Errorf`/`Infof`。
- `Save()` 按 uuid 幂等——主键是自增 id，应 `where(uuid).First` 判断后 Create/Updates。
- 在 `common/oryxx` 引入服务内 `internal/`、生成 pb 或 `extproto` 依赖（sdk 必须零依赖）。
- 业务服务直接调用 `RecordBeginHook`/`RecordEndHook`（内部钩子，仅 gtw 链路）。
- 把 `SrsVhost.ID` 定为 int64——`/api/v1/vhosts` 返回字符串 id，json 解构报 `cannot unmarshal string into ... int64`，整个响应 503。
- 时间字段用 int64 Unix 毫秒或给业务时间字段加 `autoCreateTime` 并同时显式赋值——统一 `time.Time` + 逻辑层赋值。
- 手工创建 snake_case 版 logic/server/svc 文件——与 goctl 生成命名混存（详见"生成规范"）。
- 把 FFmpeg 进程 ctx 绑定到 gRPC 请求 ctx（`Manager.Start(ctx, ...)`）——RPC 返回即 `cancel()`，进程立刻被杀，转推无法持续；必须 `context.Background()` 派生。
- 用 ffmpeg-go 的 `Compile(opts...)` 传进程组/资源 option——`Compile(options ...CompilationOption)` 的函数体只遍历全局 `GlobalCommandOptions`，**未使用传入的 options**；需访问进程组等 `SysProcAttr` 时走 `ffmpeg.GlobalCommandOptions = append(...)`。
- 把 `LogCompiledCommand`/`.Silent()` 当"仅本服务关闭"——它是包级全局变量（`LogCompiledCommand=true`），影响所有用 ffmpeg-go 的包（含 `common/mediax`）；需在 `buildFfmpegCmd` 用 `.Silent(true)` 关掉 std `log.Printf` 打印，改由本服务用 logx 记录。
- 用 `defer relayMgr.StopAll()` 代替 `proc.AddShutdownListener` 接入退场——defer 无法覆盖"信号 → 监听器"路径，且监听器异步不等待；必须注册 ShutdownListener 并 `defer waitForCalled()`。
- 服务停止后残留 ffmpeg —— 即"孤儿进程继续推流"。原因：ffmpeg 是独立子进程，`context.Background()` 的 ctx 永不取消，`defer s.Stop()` 不杀非己 child。用 `StopAll()` + `wg.Wait()` + ShutdownListener 治理。
- 自己做转码前先确认需求——SRS/Oryx 的 transcode 是全局单任务/静态配置，多路按需转码用本服务 FFmpeg 方案（`StreamRelay` 后续迭代加编码参数）或引入外部流媒体组件（LAL/ZLM），不要尝试改 Oryx transcode API 的配置达到"按流控制"。
- cluster 广播停止时非 owner 节点回 ack 表示"未找到"当作失败——应只让 owner 节点回 ack，其余保持沉默，否则首个错误 ack 会造成假失败。

## 依据

- Oryx 源码：`ossrs/oryx`（`platform/service.go`、`platform/dvr-local-disk.go`、`platform/callback.go`、`platform/utils.go`、`platform/trancode.go`）
- SRS 源码：`ossrs/srs 6.0release`（`srs_app_http_api.cpp`、`srs_app_statistic.cpp`、`srs_app_server.cpp`）
- 项目内：`common/oryxx/oryx.go`、`common/oryxx/srs.go`、`common/oryxx/oryxtype.go`、`app/oryxserver/oryxserver.proto`、`app/oryxserver/model/gormmodel/record.go`、`app/oryxgtw/internal/logic/hook/hooklogic.go`、`app/oryxgtw/internal/types/types.go`、`app/oryxserver/internal/relay/*`（StreamRelay 实现）
- 研究归档：`.trellis/tasks/08-24-oryx-hook-proxy/research/`、`.trellis/tasks/08-25-stream-relay/research/oryx-relay-transcode.md`
