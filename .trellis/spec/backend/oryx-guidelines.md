# Oryx 网关与 SRS API 代理（oryxgtw/oryxserver/oryxx）

## 适用范围

修改 `app/oryxgtw`（REST 回调网关）、`app/oryxserver`（gRPC 代理 + Relay Pull 中继拉流）、`common/oryxx`（SDK）时读取。

## 1. Scope / Trigger

- 本规范覆盖三层链路：Oryx 回调 → `oryxgtw`（HTTP 11004）→ gRPC → `oryxserver`（21016）→ Oryx HTTP API（`/terraform/v1/*` 与 `/api/v1/*`）。
- `oryxserver` 另有 **Relay Pull** 链路：业务 gRPC → 内部 FFmpeg 进程（拉外部源流转推固定 SRS 目标，动态启停，第一版仅 copy 转推；API 为 `StartRelayPull`/`StopRelayPull`）。
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

- gRPC：`OryxServer` 服务，平台级 17 个 RPC + `RecordList`/`RecordDelete`（本地落库）+ `RecordBeginHook`/`RecordEndHook`（⚠️ 内部钩子，仅 `oryxgtw` 调用）+ `StartRelayPull`/`StopRelayPull`（中继拉流）+ `StopRelayAndRecording`（停止中继并结束录制）。

### Relay Pull（FFmpeg 中继拉流）— 接口与进程契约

```protobuf
message StartRelayPullReq {
  string source_url = 1;   // 源流地址（RTMP/RTSP/HTTP-FLV/HLS 等 FFmpeg 支持的协议）
  string app = 2;          // 目标应用名，默认 "live"（RelayConfig.DefaultApp）
  string stream = 3;       // 目标流名，不填则自动生成 UUID（tool.SimpleUUID）
  uint64 max_duration_seconds = 6; // 0 不限制；正数为首次启动起的总秒数
}
message StartRelayPullRes {
  string relay_id = 1;     // 内部 UID：host_port/app/stream（如 127.0.0.1_1935/live/drone_xxx）
  string app = 2;          // 目标应用名（实际使用值）
  string stream = 3;       // 目标流名（自动生成时返回生成值）
  bool already_running = 4; // 启动前是否已有活跃租约
}
message StopRelayPullReq {
  string app = 1;          // 目标应用名
  string stream = 2;       // 目标流名；单固定目标服务下 (app, stream) 唯一确定一个中继
}
message StopRelayPullRes {}
```

- **relay_id 与内部身份（UID）**：响应 `relay_id = UID`，即 `host_port/app/stream`（如 `127.0.0.1_1935/live/drone_xxx`）。UID 是 relay 系统的唯一身份，用于：内存 `meta` map key、ffmpeg process ID、Redis state/lease/lock key 后缀、Sorted Set member、Asynq reconcile/stop payload。`:` 在 UID 中替换为 `_`。`relayURL` 仅用于 FFmpeg 输出地址，不是身份。`CanonicalUID(target)` 从任意 URL/normalized endpoint 生成 UID；`ParseTarget(uid)` 从 UID 提取 `app`/`stream`。

- **播放地址约定**（业务侧按协议自行拼接，Oryx 无获取播放地址接口）：
  `RTMP: rtmp://<host>:<rtmp_port>/{app}/{stream}`；`HTTP-FLV: http://<host>:<http_port>/{app}/{stream}.flv`；`HLS: http://<host>:<http_port>/{app}/{stream}.m3u8`；`WebRTC: webrtc://<host>:<rtc_port>/{app}/{stream}`。
