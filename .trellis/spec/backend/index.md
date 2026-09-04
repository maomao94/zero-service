# Backend Coding Specs

> 覆盖 `zero-service` 全部 Go 后端代码。
>
> **加载策略**：先读 [core-rules.md](./core-rules.md)（~100行），再按下方路由读1-2个相关spec。

## 快速路由（按任务关键词）

| 关键词 | 读这个spec |
|--------|-----------|
| Handler, Logic, ServiceContext, API, RPC, .api, .proto, gen.sh | [go-zero-conventions](./go-zero-conventions.md) + [contract-generation](./contract-generation.md) |
| model, store, GORM, 事务, 分页, 租户, Upsert, CAS, RowsAffected | [gormx-guidelines](./gormx-guidelines.md) |
| Scheduler, CronJob, lease, RunNow, asynq, 定时任务 | [crontask-guidelines](./crontask-guidelines.md) + [trigger-guidelines](./trigger-guidelines.md) |
| 错误, error, gRPC status, 日志, log, trace, metadata | [error-handling](./error-handling.md) |
| 日期, 时间, 格式化, 解析, carbonx, carbon, time.Now, time.Format | [carbonx-guidelines](./carbonx-guidelines.md) + [common-package-design](./common-package-design.md) |
| common/, option, 公共包, client, SDK | [common-package-design](./common-package-design.md) |
| goroutine, mr, antsx, Promise, ReplyPool, 并发, 锁 | [concurrency-guidelines](./concurrency-guidelines.md) |
| ServiceContext, 启动, 关闭, 配置, client, scheduler | [service-lifecycle](./service-lifecycle.md) |
| WebSocket, MQTT, HTTP client, 长连接, 消息 | [messaging-guidelines](./messaging-guidelines.md) + [networking-guidelines](./networking-guidelines.md) |
| RRULE, rrulex, 调度规则, 中文描述 | [rrulex-guidelines](./rrulex-guidelines.md) |
| 跨层, 跨服务, 数据流, 复用 | [guides/index.md](../guides/index.md) |

## 领域契约（按功能模块）

| 模块 | 关键词 | spec |
|------|--------|------|
| Trigger | asynq, Plan, Batch, ExecItem, 节假日, CronJob | [trigger-guidelines](./trigger-guidelines.md) |
| ISP | ISP帧, 注册, 命令, 巡检, 回执, SessionID, ClientID | [isp-guidelines](./isp-guidelines.md) |
| IEC 104 | 控制命令, ASDU, IOA, COT, 集群路由 | [iec104-guidelines](./iec104-guidelines.md) |
| DJI | DJI SDK, topic, DRC, hooks, OSD, 拓扑, 飞行区 | [dji-guidelines](./dji-guidelines.md) |
| GIS | 坐标, GEOS, H3, 围栏, FenceStore, GeoJSON | [gis-guidelines](./gis-guidelines.md) |
| 实时事件 | Socket.IO, Kafka, StreamEvent, 实时 payload | [realtime-guidelines](./realtime-guidelines.md) |
| AI/MCP | Eino, tool, runner, MCP, 会话执行 | [ai-guidelines](./ai-guidelines.md) |
| Bridge | bridgegtw, kafka, modbus, mqtt, dump | [bridge-guidelines](./bridge-guidelines.md) |
| Alarm | alarm, 飞书, Lark, 告警, IM群 | [alarm-guidelines](./alarm-guidelines.md) |
| File | file, OSS, 上传, 中继, ossx, filex | [file-guidelines](./file-guidelines.md) |
| LAL | lalhook, lalproxy, lalx, mediax, 流媒体 | [lal-guidelines](./lal-guidelines.md) |
| Oryx | oryxgtw, oryxserver, oryxx, SRS, relay | [oryx-guidelines](./oryx-guidelines.md) |
| Pod Engine | podengine, dockerx, executorx, 容器编排 | [podengine-guidelines](./podengine-guidelines.md) |
| 网络 | netx, wsx, socketiox, ssex, SSE | [networking-guidelines](./networking-guidelines.md) |
| Log Dump | logdump, gRPC-to-logx, 日志汇聚 | [logdump-guidelines](./logdump-guidelines.md) |
| Flow | flowx, Azure, go-workflow, 编排 | [flow-guidelines](./flow-guidelines.md) |
| LiveKit | livekitx, meeting, JWT, Webhook, Twirp, 房间, 录制, SIP, trunk, dispatch, 外呼, 来电 | [livekit-guidelines](./livekit-guidelines.md) |
| Live 前端 | web/live, 聊天, Echo, sendMeetingData, performMeetingRpc, 票据入会, 访客 | [livekit-guidelines → Web 前端（web/live）约定](./livekit-guidelines.md) |

## Pre-Development Checklist

- [ ] 从契约源、定义、调用方、消费者和测试确认真实代码路径
- [ ] 读取本任务触发的基础规范，以及对应公共基础设施/领域规范
- [ ] 区分连接、业务、消息和任务身份；确认状态与持久化字段的写入所有者
- [ ] 标出 context、超时、重试、幂等、空值、并发和资源关闭语义
- [ ] 契约变更确认生成脚本和所有直接调用方；生成文件不手工修改
- [ ] 实验、Mock、历史快照和一次性方案不升级为全局规则

## Quality Check

- [ ] 验证范围由风险决定，至少覆盖成功、失败、取消/超时、重复和边界输入
- [ ] 数据写入检查事务、唯一约束、版本/CAS、`RowsAffected` 与字段所有权
- [ ] 并发和异步代码有退出/关闭策略，并对关键包运行 race test
- [ ] 消息发送成功没有被描述为远端处理、持久化或 Exactly Once
- [ ] Spec、契约源、生成物、实现、测试与受影响文档保持一致
- [ ] `git diff --check` 通过，最终 diff 只包含任务范围且不泄露敏感信息
