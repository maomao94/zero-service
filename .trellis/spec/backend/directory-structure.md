# 目录结构

> 后端目录组织和模块边界。新增能力前先确认应该落在服务内部、公共组件、接口契约、模型、部署或文档目录。

## 顶层模块

| 路径 | 职责 |
| --- | --- |
| `app/` | 核心 go-zero 微服务，例如 IEC 104、Trigger、文件、GIS、告警、Modbus/MQTT、DJI Cloud、PodEngine 等 |
| `aiapp/` | AI 应用服务组，包含 OpenAI 兼容聊天、AI 网关、Eino Agent、SSE 网关和 MCP Server |
| `socketapp/` | SocketIO 网关和推送服务 |
| `gtw/` | BFF 网关，聚合 gRPC 后端并提供 HTTP/gRPC Gateway 入口 |
| `facade/` | 对外协议层，当前重点是 `streamevent` 跨语言 gRPC 协议 |
| `common/` | 跨服务复用能力，例如 `djisdk`、`einox`、`mcpx`、`mqttx`、`dbx`、`ssex`、`asynqx`、`dockerx` |
| `model/` | 数据库模型和模型生成脚本 |
| `deploy/` | Docker Compose、部署编排和环境相关材料 |
| `docs/`、`swagger/`、`third_party/`、`util/` | 项目文档、Swagger、第三方 proto 和工具集 |

## go-zero 服务布局

典型服务遵循以下结构：

```text
app/{service}/
├── {service}.api / {service}.proto
├── {service}.go
├── gen.sh
├── etc/
│   └── {service}.yaml
└── internal/
    ├── config/
    ├── logic/
    ├── server/ 或 handler/
    └── svc/
```

- `.api` 和 `.proto` 是接口契约源头。
- `gen.sh` 负责生成 Handler、Server、Types、pb 和路由等框架代码。
- `internal/logic/` 是业务编排主落点。
- `internal/svc/ServiceContext` 负责依赖注入和资源装配。
- `internal/config/` 保存配置结构，不硬编码端口、连接串、密钥或环境参数。

## 分层职责

- Handler/Server：解析请求、基础校验、调用 Logic、返回结果；不要写业务编排。
- Logic：承载业务流程、调用 model/client/cache/SDK/common 工具，并保持上下文 `context.Context` 传递。
- Model/SDK/Client：封装数据库、缓存、消息队列、MQTT、OSS、Docker、第三方 API 等外部资源访问。
- `common/`：只放跨服务复用且边界清晰的能力；单服务有状态逻辑保留在该服务 `internal/`。

## 新增能力落点

1. 新接口或 RPC：先改目标服务的 `.api` / `.proto`。
2. 业务逻辑：放入目标服务 `internal/logic/`。
3. 依赖装配：放入 `internal/svc/` 和 `internal/config/`。
4. 跨服务通用协议、SDK、工具：优先放到 `common/` 的既有子包；新增子包前先确认没有可扩展封装。
5. 数据模型：放到 `model/` 或服务既有 model 位置，并使用项目已有生成脚本。
6. 独立 SQL：放到项目约定 SQL 目录，文件名关联 Trellis 任务或需求号。

## 命名和风格

- Go 包名、文件名、结构体和函数遵循 Go/go-zero 习惯，禁止套用 Java 风格分层和命名。
- 新文件命名参考相邻实现，不引入同义目录或重复抽象。
- 修改前先阅读相邻 Handler、Logic、svc、model、config、types，复用返回值、错误处理、日志和依赖注入方式。
