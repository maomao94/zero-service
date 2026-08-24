# Research: Oryx 录制中 nn / duration / size 的计算与聚合链路（on_hls → record/files → 回调）

- **Query**: Oryx 录制任务中 `nn`(TS 分片数)、`duration`(累计秒)、`size`(累计字节) 三个值的计算与聚合链路；on_hls 消息来源、聚合位置、record/files 数据源、on_record_end 时机、回调透传能力
- **Scope**: external（GitHub 源码调研：ossrs/oryx `release/5.14`，对照 `main`）
- **Date**: 2026-08-25
- **基线**: 用户部署 Oryx v5 线（SRS 5.0.213，对应 Oryx `release/5.14` 分支，见既有调研 oryx-relay-transcode.md）；`main`（v6 线）本议题行为与 5.14 完全一致（已逐行核对，见文末差异说明）

## 结论速览（TL;DR）

| # | 问题 | 结论 |
|---|---|---|
| 1 | on_hls 从哪来 | SRS `http_hooks.on_hls` → `http://<oryx>:2022/terraform/v1/hooks/srs/hls`（**不是** `/hooks/hls`）；Oryx 的 `handleOnHls` 分发到 recordWorker/vodWorker/dvrWorker/transcript。 |
| 2 | nn/duration/size 如何聚合 | 每个 TS 分片一条 `TsFile{seqno,duration,size}` 追加到 `M3u8VoDArtifact.Files[]`，**整体序列化后存 Redis**（`SRS_RECORD_M3U8_ARTIFACT`，key=uuid）。`nn` 为 `len(Files)`（增量存），`duration`/`size` **不增量存，查询时对 Files[] 求和**。 |
| 3 | record/files 数据源 | 全部来自 Redis `SRS_RECORD_M3U8_ARTIFACT`（HScan count=100），实时求和，与内存无关。 |
| 4 | on_record_end 时机 | 端回调在**所有已消费 TS 聚合完成、mp4 转换完成之后**同步触发（`defer`），值已完整；但存在异步窗口：还在 `msgs` channel(1024) 中/拷贝中的最后一个 on_hls 消息可能被丢弃（Timeout/主动 end/流突然断开）。 |
| 5 | 回调透传 | **不能**。`callback.go` 硬编码字段白名单（request_id/action/opaque/vhost/app/stream/uuid/artifact_code/artifact_path/artifact_url），**无 nn/duration/size**。唯一可靠途径：收 end 回调后轮询 `POST /terraform/v1/hooks/record/files`（或直读 Redis `SRS_RECORD_M3U8_ARTIFACT`）。 |

## 1. on_hls 消息从哪来（SRS → Oryx）

### SRS 配置指向

`platform/containers/conf/srs.release.conf`（vhost `__defaultVhost__`，L76-83）：

```
http_hooks {
    enabled         on;
    on_publish      http://localhost:2022/terraform/v1/hooks/srs/verify;
    on_unpublish    http://localhost:2022/terraform/v1/hooks/srs/verify;
    on_play         http://localhost:2022/terraform/v1/hooks/srs/verify;
    on_stop         http://localhost:2022/terraform/v1/hooks/srs/verify;
    on_hls          http://localhost:2022/terraform/v1/hooks/srs/hls;
}
```

- on_hls 指向 Oryx 的路径是 **`/terraform/v1/hooks/srs/hls`**（`release/5.14` 与 `main` 相同）。
- 生产可被 `include containers/data/config/srs.vhost.conf` 覆盖，但官方默认即上述路径。

### Oryx 接收与分发

`platform/srs-hooks.go` `handleOnHls`（L709-791）：

- L712：`ep := "/terraform/v1/hooks/srs/hls"`，POST body 直接 `json.Unmarshal` 成 `SrsOnHlsMessage`（L721-727），校验 `action == "on_hls"`。
- L733-741：若 `SRS_RECORD_PATTERNS` hash 的 `all == "true"`（即录制开关开启）→ `recordWorker.OnHlsTsMessage(ctx, &msg)`（**这里就是日志 `on_hls ok, ...` 的来源**）。
- L744-769：按开关分别再派发给 `dvrWorker` / `vodWorker` / `transcriptWorker`。

