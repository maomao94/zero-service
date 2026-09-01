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


## Session 192: 开发 common/livekitx 快速开箱包

**Date**: 2026-08-31
**Task**: 开发 common/livekitx 快速开箱包
**Branch**: `master`

### Summary

基于 server-sdk-go/v2 v2.18.1 实现 common/livekitx 快速开箱包：统一配置与幂等 Close、协议生成管理 client（Room/Egress/Ingress/SIP/AgentDispatch）、Join/SIP Token、实时连接回调桥接、Data/chat/RPC、16 个 typed Hook（Webhook/Room/Participant/Track/Connection/Data/Chat/RPC）、Webhook 验签与分发、Store 可序列化 ConnectionState（默认内存，业务可注入 Redis/DB）、go-zero httpc.Service 与标准 http.Client 注入（互斥）。请求超时由业务 context 控制，不派生默认超时；Connector/AgentSimulation/Cloud Agents 排除。TDD 全程：单测+race+vet 通过，全仓 go test 通过；本地 livekit-server --dev 真实集成 3/3（房间生命周期、Hook 桥接、聊天 Data、RPC 往返）；管理 API Twirp mock 4 项；修复 Webhook 验签 KeyProvider 与 API 房间级 grant 问题；README 全功能中文文档；沉淀 backend livekit-guidelines 规范。

### Git Commits

| Hash | Message |
|------|---------|
| `9bc8de2f` | (see git log) |
| `104133dc` | (see git log) |

### Status

[OK] **Completed**


## Session 193: 简化 livekitx Hook 设计：Room API+原生回调

**Date**: 2026-09-01
**Task**: 简化 livekitx Hook 设计：Room API+原生回调
**Branch**: `master`

### Summary

按用户 9 轮反馈把 common/livekitx 从 typed Hook 分发层重构为极简 Room API + SDK 原生回调：删除 eventDispatcher/16 个 typed Hook/Store/ConnectionState/RealtimeRoom/Connect/默认 callback/EndRoom/ChatTopic；Option 重构为 func(*Client) 直接注入（Config 仅配置项）；JoinRoomOption 接口（WithCallback 场景定制 + WithConnectOption 透传）；JoinRoom 用 ConnectInfo 自动签 token；Room API 增至 9 个（JoinRoom/CreateRoom/CreateAndJoinRoom/DeleteRoom/RemoveParticipant 踢人/InviteParticipant 邀请/MuteParticipant 静音/MuteParticipantVideo 关视频/SendData 语音图片富媒体）；Client.JoinToken 签发入会 token；NewWebhookKeyProvider 验签；SDK 日志默认接 go-zero logx；API 字段加 Service 后缀；错误聚合到 errors.go；文档 docs/livekit-callbacks-guide.md（全部回调场景+字段，SDK v2.18.1 源码核对）+ README 重写；livekit-guidelines 规范更新。验证：单测/race/vet/全仓通过，集成测试 4/4（入会/聊天双路径/RPC/断开原因/SendData/PerJoinCallback）。

### Git Commits

| Hash | Message |
|------|---------|
| `d0b87ca3` | (see git log) |
| `c4d64cd1` | (see git log) |
| `053bdb22` | (see git log) |

### Status

[OK] **Completed**


## Session 194: livekitx 开放 UpdateSubscriptions/ListParticipants 透传

**Date**: 2026-09-01
**Task**: livekitx 开放 UpdateSubscriptions/ListParticipants 透传
**Branch**: `master`

### Summary

新增 UpdateSubscriptions（强制订阅/取消订阅轨道）与 ListParticipants（参与者列表）透传方法，业务自判调用时机；muteParticipantTracks 复用 ListParticipants；补 mock 测试断言请求字段与参数校验；更新 livekit-guidelines spec。

### Git Commits

| Hash | Message |
|------|---------|
| `e59317fa` | (see git log) |

### Status

[OK] **Completed**


## Session 195: livekitx 补齐 RoomService 透传方法与 LiveKit 概念梳理

**Date**: 2026-09-01
**Task**: livekitx 补齐 RoomService 透传方法与 LiveKit 概念梳理
**Branch**: `master`

