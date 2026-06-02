# PRD: 优化项目文档体系

## 背景

当前文档问题：
- README 434 行过长，与 docs/ 内容大量重复
- 部分核心服务缺独立文档（DJI 云平台、SocketIO 优化）
- 混入外部工具文档（mattpocock-skills-guide.md）
- 文档风格不统一，缺少导航索引

项目实际状态（基于代码探索 + 用户确认）：

| 模块 | 状态 | 说明 |
|------|------|------|
| IEC 104 数采平台 | 生产就绪 | ieccaller/iecstash/streamevent，docker-compose 已覆盖 |
| Trigger | 生产就绪 | 异步任务+计划任务双引擎，有 Swagger |
| DJI 云平台 | 生产就绪 | 90+ 逻辑文件零 TODO，完整实现 |
| 文件/GIS/告警/容器 | 生产就绪 | 基础服务 |
| SocketIO | 生产就绪 | socketgtw/socketpush |
| BFF 网关 | 生产就绪 | gtw |
| aiapp/ | 实验性 | aichat/aigtw/aisolo/mcpserver/ssegtw，暂不适合生产 |
| xfusionmock | 生产就绪 | 千寻定位 mock 服务（特定用途） |
| iecagent | 实验性 | 有 TODO 和注释代码 |

## 目标

打造**结构清晰、适合开源**的文档体系，聚焦生产就绪模块。

## 文档体系

### 根目录

| 文档 | 内容 |
|------|------|
| `README.md` | 精简入口：~150 行，特性+架构+快速开始+文档导航 |
| `CONTRIBUTING.md` | 贡献指南（新增） |

### docs/ 目录

```
docs/
├── README.md                    # 文档索引
│
├── # 用户文档
├── quick-start.md               # 快速开始（详细版）
├── architecture.md              # 架构概览
├── service-ports.md             # 端口清单（保留）
├── error-codes.md               # 错误码（从 code.md 迁移）
│
├── # 核心服务
├── iec104.md                    # IEC 104 数采平台（保留优化）
├── iec104-protocol.md           # IEC 104 消息对接（保留优化）
├── trigger.md                   # Trigger 服务（保留优化）
├── socketio.md                  # SocketIO 实时通信（优化）
├── djicloud.md                  # DJI 云平台（新增）
│
├── # 开发者
├── development.md               # 开发指南
├── deployment.md                # 部署指南
│
└── kml-kmz-guide.md             # KML 指南（保留）
```

### 清理项

| 文档 | 处理 |
|------|------|
| `mattpocock-skills-guide.md` | 删除 |
| `ai-solo-smoke-checklist.md` | 删除或移到内部 |
| `code.md` | 迁移到 docs/error-codes.md |

## README 重构

从 434 行精简到 ~150 行：

```markdown
# Zero-Service

一句话介绍

## 特性（5-6 个亮点）

## 快速开始
- 环境要求
- 安装
- 启动示例

## 架构（简化图 + 指向详细文档）

## 核心服务（简表 + 链接）

## 文档导航

## 技术栈

## 参与贡献
```

## 新增文档要点

### docs/djicloud.md
- DJI Cloud API MQTT 封装
- 配置说明（MqttConfig/AckTimeout/UpstreamReply）
- RPC 接口列表
- 数据流图

### docs/architecture.md
- 系统架构详细版
- 模块依赖关系
- 技术选型

### docs/quick-start.md
- 环境要求详细
- 单服务启动
- Docker Compose 启动
- 常见问题

### docs/development.md
- 整合 local-development-tools.md
- 代码生成流程
- 调试技巧

### docs/deployment.md
- Docker 部署
- 集群部署
- 配置管理

## 验收标准

1. README 从 434 行精简到 ~150 行
2. 所有生产就绪服务有文档覆盖
3. README 和 docs/README.md 提供清晰导航
4. 文档风格统一
5. 删除外部工具文档
6. 链接全部有效

## 约束

- AI 模块（aiapp）不单独写文档，仅在 README 简要提及为 Demo
- 保持现有文档核心内容，主要是结构调整和补充
- 图片资源保持在 docs/images/
