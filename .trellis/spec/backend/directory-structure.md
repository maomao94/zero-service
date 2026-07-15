# 目录结构

> 后端目录组织和模块边界。新增能力前先确认应该落在服务内部、公共组件、接口契约、模型、部署或文档目录。

## 顶层模块

| 路径 | 职责 |
| --- | --- |
| `app/` | 核心 go-zero 微服务（19 个），涵盖设备接入、协议转换、文件、GIS、告警、DJI Cloud、PodEngine 等 |
| `aiapp/` | AI 应用服务组（5 个）：OpenAI 兼容聊天、AI 网关、Eino Agent、SSE 网关、MCP Server |
| `socketapp/` | SocketIO 网关和推送服务（2 个） |
| `gtw/` | BFF 网关，聚合 gRPC 后端并提供 HTTP/gRPC Gateway 入口 |
| `facade/` | 对外协议层，当前为 `streamevent` 跨语言 gRPC 协议 |
| `common/` | 跨服务复用能力（42 个包），按通信、工具、领域三层组织 |
| `model/` | 数据库模型和模型生成脚本 |
| `deploy/` | Docker Compose、部署编排和环境相关材料 |
| `docs/`、`swagger/`、`third_party/`、`util/`、`cli/` | 项目文档、Swagger、第三方 proto、工具集和独立 CLI 工具（如 dtui） |

## app/ 服务清单（19 个）

### 协议接入（bridge 系列）

| 服务 | 类型 | 职责 |
| --- | --- | --- |
| `bridgemodbus` | proto (gRPC) | Modbus TCP/RTU 协议接入 |
| `bridgemqtt` | proto (gRPC) | MQTT 设备协议接入 |
| `bridgekafka` | proto (gRPC) | Kafka 消息接入 |
| `bridgedump` | proto (gRPC) | Bridge 数据转储 |
| `bridgegtw` | api (HTTP) | Bridge 网关聚合 |

### 设备与物联

| 服务 | 类型 | 职责 |
| --- | --- | --- |
| `iecagent` | proto (gRPC) | IEC104 从站代理 |
| `ieccaller` | proto (gRPC) | IEC104 主站调用 |
| `iecstash` | proto (gRPC) | IEC104 数据暂存 |
| `ispagent` | proto (gRPC) | ISP 硬件盒子代理，SQLite 本地任务调度与设备模型上报 |
| `djicloud` | proto (gRPC) | DJI Cloud API 物联平台（机巢管理、OSD/State、飞行区、DRC） |

### 基础能力

| 服务 | 类型 | 职责 |
| --- | --- | --- |
| `file` | proto (gRPC) | 通用文件服务（上传/下载） |
| `gis` | proto (gRPC) | GIS 空间计算（点在围栏内、距离计算、最近围栏等） |
| `alarm` | proto (gRPC) | 告警服务 |
| `trigger` | proto (gRPC) | 触发器编排服务 |
| `logdump` | proto (gRPC) | 日志转储 |

### 编排与直播

| 服务 | 类型 | 职责 |
| --- | --- | --- |
| `podengine` | proto (gRPC) | Pod 编排引擎 |
| `lalproxy` | proto (gRPC) | 直播/流媒体代理 |
| `lalhook` | api (HTTP) | 直播/流 hook |

### 测试

| 服务 | 类型 | 职责 |
| --- | --- | --- |
| `xfusionmock` | proto (gRPC) | xFusion 设备模拟（Demo/测试用） |

## aiapp/ 服务清单（5 个）

| 服务 | 类型 | 职责 |
| --- | --- | --- |
| `aichat` | proto (gRPC) | OpenAI 兼容聊天服务 |
| `aigtw` | — | AI 网关，聚合 AI 后端并处理 OpenAI 格式错误 |
| `aisolo` | proto (gRPC) | Eino Agent 独立执行服务 |
| `ssegtw` | api (HTTP) | SSE 流式网关 |
| `mcpserver` | — | MCP Server 实现 |

## socketapp/ 服务清单（2 个）

| 服务 | 类型 | 职责 |
| --- | --- | --- |
| `socketgtw` | — | SocketIO 握手/路由网关 |
| `socketpush` | — | WebSocket 推送服务 |

## common/ 公共包清单（42 个）

### 通信层