### Summary

补齐 5 个透传方法（ListRooms/GetParticipant/UpdateParticipant/UpdateRoomMetadata/PerformRpc），api.go requestRoom 增加 PerformRpcRequest 房间限定，补 mock 测试；梳理 SIP/PhoneNumberService/11 个 Twirp service 能力边界；spec 沉淀 PerformRpc 服务端语义、SendData 选型规则、PhoneNumberService/SIP 边界。

### Git Commits

| Hash | Message |
|------|---------|
| `e39435dc` | (see git log) |
| `47af16f9` | (see git log) |

### Status

[OK] **Completed**


## Session 196: live-gtw: trigger 废弃方法替换 + livegtw 网关实现

**Date**: 2026-09-01
**Task**: live-gtw: trigger 废弃方法替换 + livegtw 网关实现
**Branch**: `master`

### Summary

1. trigger 目录 14 处 tool.GetCurrentUserId(l.ctx, nil) 全部替换为 authctx.GetUserId(l.ctx)（10 个 logic 文件）；2. livegtw 网关骨架实现：10 个会议 API 转发、webhook 验签、测试页路由、配置，编译/vet/单测全绿。

### Git Commits

| Hash | Message |
|------|---------|
| `8611e895` | (see git log) |

### Status

[OK] **Completed**


## Session 197: LiveKit SDK 去封装 + CreateMeeting 参数扩展

**Date**: 2026-09-01
**Task**: LiveKit SDK 去封装 + CreateMeeting 参数扩展
**Branch**: `master`

### Summary

1) CreateMeetingReq 增加 empty_timeout/departure_timeout/max_participants/metadata 四个字段，LiveKit CreateRoom 传递新参数并设默认值；2) 删除 LiveKitAPI 接口封装层，ServiceContext.LiveKit 改为 *livekitx.Client，业务直接调用 SDK；3) 删除 Client.JoinToken 方法，业务直接调用 livekitx.NewJoinToken；4) 删除 room.go 所有 wrapper 方法（CreateRoom/DeleteRoom/RemoveParticipant/SendData/PerformRpc/MuteParticipant 等），保留 Room() 暴露底层 SDK；5) 测试改为 httptest mock server 方式；6) 更新 livekit-guidelines.md spec。

### Git Commits

| Hash | Message |
|------|---------|
| `b55a90f4` | (see git log) |
| `85ca200e` | (see git log) |
| `8d8521c8` | (see git log) |
| `6c337d47` | (see git log) |

### Status

[OK] **Completed**


## Session 198: Spec 分层加载优化与 AI 指导改进

**Date**: 2026-09-01
**Task**: Spec 分层加载优化与 AI 指导改进
**Branch**: `master`

### Summary

优化 .trellis/spec/ 分层加载：新建 core-rules.md（~100行核心规则），删除 coding-standards.md 去重，精简 error-handling 和 service-lifecycle，index.md 增加关键词路由表，go-zero-conventions/gormx/concurrency 等 spec 表格化、Good/Bad 对比优化。总行数 4901→4785，小任务上下文从~500-1000行降到~200-300行。

### Git Commits

| Hash | Message |
|------|---------|
| `38b19207` | (see git log) |
| `b64b4f9d` | (see git log) |
| `dd9b5872` | (see git log) |

### Status

[OK] **Completed**


## Session 199: Live 会议系统 MVP 全栈交付

**Date**: 2026-09-01
**Task**: Live 会议系统 MVP 全栈交付
**Branch**: `master`

### Summary

完成 Live 会议业务系统 MVP：app/live gRPC 服务（11 RPC/会议号生成/RedisLock/webhook）、app/livegtw HTTP 网关（10 API/验签/JWT 测试页路由）、前端页面重写为产品化会议界面（JWT 登录→大厅→群视频/群聊/成员管理/下拉选择/定时轮询）、deploy/livekit 启动脚本、三端联调验证全绿（API/webhook/幂等/落库/重放）。

### Git Commits

| Hash | Message |
|------|---------|
| `7557c395` | (see git log) |

### Status

[OK] **Completed**
