# Journal - boss (Part 4)

> Continuation from `journal-3.md` (archived at ~2000 lines)
> Started: 2026-08-25

---



## Session 177: 重构 mqttx reply 逻辑，抽取通用广播 ack SDK

**Date**: 2026-08-25
**Task**: 重构 mqttx reply 逻辑，抽取通用广播 ack SDK
**Branch**: `master`

### Summary

新增 common/mqttx/broadcast 广播集群 SDK（djisdk 风格主题定义、协议中立 body、errorKind 注册表、消费分发骨架）；ieccaller 与 oryxserver 迁移至 SDK（前缀 oryx/server、iec；13+1 executor 注册、ErrSkipAck 语义、iec_rejected 业务注册）；wire ack 通道确定 broadcast_reply/{id}；spec 更新 messaging/oryx/iec104；全仓 go test ./... 通过

### Git Commits

| Hash | Message |
|------|---------|
| `694e6e6b` | (see git log) |

### Status

[OK] **Completed**


## Session 178: 广播装配闭环重构与 nacos 元数据对齐

**Date**: 2026-08-25
**Task**: 广播装配闭环重构与 nacos 元数据对齐
**Branch**: `master`

### Summary

将 MQTT 广播注册收敛到 NewServiceContext 闭环：业务执行器包（app/*/mqtt）改为收窄依赖（RelayManager / ClientManager / Store），不再注入 ServiceContext（消除 svc→mqtt→svc 导入环）；main() 删除接线样板；oryxserver 补齐 nacos 广播元数据（deployMode/broadcastTopic/broadcastAckTopic/broadcastInstanceId + svc 公开访问器，对齐 ieccaller）；spec 更新 messaging-guidelines 装配约定与元数据约定；build/vet/gofmt/两 app+mqttx 测试全绿

### Git Commits

| Hash | Message |
|------|---------|
| `a6474941` | (see git log) |
| `0a430656` | (see git log) |
| `fb6b91fa` | (see git log) |

### Status

[OK] **Completed**


## Session 179: 流媒体 relay 协议调研：RTMP vs SRT vs HTTP-FLV

**Date**: 2026-08-25
**Task**: 流媒体 relay 协议调研：RTMP vs SRT vs HTTP-FLV
**Branch**: `master`

### Summary

纯咨询会话，无代码改动。调研 RTMP relay 与其他协议（SRT/HTTP-FLV/HLS/WebRTC）对比：SRS Edge 只支持 RTMP/FLV 回源；SRT（UDP+ARQ）在公网弱网场景优于 RTMP，定位是中继/回传协议而非分发协议；平台上编码器接入仍以 RTMP 为主。附 ffmpeg 推 RTMP/SRT 与播放 SRT 的地址示例。用户确认不创建任务。

### Git Commits

| Hash | Message |
|------|---------|
| `a6474941` | (see git log) |
| `0a430656` | (see git log) |
| `fb6b91fa` | (see git log) |

### Status

[OK] **Completed**


## Session 180: StreamRelay: FFmpeg 生命周期清理与优雅关闭加固

**Date**: 2026-08-25
**Task**: StreamRelay: FFmpeg 生命周期清理与优雅关闭加固
**Branch**: `master`

### Summary

排查 StreamRelay 转推问题：确认 ffmpeg-go 无拦截逻辑，二级同 key 推流被 SRS 保旧拒新（1028→Input/output error，exit 251）；新增 Manager.StopAll()+sync.WaitGroup 等待 FFmpeg 全部退出，接入 go-zero proc.AddShutdownListener（defer waitForCalled 保证等待），配置 GracePeriod(proc.SetTimeToForceQuit)，用 .Silent(true) 关闭 ffmpeg-go 包级 std log 改走 logx（LogCompiledCommand/Silent 为包级全局影响 common/mediax），并更新 oryx-guidelines.md 落库生命周期契约与反模式。

### Git Commits

| Hash | Message |
|------|---------|
| `8d7c94df` | (see git log) |

### Status

[OK] **Completed**
