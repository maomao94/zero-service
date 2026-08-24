# design.md — Stream Relay（FFmpeg 转推）

## 架构与边界

```
业务服务 ---Nacos---> oryxserver (zRPC :21016)
                          │
                          ├─ StreamRelay    → 启动 FFmpeg 进程（拉流 → 转推 SRS）
                          ├─ StreamRelayStop→ 停止 FFmpeg 进程（本地 / MQTT 广播）
                          └─ (后续) 转码
                              │
                              v
                     SRS (rtmp://srs:1935/<app>/<stream>)
```

- 转推任务**仅内存管理**（`sync.Map`），进程退出即清理。
- FFmpeg 是常驻进程：`Start()` 后立即返回，`Wait()` 由后台 goroutine 监听退出并打印 error 日志。

## 数据流与契约

### Proto 契约（已与用户确认）

```protobuf
// 转推流请求
message StreamRelayReq {
  string source_url = 1;   // 源流地址（RTMP/RTSP/HTTP-FLV/HLS 等 FFmpeg 支持的协议）
  string app = 2;          // 目标应用名，默认 "live"
  string stream = 3;       // 目标流名，不填则自动生成 UUID
}

message StreamRelayRes {
  string task_id = 1;      // 转推任务 ID（停止转推时使用）
  string app = 2;          // 目标应用名（实际使用值）
  string stream = 3;       // 目标流名（自动生成时返回生成值）
}

// 播放地址约定（业务侧按协议自行拼接，地址均指向 SRS/Oryx）：
//   RTMP:     rtmp://<host>:<rtmp_port>/{app}/{stream}
//   HTTP-FLV: http://<host>:<http_port>/{app}/{stream}.flv
//   HLS:      http://<host>:<http_port>/{app}/{stream}.m3u8
//   WebRTC:   webrtc://<host>:<rtc_port>/{app}/{stream}
// 注：Oryx 无「获取播放地址」接口，/terraform/v1/mgmt/streams/query 仅返回
//     vhost/app/stream 等元数据；以上均为 SRS 标准约定格式。

message StreamRelayStopReq {
  string task_id = 1;      // 转推任务 ID
}

message StreamRelayStopRes {}
```

- proto 字段 snake_case + 全量 `json_name`（camelCase），遵循项目契约规范。
- 新增 RPC 加入 `service OryxServer`，位于「平台级接口」分组（业务服务经 Nacos 调用）。
- **播放地址约定**：Oryx 无「获取播放地址」接口，由业务侧按 proto 中注释的协议约定（RTMP/HTTP-FLV/HLS/WebRTC）自行拼接；oryxserver 只返回 `app`/`stream`。

### StreamRelay 启动流程

```
1. 校验 source_url 非空
2. app 为空 → 取配置 DefaultApp（默认 live）
3. stream 为空 → tool.SimpleUUID()
4. 目标地址 = {SrsRtmpAddr}/{app}/{stream}
5. ctx, cancel := context.WithCancel(context.Background())
6. 构建 ffmpeg-go Stream → Compile() 得 *exec.Cmd
7. cmd.Start()（非阻塞）
8. 任务 {TaskID, Cancel, Cmd, SourceURL, ...} 存入 sync.Map
9. goroutine: cmd.Wait() → 退出则打印 error 日志 + 从 sync.Map 清理
10. 返回 task_id + app（实际值）+ stream（自动生成时返回生成值）
```

### StreamRelayStop 停止流程

```
1. 本地 sync.Map 查 task_id
2. 命中 → cancel() 停止进程，从 map 清理，返回成功
3. 未命中且 IsCluster() → MQTT 广播 stop 命令 + 等待 ACK（参考 ieccaller）
4. 未命中且非集群 → 返回明确错误（"任务不在当前节点"）
```

### FFmpeg 命令等价形式

```
ffmpeg -i <source_url> -c copy -f flv rtmp://<srs>/<app>/<stream>
```

ffmpeg-go 构建方式：

```go
in := ffmpeg.Input(sourceURL)
out := ffmpeg.OutputContext(ctx, []*ffmpeg.Stream{in}, targetURL, ffmpeg.KwArgs{
    "c": "copy",
    "f": "flv",
})
out = out.WithErrorOutput(stderrBuf)   // 捕获 stderr，Wait 退出时打印
cmd := out.Compile()
cmd.Start()
```

