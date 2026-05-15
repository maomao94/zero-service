# go-zero 项目约定

> 本项目基于 go-zero 微服务框架，契约源文件、代码生成和 Logic 分层是后端开发的核心边界。

## 服务目录结构

```text
app/{服务名}/
├── {服务名}.proto          # gRPC 服务定义
├── {服务名}.api            # API 网关定义，部分服务才有
├── {服务名}.go             # 服务入口
├── gen.sh                  # 代码生成脚本
├── etc/
│   └── {服务名}.yaml       # 服务配置
└── internal/
    ├── config/             # 配置结构体
    ├── logic/              # 业务逻辑主落点
    ├── server/ 或 handler/ # 生成的 gRPC/HTTP 入口
    └── svc/                # 依赖注入 ServiceContext
```

`aiapp/`、`socketapp/`、`gtw/`、`facade/` 下的服务遵循同样的契约源文件、生成脚本和 `internal/` 分层思路。

## 三层架构

go-zero 遵循 Handler/Server → Logic → Model/SDK 的分层：

1. Handler/Server：由 `gen.sh` 生成，负责请求解析、路由和响应输出。
2. Logic：业务逻辑层，负责流程编排、调用依赖和处理错误。
3. Model/SDK/Client：负责数据库、缓存、消息队列、MQTT、OSS、Docker、第三方 API 等资源访问。

业务逻辑不要写在 Handler/Server 中；依赖不要绕过 `ServiceContext` 临时创建。

## 代码生成

项目使用各服务目录下的 `gen.sh` 封装 goctl/protoc 命令，通常直接执行：

```bash
cd app/{service}
/usr/bin/env bash gen.sh
```

常见底层命令包括：

```bash
goctl rpc protoc {service}.proto --go_out=. --go-grpc_out=. --zrpc_out=. --style=go_zero
goctl api go {service}.api --go_out=. --style=go_zero
```

修改顺序必须是：

1. 修改 `.api` / `.proto` 契约。
2. 执行对应 `gen.sh`。
3. 检查生成代码 diff。
4. 在 `internal/logic/`、`internal/svc/`、`internal/config/`、model 或 `common/` 中补业务实现。

## 服务类型

| 类型 | 定义文件 | 生成方式 | 命名约定 |
| --- | --- | --- | --- |
| API 网关 | `.api` | `gen.sh` / `goctl api go` | `xxxRequest` / `xxxResponse` |
| gRPC 服务 | `.proto` | `gen.sh` / `goctl rpc protoc` | `xxxReq` / `xxxRes` |
| 对外协议 | `facade/**.proto`、`third_party/**` | 对应目录生成脚本 | 跟随 proto 现有命名 |

## ServiceContext 和配置

- 外部依赖通过 `internal/config` 读取配置，再在 `internal/svc/ServiceContext` 中初始化。
- Logic 通过 `l.svcCtx` 使用依赖，不在函数中临时创建数据库、Redis、MQTT、RPC、OSS、Docker 或 AI Provider 客户端。
- 配置示例只能使用占位值，不提交真实账号、密码、连接串、Token 或内网地址。

## 公共组件

新增能力前先检索 `common/` 和相邻模块，优先复用：

- `common/djisdk/`：DJI Cloud API MQTT topic、协议体、Client 和回调。
- `common/einox/`：Eino Agent、知识库、记忆、中断、协议适配。
- `common/mcpx/`：MCP Server/Client、鉴权、异步结果、工具封装。
- `common/mqttx/`：MQTT 客户端封装。
- `common/ssex/`：SSE Writer 和流式响应工具。
- `common/dbx/`：数据库扩展和多库支持。
- `common/asynqx/`：asynq 任务队列扩展。
- `common/dockerx/`：Docker 操作封装。

只有当能力可跨服务复用且边界稳定时，才新增或扩展 `common/`；单服务私有逻辑保留在该服务 `internal/`。

## 验证命令

- 构建整个项目：`go build ./...`
- 构建特定服务：`go build ./app/trigger/...`
- 运行相关测试：`go test -v ./model/...`、`go test -v ./app/trigger/...`
- 运行特定测试：`go test ./model -run TestPlanModel_Insert -v`
- 依赖变更后整理：`go mod tidy`

优先运行与变更相关的最小命令；跨公共组件或契约变更时扩大验证范围。