- **任务管理**：`RelayRegistry` 是唯一的 relay 生命周期管理器（合并了原 `DistributedRelay`），拥有 relay metadata、UID 选择和 relay hooks；`common/ffmpegx.Manager` 是唯一通用进程生命周期 owner。`RelayRegistry.StartRelay(ctx, source, target, relayURL)` 内部从 `target` 生成 UID，登记 metadata，通过 `Manager.Start(uid, ...)` 注册 progress/stderr/exit hook；退出时先按 UID 指针身份清理，再执行业务 hook。`startLocalRelay(ctx, source, uid, relayURL)` 不再接受 target 参数，直接用 UID 作为进程 ID 和 metadata key。
- **退出与输出**：`Manager.Start` 重复 target 返回可 `errors.Is` 为 `ffmpegx.ErrProcessExists` 的错误，不替换旧进程；exit hook 接收 `ExitResult{ID, WaitErr, ContextErr}`，自然退出、cancel、Stop、deadline 均回调。relay 通过 `WithStdoutHandler` 逐行解析 FFmpeg `key=value`，仅完整报告的 `progress=continue` 续租；通过 `WithStderrHandler` 逐行处理 FFmpeg 日志。两个 pipe 必须并发消费，`ffmpegx` 不保存 progress 或 stderr 状态。
- **最大运行时长**：`max_duration_seconds=0` 时默认 1 天（`defaultRelayDuration = 24 * time.Hour`），不设无限；正数转换为从首次 Start 起的绝对 `deadline_at_unix` 并保存到 `RelayState`。Reconcile 使用剩余时间创建 deadline context，不重置完整时长；deadline 退出删除 state/lease 且不补拉，流异常且未到 deadline 才释放租约并入队补拉。
- **State TTL**：`SaveState` 动态计算 TTL = deadline 剩余时长 + 1 天兜底（`stateTTLOverhead`）；deadline=0 时 TTL = 默认时长 + 1 天。防止进程崩溃后 key 永久残留，同时不提前过期（2 天中继 → TTL ≈ 3 天）。
- **FFmpeg 命令**：`BuildRelayCmd(ctx, source, target) *exec.Cmd` 构造 `ffmpeg -progress pipe:1 -stats_period 5 -rw_timeout 10000000 -i <source> -c copy -f flv <relayURL>`，不返回 error，也不使用 `bytes.Buffer` 保留 stderr。`-rw_timeout 10000000`（10 秒 socket 读写超时）对所有网络协议（RTMP/RTSP/HTTP/HLS）生效，源流不存在时 ffmpeg 在 10 秒后自动退出。Start 配置 stdout/stderr handler 才由 Manager 创建和并发消费相应 pipe；无 handler 时 Manager 不碰对应流，也不解析命令参数。`progress=continue/end` 表示收到完整状态报告，不等同于业务流一定健康。
- **鉴权查询参数在 target 尾部**：`authQuery`（`?secret=xxx` / `?sign=md5(pushkey)`）由 `buildAuthQuery` 拼在**整个 target URL 后**（`target += "?" + authQuery`），最终命令为 `-f flv rtmp://<srs>/<app>/<stream>?secret=...`。SRS/Oryx 用 `?secret=`、WVP/ZLM 用 `?sign=`，authStyle 配置切换。
- **进程生命周期契约**（服务退场必须停掉 ffmpeg，否则孤儿进程继续推流）：
  - `Manager.Start(ctx, id, builder, options...)` 使用 `context.WithCancel` 派生 process context/cancel，不自动脱离父 context；Start 成功后 watcher 唯一调用一次 `Wait()`。父 context 取消、Stop、StopAll 和 deadline 都同步调用 exit hook；业务层通过 `ExitResult.ContextErr` 与自身 metadata 身份判断后续动作。重复 ID 返回 `ErrProcessExists`，不支持隐式 replacement。
  - `RelayRegistry.StopAll()` 只清理并停止自身登记的 relay target；不得调用共享 Manager 的全局 StopAll。
  - 接入 go-zero：wrap-up 先调用 `RelayRegistry.StopAll()` 清 metadata/停 relay，再调用 `FFmpegManager.StopAll()` 兜底停止未来截图、录制、转码等所有 FFmpeg 场景；必须 `defer` listener 返回的 `waitForCalled`。
  - `proc.SetTimeToForceQuit(c.GracePeriod)`（config 里 `GracePeriod time.Duration`，默认 10s）必须在 `s.Start()` 前调用。
