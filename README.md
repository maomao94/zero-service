# Zero-Service

[![Go](https://github.com/maomao94/zero-service/actions/workflows/go.yml/badge.svg)](https://github.com/maomao94/zero-service/actions/workflows/go.yml)
[![Docs](https://github.com/maomao94/zero-service/actions/workflows/docs.yml/badge.svg)](https://github.com/maomao94/zero-service/actions/workflows/docs.yml)
[![Release](https://img.shields.io/github/v/release/maomao94/zero-service?sort=semver)](https://github.com/maomao94/zero-service/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/maomao94/zero-service)](https://go.dev/)
[![License](https://img.shields.io/github/license/maomao94/zero-service)](./LICENSE)

面向工业物联网与边缘集成场景的 Go 微服务集合。基于 [go-zero](https://github.com/zeromicro/go-zero) 构建，覆盖工业协议接入、流数据处理、任务调度、实时通信和无人机机场等能力。

Zero-Service 不是必须整体部署的单体应用：各服务可独立运行，也可通过 gRPC、Kafka、MQTT 和 SocketIO 组合成完整业务链路。

## 目录

- [核心能力](#核心能力)
- [架构概览](#架构概览)
- [快速开始](#快速开始)
- [文档](#文档)
- [仓库结构](#仓库结构)
- [参与贡献](#参与贡献)
- [版本与变更](#版本与变更)
- [开源许可](#开源许可)

## 核心能力

- **视频会议**：基于 LiveKit 的实时音视频会议，提供会议管理、票据邀请、Webhook 状态同步、Egress 录制与聊天消息。
- **IEC 104 数采**：多从站通信，通过 Kafka、MQTT 和 gRPC 并行分发采集数据，支持 ASDU 合并与时序存储。
- **DJI 云平台接入**：封装第一代上云 API（Dock 3 Cloud API）的 MQTT 协议，支持航线任务、直播推流和 DRC 指令飞行。新项目建议评估[上云 API 2](https://developer.dji.com/cn/cloud-api-2)，本实现保留作协议学习与存量系统参考。
- **任务调度**：asynq 异步回调、Plan/Batch/ExecItem 计划任务，以及基于 RRULE 的 CronJob 周期调度。
- **实时通信**：SocketIO 网关负责连接管理、房间广播、服务端推送与 MQTT 桥接。
- **工业协议桥接**：Modbus TCP/RTU、MQTT、Kafka、gRPC-Gateway 及反向隔离装置接入。
- **变电站巡检**：ISP 协议服务端与代理，连接上级平台和下级巡检设备。
- **开放 API 网关**：面向第三方应用提供 Basic 鉴权、请求签名、统一 JSON 信封与 method 路由。
- **通用基础能力**：地理信息计算、对象存储、文件传输、容器管理、服务发现与可观测性组件。

## 架构概览

```text
设备 / 第三方系统 / 前端应用
              |
   +----------+-----------+
   |                      |
协议接入与 BFF       SocketIO 实时网关
   |                      |
   +----------+-----------+
              |
    领域服务与协议桥接服务
              |
   +----------+-----------+
   |          |           |
 Kafka      Redis      数据库 / 对象存储
```

系统以 gRPC 作为主要服务间通信协议，通过 Kafka、MQTT、SocketIO 适配不同实时性与吞吐需求。分层结构与核心数据流见[架构文档](./docs/architecture.md)。

## 快速开始

### 环境要求

- Go 1.26 或更高版本（以 `go.mod` 为准）
- Git
- 与目标服务匹配的外部依赖，例如 Redis、Kafka、数据库或 MQTT Broker

### 获取代码

```bash
git clone https://github.com/maomao94/zero-service.git
cd zero-service
go mod download
```

### 启动服务

各服务均可独立运行。以 Trigger 为例，先按本地环境调整 `app/trigger/etc/trigger.yaml` 中的 Redis 与数据库配置：

```bash
cd app/trigger
go run . -f etc/trigger.yaml
```

### 验证

```bash
# Trigger 开发模式默认开启 gRPC reflection
grpcurl -plaintext localhost:21006 list

# gtw 已配置对应路由时检查 HTTP 入口
curl http://localhost:11001/health
```

更多启动方式、依赖矩阵和常见问题见[快速开始指南](./docs/quick-start.md)，端口分配见[服务端口清单](./docs/service-ports.md)。

## 文档

[文档索引](./docs/README.md)是唯一入口，按使用场景分类导航。

| 文档 | 内容 |
| --- | --- |
| [快速开始](./docs/quick-start.md) | 环境准备、服务启动、验证与常见问题 |
| [架构概览](./docs/architecture.md) | 系统分层、模块依赖、核心数据流与技术选型 |
| [服务端口清单](./docs/service-ports.md) | 各服务默认端口、协议与用途 |
| [网关标准开发流程](./docs/gateway-standard.md) | 分层、路由鉴权、统一信封、错误码出口与验收清单 |
| [错误码规范](./docs/error-codes.md) | 错误码维度、模块分配、HTTP/gRPC 映射与折叠策略 |
| [开发指南](./docs/development.md) | 代码生成、模块扩展与调试方式 |
| [部署指南](./docs/deployment.md) | Docker、单服务与集群部署 |
| [部署模板总览](./deploy/README.md) | 数据库、中间件、网关与运维工具的本地编排 |

## 仓库结构

```text
app/         核心业务、视频会议、流媒体与协议服务
aiapp/       AI 应用与模型接入服务
socketapp/   SocketIO 网关与推送服务
gtw/         存量 BFF HTTP 网关
zerorpc/     核心业务 gRPC 服务
common/      跨服务复用的公共组件
facade/      跨服务协议与数据契约
model/       数据模型及生成脚本
deploy/      Docker Compose 与部署配置
web/         前端工作台与子系统（Live / SocketIO / OpenAPI）
cli/         Docker TUI 等命令行工具
util/        运维脚本与 Taskfile
swagger/     生成的服务 Swagger 描述
third_party/ protobuf 依赖与代码生成插件
1.7.1/、1.9.x/  goctl 代码生成模板（按版本固定，供 model/ 下的脚本使用）
docs/        架构、对接和开发文档
scripts/     仓库维护脚本（文档校验等）
```

## 参与贡献

- 提交代码前请阅读[贡献指南](./.github/CONTRIBUTING.md)：分支模型、提交规范、代码生成与 PR 要求。
- 报告安全漏洞请走[安全策略](./.github/SECURITY.md)中的私密通道，不要公开提 Issue。
- 使用与对接问题先查[文档索引](./docs/README.md)，仍无法解决再提 Issue。

开发自检：

```bash
# 只检查本次改动的 Go 文件；仓库仍有历史未格式化文件，全量 gofmt -l . 会有输出
for f in $(git diff --name-only --diff-filter=ACMR origin/develop...HEAD -- '*.go'); do gofmt -l "$f"; done
go vet ./...
go test -race ./...
python3 scripts/check_docs.py   # 改动 Markdown 时
git diff --check                # 应无输出
```

## 版本与变更

版本号遵循[语义化版本](https://semver.org/lang/zh-CN/)，发布记录见 [Releases](https://github.com/maomao94/zero-service/releases)。分支模型为 Git Flow：`master` 为稳定发布分支，`develop` 为集成分支。

## 开源许可

本项目基于 [MIT License](./LICENSE) 开源。
