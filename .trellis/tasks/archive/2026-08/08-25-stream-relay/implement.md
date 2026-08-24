# implement.md — Stream Relay（FFmpeg 转推）

## 实现顺序（有序清单）

1. **Proto 定义**
   - `app/oryxserver/oryxserver.proto`：新增 `StreamRelayReq/Res`、`StreamRelayStopReq/Res` 消息 + `service OryxServer` 两个 RPC。
   - 运行 `app/oryxserver/gen.sh` 重新生成 pb/grpc 代码（`--client=false`，仅服务端）。

2. **Config**
   - `app/oryxserver/internal/config/config.go`：新增 `DeployMode`、`RelayConfig{SrsRtmpAddr, DefaultApp}`、`MqttConfig{mqttx.MqttConfig}`。
   - `app/oryxserver/etc/oryxserver.yaml`：补充 `DeployMode`、`RelayConfig` 示例（MqttConfig 注释掉）。

3. **RelayManager（internal/relay 包）**
   - `internal/relay/manager.go`：`Manager` + `sync.Map`，`Start/Stop/TaskIDs`。
   - `internal/relay/ffmpeg.go`：ffmpeg-go 构建 + `cmd.Start()` + 退出监听 goroutine（打印 error 日志）。

4. **ServiceContext 装配**
   - `internal/svc/servicecontext.go`：初始化 `RelayManager`；按 `DeployMode` 初始化 `MqttClient`（cluster 模式）。

5. **Logic 层**
   - `internal/logic/streamrelaylogic.go`：参数校验 → 拼接 target → `RelayManager.Start` → 返回。
   - `internal/logic/streamrelaystoplogic.go`：本地 `Stop` → 未命中走 MQTT 广播（cluster）。

6. **Server 注册**
   - `internal/server/oryxserverserver.go`：补 `StreamRelay`/`StreamRelayStop` 两个方法（gen 后可能自动生成，需核对）。

7. **MQTT 广播（internal/relay/broadcast.go）**
   - `RelayBody{ Tid, AckTopic, Method, TaskId }`、`RelayAckBody{ Tid, Success, Error, ErrorKind }`。
   - 发送：`mqttx.RequestReply[*RelayAckBody]`（对齐 ieccaller `PushPbBroadcastWithAck`）。
   - 接收：`AddHandlerFunc` → 本地查 task → cancel → ack。

## 校验命令

```bash
go build ./app/oryxserver/...
go vet ./app/oryxserver/...
```

## 参考实现（关键文件）

- `app/ieccaller/internal/svc/servicecontext.go`：MQTT 初始化 + `PushPbBroadcastWithAck` / `pushBroadcast` / `IsBroadcast`。
- `app/ieccaller/mqtt/broadcast.go`：广播消费 + `publishAckReply`。
- `app/ieccaller/internal/config/config.go`：`DeployMode` + `MqttConfig` 结构。
- `common/mqttx/{client.go,config.go,request_replyer.go,reply_router.go}`：MQTT 客户端 + Request/Reply。
- `common/antsx/{replypool.go,promise.go,errors.go}`：ReplyPool + ErrReplyExpired/ErrDuplicateID。
- `common/iec104/types/types.go`：`BroadcastBody`/`BroadcastAckBody`（结构参照，但本任务在 internal/relay 自定义）。
- `common/mediax/mediax.go`：ffmpeg-go 现有用法（截图，非转推）。
- `app/oryxserver/internal/logic/srsstreamslogic.go`：logic 层错误处理范式（`tool.NewErrorByPbCodeWrap`）。

## 关键细节 / 风险点

- **ffmpeg-go 停止**：必须用 `OutputContext` 传可取消 ctx，`cancel()` 才能杀进程；`Compile()` 内部是 `exec.CommandContext`。
- **退出监听**：`cmd.Wait()` 在 goroutine 中调用，返回时打印 error 日志并清理 sync.Map；注意 `Wait()` 前不能重复 `Start()`。
- **错误码**：参数缺失用 `extproto.Code__1_01_PARAM_MISSING`；非集群下「任务不在本节点」用 `extproto.Code__1_06_THIRD_PARTY`（或按项目既有范式）。
- **集群 ack 超时**：`ReplyRouter` TTL 建议 10s（同 ieccaller）。
- **FFmpeg 依赖**：部署环境需有 `ffmpeg` 可执行文件。
- **生成文件不可手改**：`oryxserver.pb.go`、`oryxserverserver.go` 等由 gen.sh 生成。

## 回滚点

- 任务无持久化，回滚仅需回退代码；无数据迁移。
- 若 FFmpeg 未安装，转推启动会失败——确保部署环境就绪。