- **容器编排兜底**：容器停止（`docker stop`/k8s 优雅终止 → PID 1 收到 SIGTERM → follow 上述优雅路径；`kill -9 容器` → 内核销毁整个 cgroup，ffmpeg 同死）。**无需** `PR_SET_PDEATHSIG`/进程组；macOS 本机开发无 PDEATHSIG，需手动 `kill`。
- **配置**：`DeployMode`（`standalone`/`cluster`，默认 standalone，对齐 ieccaller）；`RelayConfig{SrsRtmpAddr, DefaultApp, SecretKey, SecretValue}`；`MqttConfig`（可选，仅 cluster 用；cluster 且无 broker → `logx.Must` 快速失败）。
- **鉴权配置**：`RelayConfig.SecretKey`（默认 `secret`）和 `RelayConfig.SecretValue`（无默认值）；请求未指定时 fallback 到配置值。`buildAuthQuery(key, value)` 拼 `?key=value`，任一为空则不拼。最终 ffmpeg 推流地址 = `rtmp://<srs>/<app>/<stream>?<SecretKey>=<SecretValue>`。
- **Asynq 任务保留**：`EnqueueReconcile`/`EnqueueStop` 均设置 `asynq.Retention(7*24*time.Hour)`，完成后保留 7 天供排查。
- **分布式停止（cluster 可选）**：本地未命中 → 经 `common/mqttx/broadcast` 广播（prefix `oryx/server`，topic `oryx/server/broadcast`，reply `oryx/server/broadcast_reply/{instanceId}`，`BroadcastReply` 默认 TTL 10s）；仅**持有任务的节点**回 ack（成功/失败都回），其余节点返回 `broadcast.ErrSkipAck` 保持沉默（避免假成功）；忽略自环（`AckTopic == 本机 reply topic`）。payload 对齐 **ieccaller 模式**：发送端 `protojson.Marshal(StopRelayPullReq)`，executor 侧 `protojson.Unmarshal` 后本地 `StopPull`，返回 `protojson.Marshal(StopRelayPullRes)`。method 键**直接用 gRPC 生成常量** `oryxserver.OryxServer_StopRelayPull_FullMethodName`（不自建字符串常量，避免与生成物漂移）。**注意**：executor 内切勿调用带「未命中再广播」分支的 `StopRelayPullLogic`，否则集群消息风暴；只在 executor 内做本地动作（对齐 ieccaller 注入最小依赖而非 ServiceContext）。
- **UID + 并发安全**：UID（`host_port/app/stream`）是本地 process 和 metadata 唯一键。`RelayRegistry.lifecycleMu` 串行化同 UID 生命周期；启动已有 UID 是幂等 no-op，直接使用 `Manager.Start` 的业务调用遇到重复 ID 必须处理 `ErrProcessExists`，并由业务显式 Stop 后再 Start。Manager 和 RelayRegistry 清理时都校验对象指针身份，旧 watcher 不得删除后继。Manager 的 `RWMutex` 只保护 process map，builder、Start、cancel、Wait、日志和同步 callback 在锁外执行；不同 UID 不由全局生命周期锁串行化。
- **服务退场用 `AddWrapUpListener` 而非 `AddShutdownListener`**：oryxserver 自管 ffmpeg，停拉流属「停止产出」，应在收到信号第一时间（wrap-up）执行，拥有完整 `GracePeriod` 预算，且先于 grpc 排水（grpc 的 `GracefulStop` 在 shutdown 阶段）停掉流量产出；`main` 必须 `defer` 返回的 `waitForCalled`。
- **单节点（standalone）行为**：零 MQTT 依赖；`StopRelayPull` 未命中本地任务 → 明确错误「任务不在当前节点」，不 panic。

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

### Relay 分布式协调契约

#### CanonicalUID / NormalizeTarget / ParseTarget

```go
// CanonicalUID: "rtmp://127.0.0.1:1935/live/stream?secret=abc" → "127.0.0.1_1935/live/stream", nil
// CanonicalUID: "127.0.0.1_1935/live/stream" → "127.0.0.1_1935/live/stream", nil
// CanonicalUID: "rtmp://host/live" → "", error (path 必须两段)
func CanonicalUID(target string) (string, error)

// NormalizeTarget: "rtmp://host:1935/live/stream?secret=abc" → "host:1935/live/stream"
func NormalizeTarget(target string) string

// ParseTarget: 从 UID 或 normalized key 解析 app 和 stream，严格两段校验
func ParseTarget(target string) (app, stream string, err error)
```

- `CanonicalUID` 是 relay 系统的唯一身份生成函数，所有入口必须调用。
- `ParseTarget` 校验：path 必须恰好 `/{app}/{stream}` 两段，多段（`a/b/c`）、空 app/stream、app 或 stream 含 `/` 均报错。
- UID = `host_port/app/stream`（`:` → `_`），用于 Redis key、内存 map、ffmpeg ID、Asynq payload。
- `NormalizeTarget` 保留为中间步骤（`CanonicalUID` 内部调用），不直接用于身份。

#### 分布式锁 + 广播顺序

```go
// 正确顺序：拿锁 → 操作 → 释放锁 → 广播
lock, ok, _ := store.Lock(ctx, target)
// ... 删除状态/租约、停止本地进程 ...
lock.Release()           // 必须用 Release()，不用 ReleaseCtx(ctx)
// ... 然后才广播 ...
```

