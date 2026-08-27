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


## Session 181: Centralize FFmpeg process management

**Date**: 2026-08-27
**Task**: Centralize FFmpeg process management
**Branch**: `master`

### Summary

将 FFmpeg 通用进程生命周期集中到 common/ffmpegx.Manager，采用 context-first CommandBuilder 与 per-process hooks；progress 仅在命令明确输出到 stdout 时消费。relay 改为 PullRegistry 管理元数据和业务钩子，服务关闭同时清理 relay 并停止全局 FFmpeg。补充 progress、生命周期、竞态测试并更新规范。目标测试、race、build、target vet 和全量测试通过；全仓 vet 仅剩两个任务外既有问题。

### Git Commits

| Hash | Message |
|------|---------|
| `0b3ad277` | (see git log) |

### Status

[OK] **Completed**


## Session 182: Simplify FFmpeg manager lifecycle

**Date**: 2026-08-27
**Task**: Simplify FFmpeg manager lifecycle
**Branch**: `master`

### Summary

简化 common/ffmpegx.Manager：process 收敛为 id/cmd/cancel/handlers/done，删除 atomic 状态机、AfterFunc、启动 channel 和 FFmpeg 参数扫描；使用 RWMutex 仅保护进程表，业务边界负责 WithoutCancel。exit/progress callback 同步执行，明确 done 与 callback 完成语义，并修复快速退出注册窗口。目标测试、10 轮 race、全仓 build/test、目标 vet 和 diff 检查通过。

### Git Commits

| Hash | Message |
|------|---------|
| `a5d02e40` | (see git log) |
| `e13be258` | (see git log) |
| `fb9d0743` | (see git log) |
| `f0522043` | (see git log) |
| `0bbb3837` | (see git log) |

### Status

[OK] **Completed**


## Session 183: FFmpeg exit/watch contract simplification

**Date**: 2026-08-27
**Task**: FFmpeg exit/watch contract simplification
**Branch**: `master`

### Summary

Simplified ffmpegx.Manager: removed starting map, added WithStderrHandler for streaming stderr via pipe (no unbounded buffer), removed error return from BuildRelayCmd, added [ffmpegx] log prefix, concurrent stdout/stderr scanning with WaitGroup.Go. Updated concurrency and oryx specs.

### Git Commits

| Hash | Message |
|------|---------|
| `d0c6a268` | (see git log) |

### Status

[OK] **Completed**


## Session 184: Simplify StartRelay return + logic-layer HasLease check

**Date**: 2026-08-27
**Task**: Simplify StartRelay return + logic-layer HasLease check
**Branch**: `master`

### Summary

StartRelay 返回 (string, error)，alreadyRunning 检查移到 Logic 层（HasLease）。lock 失败时检查租约而非直接返回 alreadyRunning=true。proto 注释完善 stream 唯一性说明。

### Git Commits

| Hash | Message |
|------|---------|
| `cba4cf1e` | (see git log) |

### Status

[OK] **Completed**
