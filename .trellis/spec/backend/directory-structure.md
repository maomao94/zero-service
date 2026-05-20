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

## 服务内部结构

单个 go-zero 服务的目录布局、分层职责和新增代码落点见 [`go-zero-conventions.md`](../go-zero-conventions.md)。

核心规则：`.api` / `.proto` 是契约源头 → `gen.sh` 生成框架代码 → `internal/logic/` 写业务编排。

## 命名和风格

- Go 包名、文件名、结构体和函数遵循 Go/go-zero 习惯，禁止套用 Java 风格分层和命名。
- 新文件命名参考相邻实现，不引入同义目录或重复抽象。
- 修改前先阅读相邻 Handler、Logic、svc、model、config、types，复用返回值、错误处理、日志和依赖注入方式。