- **锁释放必须在广播前**——广播接收方需要拿同一把锁执行本地操作，发送方持有锁会导致接收方 `lock held, skip`。
- **锁释放必须用 `lock.Release()`**——`ReleaseCtx(ctx)` 在请求 ctx 超时后会失败，锁只能等 TTL 过期。`Release()` 内部用 `context.Background()`。

#### Asynq TaskID 策略

| 任务类型 | TaskID | 原因 |
|---------|--------|------|
| `EnqueueReconcile` | 自动生成 | 补拉可能需要多次重试 |
| `EnqueueStop` | 自动生成 | 补停可能需要多次重试 |

- 幂等去重任务（如 Reconcile）用固定 TaskID 防重复入队；需要重试的任务用自动生成 TaskID，允许重复入队。
- 误用固定 TaskID 会导致重试入队被 `ErrDuplicateTask` 拒绝。

#### StopRelayPull 流程

```
gRPC StopRelayPull
  → stopRelayPullOnce（核心逻辑）
      → 检查存在（本地 HasTarget + Redis GetState）
      → 委托 StopRelay（锁 → 删状态 → 停本地）
      → 本地未命中 + cluster → 广播
  → 失败 → EnqueueStop（Asynq 补停）

Asynq StopHandler
  → StopRelayPullFromAsynq（调同一核心逻辑，不入队）
  → 失败 → Asynq 重试
```

- `StopRelayPullLogic.stopRelayPullOnce` 是核心逻辑（不入队），`StopRelayPull`（gRPC）和 `StopRelayPullFromAsynq`（Asynq）分别包装。
- `stopRelayPullOnce` 内部委托 `RelayRegistry.StopRelay` 完成锁+清理+停止，不再自己获取分布式锁（避免双重锁死锁）。
- MQTT executor 只做本地 `StopRelayByAppStream`，不调用 `StopRelayPullLogic`（避免集群消息风暴）。

#### StopRelayAndRecording 流程

```protobuf
message StopRelayAndRecordingReq {
  string app = 1 [json_name = "app"];
  string stream = 2 [json_name = "stream"];
}
message StopRelayAndRecordingRes {}
```

```go
func (l *StopRelayAndRecordingLogic) StopRelayAndRecording(in *oryxserver.StopRelayAndRecordingReq) (*oryxserver.StopRelayAndRecordingRes, error) {
    // 1. 复用 StopRelayPullLogic 停止中继流
    stopLogic := NewStopRelayPullLogic(l.ctx, l.svcCtx)
    stopLogic.StopRelayPull(&oryxserver.StopRelayPullReq{App: app, Stream: stream})

    // 2. 查询录制中的记录（status=1）
    var records []gormmodel.Record
    db.Where("app = ? AND stream = ? AND status = ?", app, stream, gormmodel.RecordStatusRecording).Find(&records)

    // 3. 逐个结束录制（容错，单个失败不影响其他）
    for _, record := range records {
        if err := l.svcCtx.OryxClient.RecordEnd(l.ctx, record.UUID); err != nil {
            // "no record task" 错误视为幂等成功
            if oe, ok := oryxx.IsOryxError(err); ok && oe.Code == 500 &&
                strings.Contains(oe.Message, "no record task") {
                continue
            }
            l.Errorf("RecordEnd failed: uuid=%s, err=%v", record.UUID, err)
        }
    }
}
```

- **业务场景**：停止中继流时，同时结束关联的录制任务
- **幂等性**：
  - `StopRelayPull` 已支持幂等（重复调用安全）
  - `RecordEnd` 对 "no record task" 错误视为幂等成功（Oryx 返回 500 + "no record task for uuid=..."）
- **容错**：单个录制停止失败不影响其他录制的停止
- **复用**：内部复用 `StopRelayPullLogic`，不重复实现分布式停止逻辑

#### 命名规范

```go
// 组件（已合并：RelayRegistry + DistributedRelay → 单一 RelayRegistry）
type RelayRegistry struct { ... }      // 本地进程 + metadata + 分布式协调（Redis state/lease + Asynq）

// RelayRegistry 方法
func (r *RelayRegistry) StartRelay(ctx, source, target, relayURL, maxDuration) (string, error)
func (r *RelayRegistry) StopRelay(ctx, target) bool
func (r *RelayRegistry) StopRelayByAppStream(app, stream) bool
func (r *RelayRegistry) StopAll()
func (r *RelayRegistry) HasTarget(target) bool
func (r *RelayRegistry) Reconcile(ctx, target, source, retryCount) error
func (r *RelayRegistry) EnqueueReconcile(ctx, target, source, retryCount) error
func (r *RelayRegistry) EnqueueStop(ctx, target, app, stream) error

// ServiceContext
svcCtx.RelayRegistry   // *relay.RelayRegistry（合并后唯一字段，无 DistRelay）
```

