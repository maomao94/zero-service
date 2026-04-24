# go-zero 项目约定

> 本项目基于 go-zero 微服务框架，遵循 go-zero 标准架构和约定。

## 目录结构

```
app/{服务名}/
├── {服务名}.proto          # gRPC 服务定义
├── {服务名}.api            # API 网关定义
├── {服务名}.go             # 服务入口
├── gen.sh                  # 代码生成脚本
├── etc/
│   └── {服务名}.yaml       # 服务配置
└── internal/
    ├── config/             # 配置结构体
    ├── logic/              # 业务逻辑（核心编码区域）
    ├── server/             # gRPC/HTTP server
    └── svc/                # 服务依赖注入（ServiceContext）
```

## 三层架构

go-zero 遵循 **Handler → Logic → Model** 三层架构：

1. **Handler/Server**：由 `gen.sh` 自动生成，负责请求解析和路由
2. **Logic**：业务逻辑层，开发者的核心编码区域
3. **Model**：数据模型层，负责数据库交互

## 代码生成命令

```bash
# gRPC 服务
goctl rpc protoc {服务名}.proto --go_out=. --go-grpc_out=. --zrpc_out=. --style=go_zero

# API 网关
goctl api go {服务名}.api --go_out=. --style=go_zero
```

项目使用 `gen.sh` 封装了上述命令，直接执行 `./gen.sh` 即可。

## 服务类型

| 类型 | 定义文件 | 生成命令 | 命名约定 |
| --- | --- | --- | --- |
| API 网关 | `.api` | `goctl api go` | xxxRequest / xxxResponse |
| gRPC 服务 | `.proto` | `goctl rpc protoc` | xxxReq / xxxRes |

## 公共组件

项目 `common/` 目录下包含跨服务复用的公共组件：

- `common/mqttx/` — MQTT 客户端封装
- `common/djisdk/` — DJI SDK 封装
- `common/antsx/` — 响应式异步框架

开发新功能前，先检索 `common/` 目录，确认是否有可复用组件。
