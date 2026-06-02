# Design: 文档体系优化

## 设计原则

1. **聚焦生产就绪模块**：核心文档覆盖生产可用的服务
2. **分层导航**：README → docs/README.md → 各服务文档，逐层深入
3. **风格统一**：标题、章节结构、代码块格式一致
4. **可维护性**：文档与代码同步更新的约定

## 文档分层

```
Level 0: README.md
  - 项目入口，~150 行
  - 快速了解 + 引导到详细文档

Level 1: docs/README.md
  - 文档索引/导航页
  - 按角色分类：用户 vs 开发者

Level 2: docs/*.md
  - 各主题详细文档
  - 独立完整，可单独阅读
```

## 各文档内容设计

### README.md（精简版 ~150 行）

```markdown
# Zero-Service

基于 go-zero 的工业级微服务脚手架...

## 特性
- 多协议接入（IEC 104/Modbus/MQTT/gRPC/HTTP）
- 数采平台（Kafka/MQTT/gRPC 三协议推送）
- DJI 云平台（Dock3 Cloud API 封装）
- 异步任务调度（asynq + 自研计划任务引擎）
- 实时通信（SocketIO 消息网关）
- 地理信息（H3/GeoHash/围栏/坐标转换）

## 快速开始
### 环境要求
- Go 1.25+
- Redis

### 安装
git clone + go mod tidy

### 启动示例
cd app/trigger && go run trigger.go -f etc/trigger.yaml

## 架构
[简化架构图]
详见 docs/architecture.md

## 核心服务
| 服务 | 说明 | 文档 |
|------|------|------|
| ieccaller | IEC 104 主站 | [链接] |
| trigger | 任务调度 | [链接] |
| djicloud | DJI 云平台 | [链接] |
| ... | ... | ... |

## 文档导航
- [快速开始](docs/quick-start.md)
- [架构概览](docs/architecture.md)
- [服务文档](docs/README.md)

## 技术栈
[简表]

## 参与贡献
详见 CONTRIBUTING.md
```

### docs/README.md（索引页）

```markdown
# 文档索引

## 用户/对接方
- [快速开始](quick-start.md)
- [架构概览](architecture.md)
- [服务端口清单](service-ports.md)
- [错误码规范](error-codes.md)

## 核心服务
- [IEC 104 数采平台](iec104.md)
- [IEC 104 消息对接](iec104-protocol.md)
- [Trigger 服务](trigger.md)
- [SocketIO 实时通信](socketio.md)
- [DJI 云平台](djicloud.md)

## 开发者
- [开发指南](development.md)
- [部署指南](deployment.md)
- [KML/KMZ 指南](kml-kmz-guide.md)
```

### docs/djicloud.md（新增）

```markdown
# DJI 云平台服务

## 概述
djicloud 封装 DJI Dock3 Cloud API...

## 架构
业务系统 -> djicloud gRPC -> common/djisdk -> MQTT Broker -> DJI 设备

## RPC 接口
| 方法 | 说明 |
|------|------|
| SendPropertySet | 属性设置 |
| StartLiveStream | 直播推流 |
| ... | ... |

## 配置
MqttConfig / AckTimeout / UpstreamReply / DangerousOps

## 数据流
[数据流图]
```

### docs/architecture.md（新增）

```markdown
# 架构概览

## 系统架构
[详细架构图]

## 模块依赖
- common/ 提供公共能力
- app/ 承载业务服务
- gtw/ 统一入口
- facade/ 对外接口

## 数据流
- 数采：IEC 104 从站 -> ieccaller -> Kafka -> iecstash -> streamevent -> TDengine
- DJI：业务 -> djicloud -> MQTT -> 设备
- 实时通信：前端 -> socketgtw -> StreamEvent

## 技术选型
- go-zero: 微服务框架
- gRPC: 服务间通信
- Kafka: 消息队列
- Redis: 缓存+任务队列
- TDengine: 时序存储
```

### docs/development.md（新增）

```markdown
# 开发指南

## 环境搭建
整合 local-development-tools.md

## 代码生成
gen.sh 工作流

## 模块扩展约定
- go-zero 服务：先改 .api/.proto，再 gen.sh
- djicloud：proto -> gen.sh -> Logic
- 公共组件：复用 common/

## 调试技巧
- grpcurl 调试 gRPC
- httpie 调试 HTTP
- websocat 调试 WebSocket
```

### docs/deployment.md（新增）

```markdown
# 部署指南

## Docker 部署
docker-compose up -d

## 单服务部署
go build + 运行

## 集群部署
- Nacos 服务发现
- Nginx/gRPC 负载均衡
- Redis Cluster + Kafka 集群

## 配置管理
- 环境变量
- 配置文件
- Nacos 配置中心
```

## 文档风格规范

### 标题
- 一级标题：文档标题
- 二级标题：主要章节
- 三级标题：子章节

### 代码块
- 使用 fenced code block（```）
- 标注语言（bash/go/json/yaml/sql）

### 表格
- 用于结构化信息（接口列表、配置项）
- 保持列对齐

### 链接
- 相对路径：`./iec104.md`
- 锚点：`#section-name`

## 不做的事

1. **不写 aiapp 文档**：仅在 README 简要提及为实验性
2. **不改代码结构**：只优化文档
3. **不删除图片**：docs/images/ 保持
4. **不改 code.md 原文件**：创建 docs/error-codes.md 作为新版本
