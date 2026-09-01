# 核心规则（所有任务必读）

> 本文件提取项目特有的、非显而易见的核心规则。小任务：读本文件 + 1个相关spec。

## 快速检查清单

- [ ] 改动前搜索了定义、调用、测试和配置
- [ ] 不顺带重命名/格式化/升级/重构无关模块
- [ ] 错误有明确处理路径（返回/包装/边界映射）
- [ ] goroutine 有退出策略
- [ ] 共享状态有保护机制
- [ ] 日志不泄露敏感信息

## 沟通与修改

| 规则 | 说明 |
|------|------|
| 语言 | 中文沟通；Go标识符、协议字段、路径保留原文 |
| 证据优先 | 修改前搜索定义、调用、测试；对身份/状态/字段所有权给代码证据 |
| 改动范围 | 紧贴需求，不顺带重命名、全仓格式化、升级依赖 |
| 风格一致 | 遵循相邻包构造/错误/测试风格；只有降低跨调用方复杂度时才抽象 |
| 结构化解析 | JSON/XML/Proto/URL/SQL/时间用解析器，不手工拼接 |

## Go 核心

| 规则 | 说明 |
|------|------|
| context | `context.Context` 为第一参数；不用 `context.Background()` 截断链路（后台任务除外） |
| 错误 | 必须返回/包装/边界处理；包装用 `%w`，确保 `errors.Is`/`errors.As` 可识别 |
| goroutine | 必须有退出/等待/所有权策略；异步接口明确是排队成功/处理成功/best-effort |
| 共享状态 | 写清由哪把锁/事务/CAS保护；锁外执行网络/数据库/回调等慢操作 |
| 泛型 | 消除真实重复时使用，约束放在使用它的包内 |

详见：[concurrency-guidelines.md](./concurrency-guidelines.md)

## 命名与边界

| 规则 | 说明 |
|------|------|
| 请求/响应 | `.api` 用 `XxxRequest`/`XxxResponse`；gRPC 用 `XxxReq`/`XxxRes` |
| 身份ID | `SessionID`（连接级）、业务设备ID、消息关联ID、任务ID 用语义明确名字 |
| option | 公共 client/SDK 的 option 修改构造配置对象，由构造函数生成运行态对象 |
| 错误边界 | 领域错误留在领域/公共包，传输格式只在边界映射 |

详见：[error-handling.md](./error-handling.md)

## 错误处理

| 规则 | 说明 |
|------|------|
| sentinel/typed error | 领域/公共包定义传输中立错误；包装保留原 cause |
| 边界映射 | gRPC status、HTTP body、OpenAI body、MCP 错误只在对应边界映射 |
| 日志集中 | server interceptor 集中记录返回错误；Logic 只记录业务定位价值信息 |
| 日志脱敏 | 保留 trace 和业务标识，不记录认证头/Token/连接串/大 payload |

详见：[error-handling.md](./error-handling.md)

## go-zero 分层

| 层 | 职责 | 不做 |
|----|------|------|
| Handler/Server | 接收传输参数、调用 Logic、返回结果 | 业务编排、事务、外部调用 |
| Logic | 请求级业务流程、校验、多依赖协调 | 复制 SQL/topic/帧操作 |
| ServiceContext | 创建共享 client/store/scheduler | 保存请求级状态 |
| Model/store/SDK | 数据或外部系统边界 | — |

- 依赖方向：传输 → Logic → store/公共包；`common/` 不反向依赖具体服务 `internal/`
- 获取 trace ID 统一用 `trace.TraceIDFromContext(ctx)`

详见：[go-zero-conventions.md](./go-zero-conventions.md)、[service-lifecycle.md](./service-lifecycle.md)

## GORM 与数据访问

| 规则 | 说明 |
|------|------|
| 模型组合 | 按需组合原子 mixin：`IDModel`/`TimeMixin`/`SoftDeleteMixin`/`VersionMixin`/`TenantMixin` |
| 职责分离 | Store 拥有 SQL/事务/字段更新；Logic 传递领域参数，不拼列名 |
| 条件更新 | 必须检查 error 和 `RowsAffected`；幂等成功/目标不存在/竞争失败是不同契约 |
| 并发更新 | 用事务/唯一约束/版本/CAS；完成路径只更新自己拥有的字段 |

详见：[gormx-guidelines.md](./gormx-guidelines.md)

## 并发工具选择

| 场景 | 项目选择 |
|------|---------|
| 同一请求内少量并行 | go-zero `mr.Finish` / `MapReduce` |
| typed并行、需保持顺序 | `antsx.Invoke` |
| 需限制并发度 | `antsx.Reactor` |
| 单个未来结果 | `antsx.Promise`（带可取消context） |
| correlation ID应答 | `antsx.ReplyPool` / `mqttx.ReplyRouter` |
| Redis后台任务队列 | `asynqx` |

不要为简单同步流程引入 Promise，也不要用裸 goroutine 重写已有并发组件。

详见：[concurrency-guidelines.md](./concurrency-guidelines.md)

## 公共包设计

| 规则 | 说明 |
|------|------|
| 进入条件 | 明确跨服务复用、可脱离具体业务 proto/model 独立描述 |
| 依赖注入 | 必需依赖用构造参数，行为变体用小接口/函数，非必需用 function option |
| option模式 | 写入配置结构，由构造函数校验并生成运行态对象；option 不直接改锁/连接/缓存 |
| 生命周期 | 长期资源暴露幂等 `Close`/`Stop` 或由 context 管理 |

详见：[common-package-design.md](./common-package-design.md)、[service-lifecycle.md](./service-lifecycle.md)

## 安全基线

| 规则 | 说明 |
|------|------|
| 不泄露 | 不新增/复制/打印/提交真实密码、Token、认证头、证书、连接串、内网地址 |
| 占位值 | 配置示例用占位值；日志对 payload 与认证信息默认脱敏 |
| 配置注入 | 不在代码硬编码环境地址或凭据；通过配置结构注入 |
| 边界校验 | 外部输入在进入数据库/文件系统/反射工具/二进制解析/网络转发前校验 |

详见：[quality-guidelines.md](./quality-guidelines.md)