#### 内部回调（可测试覆盖）

```go
// onProgressFunc: ffmpeg stdout progress=continue 时触发续租
// onProcessExitFunc: ffmpeg 进程退出时触发清理/补拉
// 测试中可覆盖为 mock 函数
r.onProgressFunc = r.defaultOnProgress
r.onProcessExitFunc = r.defaultOnProcessExit
```

#### ffmpegx.WithExitHandler 签名

```go
// 回调签名：func(id string, result ExitResult)
// id = process ID（即 target），result 只含 WaitErr + ContextErr（无 ID 字段）
ffmpegx.WithExitHandler(func(id string, result ffmpegx.ExitResult) { ... })
```

#### StartRelay 状态写入顺序

```go
// 正确：先检查租约，再写状态
lock → HasLease → [租约存在则报错] → 检查 source 冲突 → SaveState → TryClaim → StartRelay

// 错误：先写状态，再检查租约（会覆盖运行中 relay 的状态）
lock → SaveState → HasLease → TryClaim → StartRelay
```

- **HasLease 必须在 SaveState 之前**——有租约说明已有节点在跑，直接返回错误，业务侧自行决定停止。
- **source 冲突检查**——state 已存在且 source 不同时返回错误，防止覆盖运行中 relay 的源地址。
- 覆盖运行中 relay 的状态会导致 Reconcile 读到错误的 source/relayURL，重启后参数不对。

#### RelayState 结构

```go
type RelayState struct {
    UID              string `json:"uid"`                          // 唯一标识：host_port/app/stream
    Host             string `json:"host,omitempty"`               // SRS 主机地址
    Port             int    `json:"port,omitempty"`               // SRS RTMP 端口
    App              string `json:"app,omitempty"`                // 目标应用名
    Stream           string `json:"stream,omitempty"`             // 目标流名
    Source           string `json:"source"`                       // 源流地址
    RelayURL         string `json:"relay_url"`                    // 完整推流地址（含鉴权参数，ffmpeg 用）
    DeadlineAtUnix   int64  `json:"deadline_at_unix,omitempty"`
    DeadlineAtStr    string `json:"deadline_at_str,omitempty"`    // yyyy-MM-dd HH:mm:ss
    CreatedAtStr     string `json:"created_at_str,omitempty"`     // yyyy-MM-dd HH:mm:ss
    RetryCount       int    `json:"retry_count,omitempty"`        // 当前补拉重试次数（仅供运维观察）
    PendingReconcile bool   `json:"pending_reconcile,omitempty"`  // 防扫描器重复入队
}

type ReconcilePayload struct {
    UID        string `json:"uid"`
    Source     string `json:"source,omitempty"`
    RetryCount int    `json:"retry_count,omitempty"`
}

type StopPayload struct {
    UID    string `json:"uid"`
    App    string `json:"app"`
    Stream string `json:"stream"`
}
```

- `DeadlineAtStr` 为可视化字段，用 `carbonx.FormatDateTime(time.Unix(deadline, 0))` 填充。
- `deadline_at_unix=0` 表示不限时长，此时 `deadline_at_str` 为空。
- `RetryCount` 由 `releaseAndRetry` 递增，超过 `maxReconcileRetries`(10) 则清理 state/lease。
- `ReconcilePayload.SourceURL` 用于 source 校验：task 的 source 与 state 不一致时跳过（说明已被新请求覆盖）。
- `PendingReconcile` 防扫描器重复入队：`releaseAndRetry` 和扫描器入队前置 true，`Reconcile` 成功后清 false，`cleanupTarget` 删 state 自然清除。

#### Relay 索引（Sorted Set）

```
relay:registry (Sorted Set)  // score = 最后更新时间（Unix 秒），member = hashKey(target)
```

- **启动 relay**：`ZADD relay:registry <now> hashKey(target)`（AddToRegistry，幂等）
- **进度回调（每 5s）**：`ZADD relay:registry <now> hashKey(target)`（RenewRegistry，与续租同步）
- **停止 relay**：`ZREM relay:registry hashKey(target)`（RemoveFromRegistry）
- **扫描器（每 30s）**：`ZRANGEBYSCORE relay:registry 0 <now-60>` 取过期条目（O(log N + M)）
  - 有 lease → 跳过（刷新 score）
  - 无 lease + pending=true → 通常跳过；**但如果 score 超过 `PendingStaleThreshold`（5min）说明 Asynq 任务可能丢失** → 重置 `pending=false` + SaveState → 下面的入队逻辑重新 EnqueueReconcile
  - 无 lease + !pending → 置 pending=true → EnqueueReconcile
  - `ScanStale` 返回 `[]*StaleEntry`（`State` + `Score`），Score = Sorted Set score（Unix 秒，最后 RenewRegistry 时间）