### on_hls 消息字段（SrsOnHlsMessage，platform/utils.go L1079-1110）

```go
type SrsOnHlsMessage struct {
    Action   SrsAction `json:"action,omitempty"`   // 固定 on_hls
    File     string    `json:"file,omitempty"`     // ts 文件路径
    Duration float64   `json:"duration,omitempty"` // 秒，SRS 生成
    M3u8URL  string    `json:"m3u8_url,omitempty"`
    SeqNo    uint64    `json:"seq_no,omitempty"`   // 注意 json tag 是 seq_no（下划线）
    Vhost/App/Stream string
    URL      string    `json:"url,omitempty"`      // ts url
}
```

- **SRS on_hls 回调不带 `size`**——size 由 Oryx 自己 `os.Stat` 得到（见下）。这与背景证据里日志 `record consume msg ... duration=<秒>, seqno=<N> ...` 字段一致。

## 2. nn/duration/size 如何聚合

### 入站：OnHlsTsMessage（platform/dvr-local-disk.go L539-574）

```go
// Copy the ts file to temporary cache dir.
tsid := uuid.NewString()
tsfile := path.Join("record", fmt.Sprintf("%v.ts", tsid))
cp := exec.CommandContext(ctx, "cp", "-f", msg.File, tsfile)   // L546
stats, _ := os.Stat(msg.File)                                  // L551  注意：stat 的是 SRS 原文件
tsFile := &TsFile{TsID: tsid, URL: msg.URL, SeqNo: msg.SeqNo, Duration: msg.Duration,
    Size: uint64(stats.Size()), File: tsfile}                  // L557-564  ← size 唯一来源
go func() { select { case v.msgs <- &SrsOnHlsObject{Msg: msg, TsFile: tsFile}: } }()  // L567-572 异步入队
```

- `size` = `os.Stat(msg.File)`（SRS ts 文件实际字节数），`duration`/`seqno` 来自 SRS on_hls。
- **异步**：入队走独立 goroutine + 容量 1024 的 channel（`RecordWorker.msgs`，L49），HTTP 处理函数立即返回。

### 聚合：buildM3u8Object 单消费者 goroutine（L636-721）

- `v.streams.LoadOrStore(msg.Msg.M3u8URL, &RecordM3u8Stream{UUID: uuid.NewString()})` —— **以 m3u8_url 为键建立录制任务**，uuid 是任务 id（L671-675）。
- `m3u8LocalObj.addMessage(ctx, msg)`（L685）→ append 到 `Messages[]`，`NN = len(Messages)`（L817-824）。
- `saveObject`（L688）→ 整个 `RecordM3u8Stream`（含 Messages）json 序列化后写 **Redis `SRS_RECORD_M3U8_WORKING`**（key=m3u8_url，L771-781）—— 持久化、可重启恢复（Start L606-633 会 HGetAll 重新加载）。
- fresh 对象起 `Run(ctx)` goroutine（L693-701）。

### 消费：Run 循环（L912-996）＋ serveMessage（L998-1027）

每 300ms 一轮（L989-992）：`pfn` 中
1. `copyMessages`（L937）——在有锁下拷贝 `v.Messages`；
2. 对每条 `serveMessage`（L939）：`defer v.removeMessage(msg)`（L1000）→ `os.Rename` 临时 ts 到 `record/{uuid}/{tsid}.ts`（L1015）→ **`updateArtifact`**（L1020）：
   ```go
   artifact.Files = append(artifact.Files, msg.TsFile)   // L803  ← 核心累加
   artifact.NN = len(artifact.Files)                      // L804  ← nn 在此维护（增量）
   artifact.Update = time.Now().Format(time.RFC3339)
   ```
   → `saveArtifact`（L1021）→ **写 Redis `SRS_RECORD_M3U8_ARTIFACT`，key=uuid**（L783-793）；
3. 打日志 `logger.Tf(ctx, "record consume msg %v", msg.String())`（L1025）—— 与证据中日志完全一致；
4. `len(v.Messages) > 0` → 继续下一轮；为空 → 检查过期（L957-959）。

