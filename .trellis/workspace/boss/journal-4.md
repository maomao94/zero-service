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


## Session 185: StopRelayAndRecording + relay-merge-distributed

**Date**: 2026-08-28
**Task**: StopRelayAndRecording + relay-merge-distributed
**Branch**: `master`

### Summary

Implemented StopRelayAndRecording RPC in oryxserver (proto + logic + spec). Completed relay-merge-distributed task: merged RelayRegistry and DistributedRelay into single struct, removed hook callbacks, updated all callers. All acceptance criteria met.

### Git Commits

| Hash | Message |
|------|---------|
| `8d1d1e04` | (see git log) |
| `815f4c58` | (see git log) |
| `27258e44` | (see git log) |

### Status

[OK] **Completed**


## Session 186: Relay UID 统一重构 + 扫描器补偿 + 日志优化

**Date**: 2026-08-28
**Task**: Relay UID 统一重构 + 扫描器补偿 + 日志优化
**Branch**: `master`

### Summary

1) CanonicalUID 实现 + Redis key 统一为 uid；2) Sorted Set registry 索引；3) scanner pending 最终补偿（PendingStaleThreshold=5min）；4) NodeReporter key 去重；5) 日志前缀规范化（[asynq-task]）+ 全中文；6) spec 更新

### Git Commits

| Hash | Message |
|------|---------|
| `739518d0` | (see git log) |
| `cab3623d` | (see git log) |
| `532e151f` | (see git log) |
| `ba66579b` | (see git log) |
| `27c4be3a` | (see git log) |

### Status

[OK] **Completed**


## Session 187: asynqx 日志优化

**Date**: 2026-08-28
**Task**: asynqx 日志优化
**Branch**: `master`

### Summary

统一 common/asynqx 包日志格式：添加 [asynq] 模块前缀，改用 logx.Infow/Errorw 结构化字段，补充 addr/db/queue 等关键上下文

### Git Commits

| Hash | Message |
|------|---------|
| `64fa6484` | (see git log) |
| `f9d23c2b` | (see git log) |

### Status

[OK] **Completed**


## Session 188: Relay UUID + app:stream key 重构

**Date**: 2026-08-31
**Task**: Relay UUID + app:stream key 重构
**Branch**: `master`

### Summary

完成中继系统重构：Redis key 从 host_port/app/stream 改为 app:stream 格式，引入 UUID 作为中继会话标识防止 Asynq 回调误操作。更新 state.go、registry.go、logic 层、task 层、cron 扫描器，同步更新 oryx-guidelines.md 和 concurrency-guidelines.md 规格文档。

### Git Commits

| Hash | Message |
|------|---------|
| `f938522f` | (see git log) |
| `7f958da3` | (see git log) |

### Status

[OK] **Completed**


## Session 189: Quick check-in

**Date**: 2026-08-31
**Task**: Quick check-in
**Branch**: `master`

### Summary

No active task. Reviewed enqueue code briefly, no changes made.

### Git Commits

(No commits - planning session)

### Status

[OK] **Completed**


## Session 190: 完善 LiveKit 中文学习与对接指南

**Date**: 2026-08-31
**Task**: 完善 LiveKit 中文学习与对接指南
**Branch**: `master`

### Summary

审阅本地 LiveKit Server 与 server-sdk-go 源码，修正 Server SDK、Protocol、Webhook、Egress、SIP、Agent、Cloud/自托管边界和示例错误；新增中文学习指南、LiveKit backend spec、docs 导航与任务研究记录。核心 Server SDK 示例通过临时 Go module 编译，文档检查和 git diff --check 通过；完整 SDK 测试仍受本机 soxr/opusfile 依赖和未运行 LiveKit Server 限制。

### Git Commits

| Hash | Message |
|------|---------|
| `096c2efc` | (see git log) |

### Status

[OK] **Completed**


## Session 191: 全面复核 LiveKit 学习指导并升级到最新版本

**Date**: 2026-08-31
**Task**: 全面复核 LiveKit 学习指导并升级到最新版本
**Branch**: `master`

### Summary

全面核对 LiveKit 指南全文、backend spec、导航和审计工件；基线统一为 LiveKit Server v1.13.6、server-sdk-go/v2 v2.18.1、Protocol v1.49.0，分离记录本地开发 commit。恢复并核验房间/参与者、Egress、Ingress、SIP、Agent Dispatch、Token、实时参与者、Webhook、Twirp、媒体和部署排障内容，所有 Go 功能示例补充中文注释。稳定 SDK 全 API fixture 通过 go mod tidy/go test ./...，zero-service 全仓 go test ./...、git diff --check、任务 JSONL 校验通过。GitHub curl 链接检查受网络超时限制，未宣称真实 Server、媒体、TURN、Redis、Egress/Ingress/SIP/Agent 端到端通过。

### Git Commits

| Hash | Message |
|------|---------|
| `2be0ada1` | (see git log) |

### Status

[OK] **Completed**