要点：
- `OutputContext` 传入可取消 `ctx`，`Compile()` 内部 `exec.CommandContext(s.Context, ...)`；`cancel()` 即 kill 进程。
- `WithErrorOutput` 用 `context.WithValue` 追加 "Stderr"，不破坏 parent cancel 语义。
- `cmd.Wait()` 返回非 nil（进程退出/被杀）→ error 日志（含 task_id、source_url、stderr）。

## MQTT 广播（可选，仅 cluster 模式）

完全对齐 `app/ieccaller` 模式（`common/mqttx` + `common/antsx`）：

| Topic | 用途 |
|---|---|
| `oryx/relay/broadcast` | 跨节点 stop 命令广播 |
| `oryx/relay/broadcast-ack/{instanceId}` | 每实例 ack 回复 |

- 发送端：`mqttx.RequestReply[*RelayAckBody](ctx, client, ackTopic, tid, send)`。
- 接收端：`AddHandlerFunc(broadcastTopic, ...)` 收到 stop 命令后，本地查 task → `cancel()` → 回 ack。
- `RelayBody{ Tid, AckTopic, Method, TaskId }`、`RelayAckBody{ Tid, Success, Error, ErrorKind }`，定义在 `internal/relay` 包（不复用 iec104 types，避免跨域耦合）。
- 忽略 `AckTopic == 本机 ackTopic` 的消息（防自处理）。

## 服务装配

### Config 新增

```go
type Config struct {
    zrpc.RpcServerConf
    // 部署模式：standalone / cluster（对齐 ieccaller）
    DeployMode string `json:",default=standalone,options=standalone|cluster"`
    NacosConfig struct{ ... }
    OryxConfig struct{ ... }
    DB gormx.Config
    // 转推配置
    RelayConfig struct {
        SrsRtmpAddr string `json:",default=rtmp://127.0.0.1:1935"` // SRS RTMP 地址
        DefaultApp  string `json:",default=live"`                  // 默认应用名
    } `json:",optional"`
    // MQTT 配置（可选，cluster 模式必需）
    MqttConfig struct {
        mqttx.MqttConfig
    } `json:",optional"`
}
```

### ServiceContext 新增

```go
type ServiceContext struct {
    Config      config.Config
    OryxClient  *oryxx.Client
    DB          *gormx.DB
    RelayManager *relay.Manager        // 转推任务管理（sync.Map + FFmpeg 进程）
    MqttClient   mqttx.Client          // 可选，cluster 模式
    // broadcast 相关（cluster 模式）
    relayInstanceId, relayTopic, relayAckTopic string
}
```

- `RelayManager` 置于独立 `internal/relay` 包，屏蔽 FFmpeg 进程细节，logic 层仅调用其方法。
- MQTT 初始化条件：`DeployMode == "cluster"` 且 `MqttConfig.Broker` 非空，否则为 nil（单节点不依赖 MQTT）。

## 公共/内部包

- `internal/relay/manager.go`：`Manager{ tasks sync.Map }`，方法 `Start(ctx, source, target) (taskID, error)`、`Stop(taskID) (bool, error)`、`TaskIDs()`。
- `internal/relay/ffmpeg.go`：ffmpeg-go 命令构建与进程启动/监听封装。
- `internal/relay/broadcast.go`：cluster 模式的 MQTT 广播发送/接收（RelayBody/RelayAckBody）。

## Trade-offs

- **内存管理**：任务不落库，进程重启后转推任务丢失——第一版可接受，后续如需可加 Redis 状态。
- **不做自动重连**：FFmpeg 退出即结束并打印日志，符合「先打印 error」的第一版要求。
- **MQTT 可选**：单节点零依赖，集群才启用广播；与 ieccaller 的 `DeployMode` 语义一致。

## 运维与回滚

- 转推任务随 oryxserver 进程生命周期，无持久化，回滚无数据迁移。
- FFmpeg 需部署环境存在 `ffmpeg` 可执行文件（PATH 可查）。
- 生成文件（pb.go/orxxserverserver.go）由 gen.sh 生成后手工补充 logic 层。