### 累计值究竟存哪？

- **内存**：`RecordM3u8Stream.Messages []*SrsOnHlsObject`（L744，Recording 中的待消费队列）＋ `recordWorker.streams sync.Map`（L44）；`artifact` 保存在 stream 结构里（L749）。
- **Redis（持久、权威）**：
  - `SRS_RECORD_M3U8_WORKING`（key=m3u8_url，value=RecordM3u8Stream，含 Messages）—— 进行中的任务状态；
  - `SRS_RECORD_M3U8_ARTIFACT`（key=uuid，value=`M3u8VoDArtifact`，含 `Files []*TsFile`）—— **每个分片带 seqno/duration/size 的完整列表**（utils.go L1016）。
- **结论**：`nn` 是增量存储（`artifact.NN = len(Files)`，双写于 RecordM3u8Stream.NN 和 artifact.NN）；`duration`、`size` **没有聚合字段**，任何时候都是对 `artifact.Files[]` 求和（record/files 每次实时算）；Redis 是唯一数据源，进程重启可恢复。

## 3. record/files 如何产出这些值（platform/dvr-local-disk.go L335-391）

```go
keys, cursor, err := rdb.HScan(ctx, SRS_RECORD_M3U8_ARTIFACT, 0, "*", 100).Result()  // L353 单次 100 条
...
for _, file := range metadata.Files {   // L367-370  实时求和
    duration += file.Duration
    size    += file.Size
}
files = append(files, map[string]interface{}{
    "uuid": metadata.UUID, "vhost": ..., "app": ..., "stream": ...,
    "progress": metadata.Processing,   // M3u8VoDArtifact json tag 就是 "progress"（原代码 typo，L1012）
    "update":   metadata.Update,
    "nn":       len(metadata.Files),   // L379
    "duration": duration,              // L380
    "size":     size,                  // L381
})
```

要点：
- **数据源**：全量来自 Redis `SRS_RECORD_M3U8_ARTIFACT`（HScan 游标式扫描，但**单次只支持 count=100 且响应不返回 cursor，无分页续扫机制**——超过 100 条录制记录时一次调用看不全，且顺序不确定）。
- **包含进行中与已完成**：进行中任务 `progress=true`（Processing 字段），完成后 `progress=false`（`finishArtifact` L809-815 置 Processing=false 并保存）；条目只有在 `/terraform/v1/hooks/record/remove`（L226-294，HDel）后消失。
- 鉴权：`SRS_PLATFORM_SECRET` token（L348-351），请求体 `{token}`。
- 返回元素字段与背景证据 `uuid/vhost/app/stream/progress/update/nn/duration/size` 完全吻合。

## 4. on_record_end 的时机与完整性

### 触发链路（Run → defer → callbackEnd）

```go
func (v *RecordM3u8Stream) Run(...) {
    message, _ := v.callbackBegin(ctx)                      // L918  on_record_begin（第一次有消息时）
    defer func() {
        ctx := parentCtx                                    // 不用 task ctx，task ctx 已被 cancel
        v.callbackEnd(ctx, message)                         // L929  结束回调（on_record_end）
    }()
    ...pfn 每 300ms...
    if err := v.finishM3u8(ctx); err != nil { ... }        // L962  写 index.m3u8 + ffmpeg 转 index.mp4（同步，耗时！）
    v.finishArtifact(ctx, v.artifact)                       // L1056 Processing=false
    r0 := v.saveArtifact(ctx, v.artifact)                   // L1057  ← 最终值已落 Redis
    ...
    cancel()                                                // L973  → 循环退出 → defer callbackEnd
}
```

- on_record_begin（callbackBegin L1091-1103）：用 `copyMessages()[0]`，artifact 为 nil → 回调只带 UUID/vhost/app/stream（无 artifact 字段）。
- **on_record_end 一定在 mp4 转换完成后才触发**（finishM3u8 L1047 `ffmpeg -i index.m3u8 -c copy -y index.mp4` 同步阻塞），且 `saveArtifact` 已把最终 Files[]/NN/Processing=false 写入 Redis，然后才 POST 回调。
- 结束条件：`expired()`（L848-876）→ 主动 `/record/end` 置 `Expired=true`（L296-333）立即过期；或 30s（development）/300s（生产，`NODE_ENV != development`）无新分片超时。