- **state 删除时**：`cleanupTarget` 同时 `ZREM` 清理索引
- **过期条目清理**：`ScanStale` 读 state 时发现已删除（redis.Nil）→ 自动 `ZREM`

## 4. Validation & Error Matrix

| 条件 | 错误行为 |
| --- | --- |
| Oryx 非 200 响应 | SDK 返回 `*OryxError{Code: HTTP状态码, Message: 文本}`（JSON 时取 `{code,data}`） |
| Oryx/SRS 200 且 `code != 0` | `*OryxError{Code: N, Message...}`（SRS Message 固定 `"SRS API 业务错误"`） |
| 网络/JSON 解析失败 | 普通 `error`（带 cause） |
| 以上任一 error 上抛到 RPC 层 | gRPC error `tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 Oryx API 失败")`（SRS 用同码，message "调用 SRS API 失败"） |
| `postProcess != "post-cp-file"` | Oryx 返回 "invalid post process"（默认 500） |
| `record/end` 指向非 live 任务 / `record/remove` uuid 不存在 | Oryx 500 错误（正常业务错误，不可重试语义按调用方处理） |
| `StartRelayPull` 缺 source_url | gRPC error `extproto.Code__1_01_PARAM_MISSING` |
| `StartRelayPull` app/stream 含 `/` | gRPC error `extproto.Code__1_01_PARAM_MISSING` |
| `StartRelayPull` target 已有租约 | 返回 error `target already has active lease`，业务侧自行决定停止 |
| `StartRelayPull` target 已存在但 source 不同 | 返回 error `target already exists with different source` |
| `Reconcile` task source 与 state source 不一致 | 释放租约 + skip（return nil），**不改 state**（避免覆盖新请求的状态） |
| `StopRelayPull` 缺 app/stream | gRPC error `extproto.Code__1_01_PARAM_MISSING` |
| `StopRelayPull` 本地未命中且非 cluster | 返回成功（幂等）；cluster 模式广播停止 |
| `StopRelayAndRecording` `RecordEnd` 返回 "no record task" | 视为幂等成功，继续处理下一个 UUID |
| `CanonicalUID` 无效 target（无 path / 多段 / 空 app/stream） | `StartRelay`/`StopRelayPull` 返回 error |
| 分布式锁获取失败 | `Start`/`Stop` 返回 error |
| 分布式锁被其他节点持有 | `Start`/`Stop` 跳过（返回成功/nil） |
| FFmpeg 拉流失败/流断开 | watcher 清理 process 与 relay metadata，调用 exit hook 释放租约并入队补拉；RPC 启动成功后不再同步返回运行期错误 |
| 同一 `(app,stream)` 二次推流（并发） | 新任务 ffmpeg 立即退出：`Error opening output ... Input/output error` + `exit status 251`（SRS 拒绝新连接，见"SRS 重复推流行为"）；旧任务存活 |
| 服务退场（SIGTERM/SIGINT） | `AddWrapUpListener` 依次执行 `RelayRegistry.StopAll()` 与 `FFmpegManager.StopAll()`，有界等待子进程 Wait 完成后退场 |
| 服务被 `kill -9`/崩溃 | 无 cleanup 可跑，ffmpeg 成孤儿继续推流；容器部署由 cgroup 兜底（进程随容器死）；裸进程需 PDEATHSIG/编排兜底。**RegistryScanner 兜底**：Sorted Set 索引 score 停更 → 60s 后被 `ZRANGEBYSCORE` 扫到 → 无 lease + !pending → 入队 Reconcile 补偿 |

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

- Relay Pull 进程退场——注册 WrapUpListener 并守护等待，而非裸 defer。同时停止 FFmpeg 包的 std log 打印（`LogCompiledCommand` 是包级全局，需 `.Silent(true)`）：

