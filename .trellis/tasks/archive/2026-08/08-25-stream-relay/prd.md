# Stream Relay API - FFmpeg 转推接口

## Goal

在 oryxserver 新增 gRPC 接口，业务侧传入一个已有的流地址，oryxserver 启动 FFmpeg 拉流转推到 Oryx/SRS，为后续动态转码做准备。

## Background

- Oryx/SRS 的 Ingest/Transcode 均为**配置驱动**，没有「调一个 API 就开始拉流/转推」的动态能力。
- 本任务是第一轮：只做**转推（relay）**，不做转码。转码作为后续迭代。
- 业务侧只调 API，不接触 FFmpeg 进程与流处理细节。
- 错误处理从简：拉不到流（流断开/流不存在）→ 打印 error 日志，不做自动重试/重启。

## Requirements

- R1：`StreamRelay` RPC —— 接收源流地址，启动 FFmpeg 转推到 SRS，返回 task_id、实际 app/stream（播放地址为协议约定，业务侧自行拼接，proto 注释写明支持协议）。
- R2：`StreamRelayStop` RPC —— 接收 task_id，停止对应 FFmpeg 进程。
- R3：FFmpeg 进程由 oryxserver 内部管理（启动/停止/退出监听），对业务侧透明。
- R4：FFmpeg 进程异常退出（流断开、拉流失败）→ 打印 error 日志（含 task_id、源地址、stderr）。
- R5：停止转推支持分布式——若 task 不在当前节点，通过 MQTT 广播到集群（单节点部署时 MQTT 可选，无需配置）。
- R6：目标流名未指定时自动生成。

## Acceptance Criteria

- [ ] `StreamRelay` 可接收源流地址并启动 FFmpeg 转推，返回 task_id、app、stream（含协议约定注释）。
- [ ] `StreamRelayStop` 可停止本地节点的指定转推任务。
- [ ] FFmpeg 进程退出时打印 error 日志（含 task_id、源地址、stderr）。
- [ ] FFmpeg 命令等价于：`ffmpeg -i <source_url> -c copy -f flv rtmp://<srs>/<app>/<stream>`。
- [ ] MQTT 未配置（单节点）时，`StreamRelayStop` 对非本地 task 返回明确错误，服务不 panic。
- [ ] `go build ./app/oryxserver/...` 通过。

## Out of Scope（本阶段不做）

- 转码（后续迭代，本任务仅 copy 转推）。
- FFmpeg 进程崩溃自动重启/自动重连。
- 转推任务持久化入库（仅内存 `sync.Map` 管理）。
- 转推任务状态查询接口。
- grpc-gateway / HTTP 对外网关。

## Key Decisions（已与用户确认）

- 使用 **ffmpeg-go**（`github.com/u2takey/ffmpeg-go`，已在 go.mod）构建命令，通过 `Compile()` 拿到 `*exec.Cmd` 做进程生命周期管理。
- 停止机制：`OutputContext` 传入可取消 `context`，`cancel()` 停止进程（`exec.CommandContext` 语义）。
- MQTT 配置**可选**，参考 `ieccaller` 的 `DeployMode`（`standalone`/`cluster`）+ MQTT Broadcast + Request/Reply 模式。
- task_id 使用 `tool.SimpleUUID()`。

## Notes

- proto 字段设计见 design.md「Proto 契约」。
- MQTT 广播模式参考 `app/ieccaller`（`common/mqttx` + `common/antsx` + `iec104/types.BroadcastBody/BroadcastAckBody`）。