### 异步窗口（不保证 100% 完整）

- 入队是异步的（L567-572 goroutine），`msgs` channel 容量 1024；
- 消费循环每 300ms 周转、pfn 内每条消息要 `os.Rename` + Redis 写；
- Run 的结束判定只看 `v.Messages` 已空 + expired —— **不等待 `msgs` channel 排空**：
  - 场景 A：`/record/end` 被调用时，最后若干 on_hls 消息还在 channel / 还在 OnHlsTsMessage 拷贝中 → 它们永远不会进入 Messages → 不进 artifact；
  - 场景 B：超时结束（300s）时，后续分片直接丢弃；
  - 代码明确注释 `Do final cleanup, because new messages might arrive while converting to mp4, which takes a long time.`（L1061-1066，删掉残余 ts）。
- **结论**：常规时序（publish → 稳定录制 → unpublish → 超时/主动 end）下 on_record_end 时 nn/duration/size 已完整（sync 聚合完成且已落 Redis）；但**这是"尽力而为"，存在秒级竞态窗口**，严格保证需要以 `record/files`（或 Redis `SRS_RECORD_M3U8_ARTIFACT`）为准。且注意 end 回调本身可能因 ffmpeg 转换 delay 数秒到数分钟。

## 5. 回调是否可透传 nn/duration/size（callback.go 白名单）

`platform/callback.go` `OnRecordMessage`（L380-524）—— 回调 payload 是**硬编码匿名 struct**：

```go
req := &struct {
    RequestID string `json:"request_id"`
    Action       string `json:"action"`
    Opaque       string `json:"opaque"`
    Vhost        string `json:"vhost,omitempty"`
    App          string `json:"app,omitempty"`
    Stream       string `json:"stream,omitempty"`
    UUID         string `json:"uuid,omitempty"`
    ArtifactCode *int   `json:"artifact_code,omitempty"`
    ArtifactPath string `json:"artifact_path,omitempty"`
    ArtifactURL  string `json:"artifact_url,omitempty"`
}{...}
```

- `on_record_end` 额外字段：`artifact_code`（artifact.Processing 时非 0，SrsStackErrorCallbackRecord）、`artifact_path`（服务器本地 `/data/record/{uuid}/index.mp4`）、`artifact_url`（`{Host}/terraform/v1/hooks/record/hls/{uuid}/index.mp4`）（L422-430）。
- **没有任何 nn / duration / size 字段；调用参数里 artifact *M3u8VoDArtifact 被传进来却没有被序列化进请求体**（只用了 Processing 和 UUID 拼路径）。这是源码级硬编码，非配置项，v5.14 与 main 一致。
- 更新到新版本同样无此字段（main L399-430 相同白名单）。
- 目标地址：`CallbackConfig.Target`（`/terraform/v1/mgmt/hooks/apply` 配置，存 Redis `SRS_HOOKS`，Opaque=token，All 控制开关，L526-560）；回调是**同步 POST**（HTTP client 默认，L311-355），失败只写日志（L520-522），不重试。

### 结论：要拿可靠 nn/duration/size 的唯一官方途径

1. **首选**：收到 `on_record_end` 后调 `POST /terraform/v1/hooks/record/files`（`{token}` + `SRS_PLATFORM_SECRET`），按 `uuid` 匹配记录，读取 `nn/duration/size/progress=false`。因为 `on_record_end` 触发前 `saveArtifact` 已完成，此时数据已就绪。
   - 限制：HScan count=100 单次、无分页续扫 → 记录数 > 100 时可能查不到（需加重试/游标自实现，「每次从 cursor 继续」是可行的：HScan 返回 cursor 但 Oryx 不返回，需要自行按 offset 处理——严格说是官方接口缺陷）。
