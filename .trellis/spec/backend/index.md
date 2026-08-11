# Backend Coding Specs

本层覆盖 `zero-service` 全部 Go 后端代码。先读取与改动直接相关的规范；跨目录、跨进程或需要判断复用位置时，再使用 [思考指南](../guides/index.md)。

## 基础规范

| 规范 | 何时读取 |
| --- | --- |
| [目录结构与仓库边界](./directory-structure.md) | 新增/移动目录、选择代码所有者或判断是否可编辑 |
| [编码规范](./coding-standards.md) | 编写或审查 Go、配置、SQL、协议与测试 |
| [质量规范](./quality-guidelines.md) | 制定验证范围、补测试、审查 diff 或交付 |
| [go-zero 服务约定](./go-zero-conventions.md) | 修改 Handler/Server、Logic、ServiceContext 或服务方法 |
| [契约与生成规范](./contract-generation.md) | 修改 `.proto`、`.api`、校验、路由或生成产物 |
| [服务依赖与生命周期](./service-lifecycle.md) | 装配 client/store/scheduler、启动后台循环或关闭资源 |
| [错误与传输边界](./error-handling.md) | 设计领域错误、gRPC/HTTP 映射、context 或日志边界 |

## 公共基础设施

| 规范 | 何时读取 |
| --- | --- |
| [公共包设计](./common-package-design.md) | 新增/扩展 `common/`、option、转换或通用工作流 |
| [GORM 与数据访问](./gormx-guidelines.md) | 修改 model/store、事务、分页、租户、Upsert 或 CAS |
| [并发与异步](./concurrency-guidelines.md) | 使用 goroutine、`mr`、`antsx`、Promise、ReplyPool 或共享状态 |
| [客户端与消息](./messaging-guidelines.md) | 修改 HTTP/WebSocket/MQTT client、关联响应或长连接 |
| [crontask 调度](./crontask-guidelines.md) | 修改 Scheduler、Store、RRULE、lease、`RunNow` 或适配器 |

## 领域契约

| 规范 | 何时读取 |
| --- | --- |
| [Trigger 调度](./trigger-guidelines.md) | 修改 asynq、Plan/Batch/ExecItem、节假日或 CronJob |
| [ISP 协议与巡检](./isp-guidelines.md) | 修改 ISP 帧、身份、注册、命令、巡检或回执 |
| [IEC 104](./iec104-guidelines.md) | 修改控制命令、应答关联、集群路由或 ASDU trace |
| [DJI 接入](./dji-guidelines.md) | 修改 DJI SDK、topic、DRC、hooks、快照或事件持久化 |
| [GIS 与电子围栏](./gis-guidelines.md) | 修改坐标、GEOS、H3、FenceStore 或围栏事务 |
| [实时事件](./realtime-guidelines.md) | 修改 Socket.IO、Kafka、StreamEvent 或实时 payload |
| [AI 与 MCP](./ai-guidelines.md) | 修改 Eino tool/runner、会话执行状态、MCP client/server |
| [Bridge 协议网关](./bridge-guidelines.md) | 修改 bridgegtw/kafka/modbus/mqtt/dump 或 mqttx/modbusx/stream |
| [Alarm 飞书告警](./alarm-guidelines.md) | 修改 alarm 告警发送或 alarmx Lark SDK 封装 |
| [File OSS 文件](./file-guidelines.md) | 修改 file OSS 管理/上传/中继或 ossx/filex 公共库 |
| [LAL 流媒体](./lal-guidelines.md) | 修改 lalhook/lalproxy 服务或 lalx/mediax 公共库 |
| [Pod Engine 编排](./podengine-guidelines.md) | 修改 podengine 容器编排或 dockerx/executorx 公共库 |
| [网络通信](./networking-guidelines.md) | 修改 netx/wsx/socketiox/ssex HTTP/WebSocket/SSE 通信层 |
| [Log Dump 日志汇聚](./logdump-guidelines.md) | 修改 logdump gRPC-to-logx 桥接或日志字段白名单 |
| [Flow 工作流](./flow-guidelines.md) | 修改 flowx Azure go-workflow 编排或日志拦截器 |

## Pre-Development Checklist

- [ ] 从契约源、定义、调用方、消费者和测试确认真实代码路径。
- [ ] 读取本任务触发的基础规范，以及对应公共基础设施/领域规范。
- [ ] 区分连接、业务、消息和任务身份；确认状态与持久化字段的写入所有者。
- [ ] 标出 context、超时、重试、幂等、空值、并发和资源关闭语义。
- [ ] 契约变更确认生成脚本和所有直接调用方；生成文件不手工修改。
- [ ] 实验、Mock、历史快照和一次性方案不升级为全局规则。

## Quality Check

- [ ] 验证范围由风险决定，至少覆盖成功、失败、取消/超时、重复和边界输入。
- [ ] 数据写入检查事务、唯一约束、版本/CAS、`RowsAffected` 与字段所有权。
- [ ] 并发和异步代码有退出/关闭策略，并对关键包运行 race test。
- [ ] 消息发送成功没有被描述为远端处理、持久化或 Exactly Once。
- [ ] Spec、契约源、生成物、实现、测试与受影响文档保持一致。
- [ ] `git diff --check` 通过，最终 diff 只包含任务范围且不泄露敏感信息。