```go
// Correct: stop-all 接入 go-zero 优雅关闭（wrap-up 先行停产出），且保证等待全部退出
waitRelayStop := proc.AddWrapUpListener(func() {
	ctx.RelayRegistry.StopAll()
	ctx.FFmpegManager.StopAll()
})
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
- 把长期 relay process ctx 直接绑定到会随 RPC 返回取消的请求 ctx——进程会被立即停止。`RelayRegistry` 应使用 `context.WithoutCancel` 保留 trace/value 并明确脱离请求取消，再由 `ffmpegx.Manager` 派生并拥有 process cancel；不要改用 `context.Background()` 丢失上下文值。
- 用 ffmpeg-go 的 `Compile(opts...)` 传进程组/资源 option——`Compile(options ...CompilationOption)` 的函数体只遍历全局 `GlobalCommandOptions`，**未使用传入的 options**；需访问进程组等 `SysProcAttr` 时走 `ffmpeg.GlobalCommandOptions = append(...)`。
- 把 `LogCompiledCommand`/`.Silent()` 当"仅本服务关闭"——它是包级全局变量（`LogCompiledCommand=true`），影响所有用 ffmpeg-go 的包（含 `common/mediax`）；需在 `buildFfmpegCmd` 用 `.Silent(true)` 关掉 std `log.Printf` 打印，改由本服务用 logx 记录。
- 用裸 defer 代替 `proc.AddWrapUpListener` 接入退场——defer 无法覆盖"信号 → 监听器"路径，且监听器异步不等待；必须注册 WrapUpListener 并 `defer waitForCalled()`。
- 服务停止后残留 ffmpeg —— 即"孤儿进程继续推流"。原因：ffmpeg 是独立子进程，`context.Background()` 的 ctx 永不取消，`defer s.Stop()` 不杀非己 child。用 `StopAll()` + `wg.Wait()` + WrapUpListener 治理。
- 自己做转码前先确认需求——SRS/Oryx 的 transcode 是全局单任务/静态配置，多路按需转码用本服务 FFmpeg 方案（后续迭代加编码参数）或引入外部流媒体组件（LAL/ZLM），不要尝试改 Oryx transcode API 的配置达到"按流控制"。
- cluster 广播停止时非 owner 节点回 ack 表示"未找到"当作失败——应只让 owner 节点回 ack，其余保持沉默，否则首个错误 ack 会造成假失败。
- **分布式锁释放用 `lock.Release()` 而非 `lock.ReleaseCtx(ctx)`**——请求 ctx 超时后 `ReleaseCtx(ctx)` 会失败，锁只能等 TTL 过期。`Release()` 内部用 `context.Background()`，永远安全。trigger 和 oryxserver 均适用。
- **广播前必须释放分布式锁**——广播接收方需要拿锁才能执行本地操作（如 StopPull），发送方持有锁会导致接收方拿不到锁而跳过。正确顺序：拿锁 → 删状态 → 停本地 → **释放锁** → 广播。
- **扫描器 pending 最终补偿**——`PendingReconcile=true` 的条目通常跳过，但 `ScanStale` 返回的 `StaleEntry.Score`（Sorted Set score，Unix 秒）如果超过 `PendingStaleThreshold`（5min），说明 Asynq 任务可能丢失（worker 崩溃、任务过期）。此时重置 `pending=false` + SaveState → 入队逻辑重新 EnqueueReconcile。防止 relay 永久卡死在 `pending=true` 状态。
- **Sorted Set 索引必须与 state 同步维护**——启动时 `AddToRegistry`、停止时 `RemoveFromRegistry`、progress 时 `RenewRegistry`、cleanupTarget 时 `RemoveFromRegistry`。漏掉任何一个都会导致扫描器误判或索引残留。
- **Reconcile source mismatch 不改 state**——source 不一致说明已被新 StartRelay 覆盖，只释放租约 + skip。不清 pending、不 SaveState，避免覆盖新请求的状态。新请求的 state 由新请求自己的 reconcile 任务或 scanner 处理。
- **`-rw_timeout` 而非 `-timeout`**——ffmpeg 的 `-timeout` 对 RTMP 不生效，`-rw_timeout`（socket 读写超时）对所有网络协议有效。源流不存在时 ffmpeg 会无限等待，必须加此参数。
- **Asynq TaskID 策略**：幂等去重任务（如 Reconcile）用固定 TaskID 防重复入队；需要重试的任务（如 Stop 广播）用 Asynq 自动生成 TaskID，允许重复入队。误用固定 TaskID 会导致重试入队被 `ErrDuplicateTask` 拒绝。
- **补偿任务类型与业务场景必须匹配**——补停用 `RelayStopTask`（广播停止），补拉用 `RelayReconcileTask`（读状态+启动）。混用会导致语义错误（如广播超时后入队 reconcile，但 state 已删，reconcile 变空操作）。
- **MQTT 广播超时不能假设目标已停止**——超时可能是 MQTT 本身的问题（网络、broker），应重试而非放弃。
- **StopRelayAndRecording 中 RecordEnd 失败应继续处理**——单个录制停止失败不应阻塞其他录制的停止，"no record task" 错误（Oryx 500）视为幂等成功。
- **Config 字段名必须与 YAML key 一致**——struct 的 `json` tag 决定 YAML 解析的 key。`DefaultSecretKey`/`DefaultSecretValue` 配 YAML 的 `DefaultSecret` 会导致值为空（go-zero 静默忽略不匹配的 key）。正确做法：struct 字段名 = YAML key，如 `SecretKey`/`SecretValue`。
- **Asynq 任务必须设 Retention**——不设 Retention 的任务完成后立即删除，无法排查。统一用 `asynq.Retention(7*24*time.Hour)` 保留 7 天。
- **handleExit 必须校验 metadata 指针身份**——用 `deleteMeta(uid)` 只检查 key 存在，旧进程退出会误删新进程的 metadata。必须 `r.meta[uid] != md` 校验指针一致后才删除。
- **用 UID 不用 target**——relay 系统中不存在 "target" 概念。所有身份操作（meta map key、ffmpeg ID、Redis key、Asynq payload）统一使用 `CanonicalUID` 生成的 `host_port/app/stream`。`StartRelay` 接受外部输入 target URL，内部转 UID；`startLocalRelay` 直接接受 UID。
- **StartRelay 租约存在应报错而非强制重启**——强制停旧进程再重启会导致进程错乱（旧进程的 exit 回调清理新进程的 metadata）。应返回 error，业务侧自行决定停止。
- **Reconcile 必须校验 source 一致性**——task 的 source 与 state 的 source 不一致说明已被新请求覆盖，应 skip 避免重复启动。
- **handleStderr 应白名单过滤**——ffmpeg 所有非媒体数据写 stderr（banner、metadata、stream info、错误）。应只保留含 error/warning/cannot/failed/fatal/denied/timeout 的行。
- **ffmpegx.ExitResult 无 ID 字段**——`WithExitHandler` 签名为 `func(id string, result ExitResult)`，id 作为独立参数传入，ExitResult 只含 WaitErr + ContextErr。
- **日志前缀规范**——`[relay]`（relay 生命周期）、`[asynq-task]`（Asynq 任务消费/入队）、`[ffmpegx]`（进程管理）、`[registry-scanner]`（孤儿扫描）、`[node-reporter]`（节点上报）。不能用 `[asynq]`，与基础包 `asynqx` 前缀冲突。日志消息统一用中文，便于中文用户排查。

## 依据

- Oryx 源码：`ossrs/oryx`（`platform/service.go`、`platform/dvr-local-disk.go`、`platform/callback.go`、`platform/utils.go`、`platform/trancode.go`）
- SRS 源码：`ossrs/srs 6.0release`（`srs_app_http_api.cpp`、`srs_app_statistic.cpp`、`srs_app_server.cpp`）
- 项目内：`common/oryxx/oryx.go`、`common/oryxx/srs.go`、`common/oryxx/oryxtype.go`、`common/ffmpegx/process.go`、`app/oryxserver/oryxserver.proto`、`app/oryxserver/internal/relay/registry.go`、`app/oryxserver/internal/relay/state.go`
- 节点上报：`app/oryxserver/internal/cron/node_reporter.go`（每秒 SADD `oryx:nodes` + HSET `oryx:node:{短ID}` + EXPIRE 5s；短ID = nodeID 去掉 `oryx-node-` 前缀，避免 `oryx:node:oryx-node-xxx` 重复前缀）
- 孤儿扫描：`app/oryxserver/internal/cron/registry_scanner.go`（每 30s ZRANGEBYSCORE 扫描 Sorted Set，per-UID 分布式锁内重新读取 state，先入队再更新 pending）
- UID 工具：`common/tool/jitter.go`（`JitterDelay(base, max)`，补拉随机延迟）
- 孤儿扫描：`app/oryxserver/internal/cron/registry_scanner.go`（每 30s ZRANGEBYSCORE 扫描 Sorted Set 索引，发现无 lease 的 relay → EnqueueReconcile 补偿）
- 研究归档：`.trellis/tasks/08-24-oryx-hook-proxy/research/`、`.trellis/tasks/08-25-stream-relay/research/oryx-relay-transcode.md`