2. **次选/兜底**：绕过 API 直读 Oryx 的 Redis：`HGET SRS_RECORD_M3U8_ARTIFACT <uuid>`，反序列化 `M3u8VoDArtifact`，`len(files)=nn`、`sum(files[i].duration)=duration`、`sum(files[i].size)=size`（与 record/files 计算方式完全一致），无 100 条限制。
3. 无法通过 on_record_begin/on_record_end 回调本身获得（源码级白名单）。

## 关键源码位置索引（release/5.14）

| 文件 | 行 | 内容 |
|---|---|---|
| `platform/containers/conf/srs.release.conf` | L76-83 | http_hooks on_hls → `/terraform/v1/hooks/srs/hls` |
| `platform/srs-hooks.go` | L709-791 | handleOnHls 接收/校验/分发（record/dvr/vod/transcript） |
| `platform/dvr-local-disk.go` | L539-574 | OnHlsTsMessage：cp + stat 得 size + 异步入队 |
| `platform/dvr-local-disk.go` | L636-721 | buildM3u8Object：LoadOrStore(uuid)、addMessage、持久化 WORKING |
| `platform/dvr-local-disk.go` | L795-807 | updateArtifact：`Files = append(...)`、`NN=len(Files)` |
| `platform/dvr-local-disk.go` | L998-1027 | serveMessage：rename ts、updateArtifact、saveArtifact、`record consume msg` 日志 |
| `platform/dvr-local-disk.go` | L912-996 | Run 循环：300ms 轮询；结束流程（finishM3u8 → finishArtifact → saveArtifact → cancel） |
| `platform/dvr-local-disk.go` | L1091-1115 | callbackBegin / callbackEnd（defer 触发） |
| `platform/dvr-local-disk.go` | L335-391 | `/terraform/v1/hooks/record/files`：HScan + 实时求和 |
| `platform/dvr-local-disk.go` | L296-333 | `/terraform/v1/hooks/record/end`：置 Expired=true |
| `platform/callback.go` | L380-524 | OnRecordMessage：**白名单 payload（无 nn/duration/size）** |
| `platform/utils.go` | L960-981 | TsFile{seqno,duration,size} |
| `platform/utils.go` | L993-1033 | M3u8VoDArtifact{Files[], NN, Processing(progress), Update,...} |
| `platform/utils.go` | L1079-1110 | SrsOnHlsMessage（seq_no tag） |
| `platform/utils.go` | L262-264 | Redis key 常量：SRS_RECORD_PATTERNS / _WORKING / _ARTIFACT |

URL 基址：`https://github.com/ossrs/oryx/blob/release/5.14/<path>#L<line>`

## 对照 main（v6）线差异

- `main/platform/callback.go` L399-430：白名单完全相同（request_id/action/opaque/vhost/app/stream/uuid/artifact_code/path/url）。
- `main/platform/dvr-local-disk.go` L333-391：`record/files` 实现相同（HScan count=100、实时求和、字段一致）。
- `main/platform/srs-hooks.go` L715：仍是 `/terraform/v1/hooks/srs/hls`。
- 结论：**升级 v6 不改变任何结论**；唯一差异在 OCR/AI-talk 等其他功能文件。

## Caveats / Not Found

- 未发现 Oryx 有把 nn/duration/size 放进 HTTP 回调的任何代码路径或配置项（grep `"nn"`/`"duration"`/`"size"` 在 callback.go 零命中）。
- `record/files` 未返回 HScan cursor，>100 条录制记录时单次调用不可穷举——本次调研未在 5.14/main 找到分页参数（cursor/offset 均未暴露），属接口固有缺陷。
- on_hls 触发与 on_record_end 之间的完整性无硬保证（上文竞态分析）；若业务要 LTV/CPS 计量等强一致场景，建议以 Redis 直读为准并轮询兜底。
- 未提供（可通过 `redis-cli` 验证）：生产环境 `SRS_RECORD_PATTERNS/all` 是否 true、`SRS_HOOKS/target`、`SRS_HOOKS/all` 等运行时值——建议结论落地前在部署环境确认。
