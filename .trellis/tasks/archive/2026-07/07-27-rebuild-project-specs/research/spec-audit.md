# Spec 审计与迁移映射

## 仓库结论

- 单个 Go module 承载多个 RPC、API、网关、异步任务和协议接入服务。
- 服务普遍遵循 go-zero 的契约生成、`internal/logic` 业务实现和 `ServiceContext` 依赖装配。
- `common/` 同时包含轻量工具和具有状态/并发/协议语义的基础设施，不能按“一包一 Spec”机械展开。
- 高风险规则集中在调度、数据访问、关联响应、连接身份、二进制协议、空间计算和实时事件契约。
- 当前 47 个 Spec 共约 1.1 万行，存在目录职责混杂、API 手册化、主题重复和实现漂移。

## 迁移映射

| 旧主题 | 标准结构中的新位置 | 处理 |
| --- | --- | --- |
| `directory-structure`、`coding-standards`、`quality-guidelines` | `backend/directory-structure.md`、`coding-standards.md`、`quality-guidelines.md` | 保留仓库级 Code-Spec，按索引分组而不另造 `project` layer |
| `go-zero-conventions` | `backend/go-zero-conventions.md` | 保留分层和生成边界，删除泛化框架说明 |
| `ctxprop`、`error-handling`、`logging` | `backend/error-handling.md` | 按请求链路合并，区分领域错误与传输映射 |
| `database`、`gormx` | `backend/gormx-guidelines.md` | 以当前 GORM 封装和测试为准，移除过时禁令 |
| `antsx-*`、`mr-concurrency`、`flowx` | `backend/concurrency-guidelines.md` | 按并发语义和生命周期合并，不列完整 API |
| `netx`、`wsx`、`mqttx`、`messaging` | `backend/messaging-guidelines.md` 与 `realtime-guidelines.md` | 公共客户端规则和业务事件契约分开 |
| `crontask` | `backend/crontask-guidelines.md` | 保留 lease/CAS、状态和存储语义 |
| `isp` | `backend/isp-guidelines.md` | 压缩报文、身份、注册、人工执行和回执契约 |
| `iec104-control`、`iec104-trace` | `backend/iec104-guidelines.md` | 合并控制应答、路由和追踪链路 |
| `djisdk`、`djicloud-hooks`、`djicloud-models`、`drc-concurrency` | `backend/dji-guidelines.md` | 按 SDK/应用/持久化边界合并 |
| `gisx`、`geofence` | `backend/gis-guidelines.md` | 合并坐标、GEOS、H3 和事务所有权 |
| `socketiox-*` | `backend/realtime-guidelines.md` | 只保留外部事件和当前交付语义 |
| `einox`、`mcpx` | `backend/ai-guidelines.md` | 保留能力、会话状态和 MCP 生命周期 |
| `bytex` | `backend/common-package-design.md` | 仅保留二进制边界、安全解码和错误返回总则 |
| `code-reuse`、`cross-layer`、`documentation` | `guides/*-guide.md` | 只保留问题清单和 Code-Spec 路由，不承载实现规则 |

## 删除且不迁移

- `backend/gnetx/`：实验/原型组件，不建立独立 Spec。
- `drone-station-sdk-template.md`：模板说明，不是稳定项目契约。
- `trellis-template-policy.md`：Trellis 元配置，不属于产品编码规范。
- `release-tagging-guide.md`：发布操作说明，与本次代码开发 Spec 目标无关。
- 旧文档中的版本号抄录、完整 API 表、长代码示例和历史排障叙述。

## 已发现的漂移

- 旧数据库规范笼统禁止 `clause.OnConflict`，当前 `common/gormx/upsert.go` 已提供统一的 `Upsert` 帮助函数；新规范应要求复用项目封装。
- 个别旧规范复制了依赖版本或实现细节，当前源码已经变化；新规范只指向 `go.mod` 和对应实现，不固定易漂移版本。
- 旧日志描述超出当前拦截器实现；新规范只记录上下文传播、边界映射和集中记录职责。
- 旧 Spec 将稳定 ISP 使用和实验性 `gnetx` 框架捆绑；新规范只记录 ISP 实际依赖的接口与身份语义。

## 证据范围

- 项目结构：`README.md`、`CONTRIBUTING.md`、`docs/architecture.md`、`docs/development.md`。
- 服务模式：`app/trigger/trigger.go`、各服务 `gen.sh`、`internal/logic`、`internal/svc/servicecontext.go`。
- 公共基础设施：`common/crontask`、`common/gormx`、`common/antsx`、`common/mqttx`、`common/netx`、`common/wsx` 及测试。
- 领域契约：`app/trigger`、`common/isp`、`app/ispagent`、`common/iec104`、`app/ieccaller`、`common/djisdk`、`app/djicloud`、`common/gisx`、`app/gis`、`socketapp`、`facade/streamevent`、`common/einox`、`common/mcpx`。
