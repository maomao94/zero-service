# Agent 知识库

这里保存未来修改代码时仍然需要知道、且无法从单个文件直接看出的稳定契约。它不保存任务计划、完整 API 手册、目录快照或一次性排障过程。

## 读取方式

1. 根据任务触发词选择下面的一条分支。
2. 通常读取一份工程规范和一份领域契约；文档明确链接依赖时再展开。
3. 用源码、测试、`.proto`、`.api`、配置和生成脚本核对规则。
4. 规则已经过时时，在同一改动中修正或删除，完成时所有修改行为均有对应验证。

## 领域语言

术语含义或身份边界不清时，从 [领域上下文地图](../../CONTEXT-MAP.md) 选择一份小型词典；实现约束仍按下方任务分支读取。

## 工程规范

| 任务分支 | 读取 |
| --- | --- |
| Handler、Logic、ServiceContext、RPC/API 服务结构 | [go-zero 服务](./standards/go-zero.md) |
| HTTP Handler、统一响应体、`JsonBaseResponseCtx` | [HTTP 网关响应](./standards/http-gateway.md) |
| `.proto`、`.api`、字段兼容或生成代码 | [契约与生成](./standards/contracts.md) |
| model、GORM、事务、租户、Upsert、CAS | [数据访问](./standards/data.md) |
| goroutine、锁、Promise、任务队列、子进程 | [并发与异步](./standards/concurrency.md)；长期资源再读[生命周期](./standards/lifecycle.md) |
| error、gRPC status、日志、trace、metadata | [错误与上下文](./standards/errors.md) |
| HTTP client、请求编码或 Transport | [HTTP client](./standards/networking/http.md) |
| WebSocket client、重连、心跳 | [WebSocket](./standards/networking/websocket.md) |
| Socket.IO、房间、Session、多节点 | [Socket.IO](./standards/networking/socketio.md) |
| SSE writer、流式响应 | [SSE](./standards/networking/sse.md) |
| MQTT request/reply、消息投递语义 | [客户端与消息](./standards/messaging.md) |
| 实时事件、连接、房间、投递语义 | [实时事件](./standards/realtime.md) |
| `common/`、Option、SDK、公共工具 | [公共包设计](./standards/shared-packages.md)和[仓库边界](./standards/repository.md) |
| 时间解析、格式化、数据库时间 | [日期时间](./standards/time.md) |
| 制定验证范围或交付前检查 | [质量与验证](./standards/quality.md) |

## 领域契约

| 路径或概念 | 读取 |
| --- | --- |
| `app/trigger`、Plan、Batch、ExecItem | [Trigger Plan](./domains/trigger/plans.md) |
| Trigger CronJob、Submit、回调字段 | [Trigger CronJob](./domains/trigger/cron-jobs.md) |
| `CalcPlanTaskDate`、计划描述 | [Trigger 规则描述](./domains/trigger/plan-rules.md) |
| `common/crontask`、lease、RunNow | [调度器](./domains/scheduling.md) |
| `common/rrulex`、RRULE、RDATE、EXDATE | [RRULE](./domains/rrule.md) |
| ISP 帧、巡检、注册、命令回执 | [ISP](./domains/isp.md) |
| IEC 104、ASDU、IOA、COT | [IEC 104](./domains/iec104.md) |
| DJI SDK、DRC、OSD、飞行区 | [DJI](./domains/dji.md) |
| GIS、GEOS、H3、围栏、GeoJSON | [GIS](./domains/gis.md) |
| Bridge、Modbus、MQTT、Kafka、dump | [Bridge](./domains/bridge.md) |
| Alarm、飞书、Lark、告警去重 | [Alarm](./domains/alarm.md) |
| File、OSS、上传、中继、缩略图 | [File](./domains/file.md) |
| Oryx、SRS、relay、FFmpeg 中继 | [Oryx](./domains/oryx.md) |
| LAL、lalhook、lalproxy、mediax | [LAL](./domains/lal.md) |
| Pod Engine、Docker host、executorx | [Pod Engine](./domains/podengine.md) |
| AI、Eino、MCP、tool、runner | [AI](./domains/ai.md) |
| Flow、Azure go-workflow、编排 | [Flow](./domains/flow.md) |
| Log Dump、gRPC-to-logx、日志汇聚 | [Log Dump](./domains/logdump.md) |

### LiveKit 分支

| 任务 | 读取 |
| --- | --- |
| `common/livekitx`、Token、Webhook、Twirp | [公共库](./domains/livekit/common.md) |
| `app/live`、会议、参会记录、会议锁 | [会议服务](./domains/livekit/meetings.md) |
| SIP provider、trunk、dispatch、外呼 | [SIP 服务](./domains/livekit/sip/service.md) |
| LiveKit SIP、FreeSWITCH 或 SIPMediaGW 部署 | [SIP 部署](./domains/livekit/sip/deployment.md) |
| SDK 版本、授权、部署、集成验证 | [平台与运维](./domains/livekit/operations.md) |
| `app/livegtw`、API 类型转换、中间件 | [LiveKit HTTP 网关](./domains/livekit/gateway/service.md) |
| LiveKit 入会票据、Redis ticket | [入会票据](./domains/livekit/gateway/tickets.md) |
| `/test/meeting` 浏览器验证页 | [会议测试页](./domains/livekit/gateway/test-page.md) |
| `web/live`、聊天、Data、RPC、Echo | [Web 前端](./domains/livekit/web.md) |

## 分析指南

- 字段、状态或错误跨两个以上目录或进程：[跨层检查](./guides/cross-layer.md)。
- 新增工具、client、SDK、常量或公共包：[复用决策](./guides/reuse.md)。
- 修改本知识库、README、`docs/` 或协议注释：[文档归属](./guides/documentation.md)。

完整业务流程和对接说明属于 [`docs/README.md`](../README.md)；字段与方法签名属于契约源或相邻 Go doc。研究材料应作为一手来源被链接，不升级为每次修改都必须遵守的规则。