| 包 | 职责 |
| --- | --- |
| `socketiox` | SocketIO 服务端/客户端（Session、房间、广播） |
| `mqttx` | MQTT 客户端抽象（Client 接口、ReplyRouter、handler 注册） |
| `wsx` | WebSocket 客户端（状态机、自动重连、认证/心跳） |
| `gnetx` | TCP 框架（Codec、Server、Client、Session、Request-Response） |
| `netx` | HTTP 客户端（Engine 抽象、链式 Request、上传/下载、OTel） |
| `ssex` | SSE 流式输出 |
| `nacosx` | Nacos 服务发现 |
| `asynqx` | asynq 任务队列 |
| `ftps` | FTP/SFTP 相关封装 |

### AI 与协议

| 包 | 职责 |
| --- | --- |
| `einox` | Eino AI Agent 框架封装 |
| `mcpx` | MCP 客户端/服务端 |
| `modbusx` | Modbus 协议工具 |
| `iec104` | IEC104 协议常量与工具 |
| `djisdk` | DJI Cloud SDK（Client、Handler、DRC、Topic、错误处理） |
| `dbx` | 多数据库扩展（多库路由、租户隔离） |

### 工具与基础设施

| 包 | 职责 |
| --- | --- |
| `gormx` | GORM 增强（BaseModel、时间钩子、分页） |
| `gisx` | GIS 空间计算（规划几何、FenceStore 快照、GEOS 绑定） |
| `ossx` | OSS 对象存储（上传、签名 URL） |
| `dockerx` | Docker 操作封装 |
| `gtwx` | 网关错误处理与路由工具 |
| `flowx` | Workflow 封装（构造、拦截器、Step/Attempt 日志） |
| `tool` | 通用工具（错误构建、类型转换等） |
| `Interceptor` | RPC 拦截器（日志、异常恢复等） |
| `configx` | 配置工具 |
| `copierx` | 对象深拷贝 |
| `filex` | 文件操作 |
| `crontask` | cron/task 调度辅助 |
| `imagex` | 图片处理 |
| `mediax` | 媒体处理 |
| `powerwechatx` | 企业微信集成 |
| `ctxprop` | gRPC/JWT/MCP 上下文传播 |
| `ctxdata` | 上下文数据存取 |

### 领域专用

| 包 | 职责 |
| --- | --- |
| `alarmx` | 告警工具 |
| `antsx` | 并行任务编排（Promise、Invoke、ReplyPool） |
| `bytex` | Modbus 字节/寄存器工具 |
| `lalx` | 直播/流媒体工具 |
| `carbonx` | 时间格式化工具 |
| `executorx` | 任务执行器 |
| `skillmd` | 技能元数据 |
| `stream` | 流处理工具 |
| `trace` | 链路追踪 |

## 服务内部结构

单个 go-zero 服务的目录布局、分层职责和新增代码落点见 [`go-zero-conventions.md`](./go-zero-conventions.md)。

核心规则：`.api` / `.proto` 是契约源头 → `gen.sh` 生成框架代码 → `internal/logic/` 写业务编排。

## 新增能力落点判断

决策顺序：
1. **跨多个 app/* 服务使用** → `common/`，按上述分类选最接近的或新建
2. **单个 app/ 服务内多个 logic 使用** → `app/<svc>/internal/` 新建 helper 文件
3. **单个 logic 内使用** → 内联在 logic 文件中

## 反模式

- 不要在 `common/` 新建包后只被一个服务使用（除非明确规划后续复用）。
- 不要在 `app/<svc>/` 的服务之间直接导入对方的 internal 包——必须通过 gRPC 接口调用。
- 不要让 `common/` 包依赖 `app/` 或 `model/`，只能依赖标准库和其他 `common/` 包（同级或子级）。
- 不要将 bridge 协议实现与业务逻辑耦合——bridge 只做协议翻译和路由。

## CLI 工具

```
cli/<name>/
  main.go              # Cobra 入口
  build.sh             # 交叉编译脚本
  README.md            # 使用说明 + Cobra/Bubble Tea 学习文档
  internal/
    cli/                # root command 组装
    docker/             # exec.Command("docker", args...) 调用封装
    tui/                # Bubble Tea Model/Update/View
```

规则：
- 不接入 go-zero，不依赖 `app/` / `common/`。
- Docker 调用用参数数组，禁止 shell 拼接。
- 配置默认 `~/.<name>/config.json`，首次启动自动初始化。

## 命名和风格

- Go 包名、文件名、结构体和函数遵循 Go/go-zero 习惯，禁止套用 Java 风格分层和命名。
- 新文件命名参考相邻实现，不引入同义目录或重复抽象。
- 修改前先阅读相邻 Handler、Logic、svc、model、config、types，复用返回值、错误处理、日志和依赖注入方式。
