# SSE网关服务

<cite>
**本文引用的文件**
- [aiapp/ssegtw/ssegtw.go](file://aiapp/ssegtw/ssegtw.go)
- [aiapp/ssegtw/etc/ssegtw.yaml](file://aiapp/ssegtw/etc/ssegtw.yaml)
- [aiapp/ssegtw/internal/config/config.go](file://aiapp/ssegtw/internal/config/config.go)
- [aiapp/ssegtw/ssegtw.api](file://aiapp/ssegtw/ssegtw.api)
- [aiapp/ssegtw/internal/handler/routes.go](file://aiapp/ssegtw/internal/handler/routes.go)
- [aiapp/ssegtw/internal/handler/sse/ssestreamhandler.go](file://aiapp/ssegtw/internal/handler/sse/ssestreamhandler.go)
- [aiapp/ssegtw/internal/handler/sse/chatstreamhandler.go](file://aiapp/ssegtw/internal/handler/sse/chatstreamhandler.go)
- [aiapp/ssegtw/internal/logic/sse/ssestreamlogic.go](file://aiapp/ssegtw/internal/logic/sse/ssestreamlogic.go)
- [aiapp/ssegtw/internal/logic/sse/chatstreamlogic.go](file://aiapp/ssegtw/internal/logic/sse/chatstreamlogic.go)
- [aiapp/ssegtw/internal/types/types.go](file://aiapp/ssegtw/internal/types/types.go)
- [aiapp/ssegtw/internal/svc/servicecontext.go](file://aiapp/ssegtw/internal/svc/servicecontext.go)
- [common/ssex/writer.go](file://common/ssex/writer.go)
- [aiapp/aigtw/aigtw.go](file://aiapp/aigtw/aigtw.go)
- [aiapp/aigtw/internal/handler/routes.go](file://aiapp/aigtw/internal/handler/routes.go)
- [aiapp/aigtw/aigtw.api](file://aiapp/aigtw/aigtw.api)
- [aiapp/aigtw/chat.html](file://aiapp/aigtw/chat.html)
- [common/antsx/README.md](file://common/antsx/README.md)
- [common/antsx/promise.go](file://common/antsx/promise.go)
- [common/antsx/invoke.go](file://common/antsx/invoke.go)
- [common/antsx/reactor.go](file://common/antsx/reactor.go)
- [common/antsx/pending.go](file://common/antsx/pending.go)
- [common/antsx/emitter.go](file://common/antsx/emitter.go)
- [common/antsx/errors.go](file://common/antsx/errors.go)
- [common/antsx/promise_ext.go](file://common/antsx/promise_ext.go)
- [common/antsx/antsx_test.go](file://common/antsx/antsx_test.go)
- [common/antsx/invoke_test.go](file://common/antsx/invoke_test.go)
- [common/antsx/pending_test.go](file://common/antsx/pending_test.go)
</cite>

## 更新摘要
**变更内容**
- 基于应用变更：SSE网关服务的antsx库文档大幅改进，提供了详细的场景选择指南、架构指导和使用示例
- 新增了完整的antsx库文档，包含Promise、Invoke、Reactor、PendingRegistry、EventEmitter等模式的详细说明
- 增强了SSE网关服务的并发处理能力和架构设计指导
- 提供了丰富的实际应用场景和最佳实践案例

## 目录
1. [简介](#简介)
2. [项目结构](#项目结构)
3. [核心组件](#核心组件)
4. [架构总览](#架构总览)
5. [详细组件分析](#详细组件分析)
6. [antsx库架构模式详解](#antsx库架构模式详解)
7. [依赖分析](#依赖分析)
8. [性能考虑](#性能考虑)
9. [故障排查指南](#故障排查指南)
10. [结论](#结论)
11. [附录](#附录)

## 简介
本项目为一个基于 Go-Zero 的 Server-Sent Events（SSE）网关服务，提供两类实时流能力：
- SSE 事件流：用于推送系统通知与状态变更等事件。
- AI 对话流：以流式方式逐字返回生成的 token，模拟大模型推理过程。

系统采用事件驱动架构，通过内置的事件发射器与等待完成注册机制，实现"生产者-消费者"解耦的长连接流式传输，并内置心跳保活与优雅断开逻辑。同时提供跨域配置、健康检查接口与本地调试页面，便于开发与联调。

**更新** 项目现已扩展为两个主要服务：
- SSE 网关服务（aiapp/ssegtw）：提供基础的 SSE 事件流和 AI 对话流功能
- AI 网关服务（aiapp/aigtw）：提供完整的聊天界面和 RESTful API 接口

**更新** 基于antsx库的架构模式：
- 基于 ants goroutine 池的响应式编程工具包，提供并发安全的异步处理能力
- 支持 Promise、Invoke、Reactor、PendingRegistry、EventEmitter 等多种架构模式
- 提供完整的场景选择指南和最佳实践指导

## 项目结构
SSE 网关服务位于 aiapp/ssegtw 目录，AI 网关服务位于 aiapp/aigtw 目录，主要由以下层次组成：
- 入口与配置：应用入口、配置加载、跨域与服务启动
- 路由与处理器：REST 路由注册、SSE 路由启用、HTTP 处理器
- 业务逻辑：SSE 事件流与 AI 对话流的具体实现
- 类型定义：请求与响应类型
- 服务上下文：RPC 客户端、事件发射器、等待完成注册
- SSE 写入器：封装 SSE 协议写入与自动刷新
- 静态文件服务：提供聊天界面 HTML 文件
- **更新** antsx库：提供并发安全的响应式编程工具包

```mermaid
graph TB
A["SSE入口程序<br/>aiapp/ssegtw/ssegtw.go"] --> B["SSE配置加载<br/>aiapp/ssegtw/etc/ssegtw.yaml"]
A --> C["SSE路由注册<br/>aiapp/ssegtw/internal/handler/routes.go"]
C --> D["SSE事件流处理器<br/>internal/handler/sse/ssestreamhandler.go"]
C --> E["AI对话流处理器<br/>internal/handler/sse/chatstreamhandler.go"]
D --> F["SSE事件流逻辑<br/>internal/logic/sse/ssestreamlogic.go"]
E --> G["AI对话流逻辑<br/>internal/logic/sse/chatstreamlogic.go"]
F --> H["SSE写入器<br/>common/ssex/writer.go"]
G --> H
F --> I["SSE服务上下文<br/>internal/svc/servicecontext.go"]
G --> I
J["AI入口程序<br/>aiapp/aigtw/aigtw.go"] --> K["AI配置加载<br/>aiapp/aigtw/etc/aigtw.yaml"]
J --> L["AI路由注册<br/>aiapp/aigtw/internal/handler/routes.go"]
L --> M["聊天界面处理器<br/>aiapp/aigtw/chat.html"]
L --> N["模型列表处理器<br/>internal/handler/ai/listModelsHandler.go"]
L --> O["对话补全处理器<br/>internal/handler/ai/chatCompletionsHandler.go"]
N --> P["模型列表逻辑<br/>internal/logic/ai/listModelsLogic.go"]
O --> Q["对话补全逻辑<br/>internal/logic/ai/chatCompletionsLogic.go"]
Q --> R["AI服务上下文<br/>internal/svc/servicecontext.go"]
S["antsx库文档<br/>common/antsx/README.md"] --> T["Promise模式<br/>common/antsx/promise.go"]
S --> U["Invoke模式<br/>common/antsx/invoke.go"]
S --> V["Reactor模式<br/>common/antsx/reactor.go"]
S --> W["PendingRegistry模式<br/>common/antsx/pending.go"]
S --> X["EventEmitter模式<br/>common/antsx/emitter.go"]
T --> Y["异步结果容器"]
U --> Z["并行流程编排"]
V --> AA["池调度器"]
W --> AB["请求-响应匹配"]
X --> AC["发布/订阅"]
```

**图表来源**
- [aiapp/ssegtw/ssegtw.go:26-59](file://aiapp/ssegtw/ssegtw.go#L26-L59)
- [aiapp/aigtw/aigtw.go:46-65](file://aiapp/aigtw/aigtw.go#L46-L65)
- [aiapp/ssegtw/internal/handler/routes.go:17-50](file://aiapp/ssegtw/internal/handler/routes.go#L17-L50)
- [aiapp/aigtw/internal/handler/routes.go:17-55](file://aiapp/aigtw/internal/handler/routes.go#L17-L55)
- [common/antsx/README.md:10-438](file://common/antsx/README.md#L10-L438)

**章节来源**
- [aiapp/ssegtw/ssegtw.go:26-59](file://aiapp/ssegtw/ssegtw.go#L26-L59)
- [aiapp/aigtw/aigtw.go:29-75](file://aiapp/aigtw/aigtw.go#L29-L75)
- [aiapp/ssegtw/etc/ssegtw.yaml:1-14](file://aiapp/ssegtw/etc/ssegtw.yaml#L1-L14)
- [aiapp/aigtw/etc/aigtw.yaml:1-15](file://aiapp/aigtw/etc/aigtw.yaml#L1-L15)
- [aiapp/ssegtw/internal/config/config.go:11-14](file://aiapp/ssegtw/internal/config/config.go#L11-L14)
- [aiapp/aigtw/internal/config/config.go:20-25](file://aiapp/aigtw/internal/config/config.go#L20-L25)
- [aiapp/ssegtw/ssegtw.api:13-38](file://aiapp/ssegtw/ssegtw.api#L13-L38)
- [aiapp/aigtw/aigtw.api:14-47](file://aiapp/aigtw/aigtw.api#L14-L47)
- [aiapp/ssegtw/internal/handler/routes.go:17-50](file://aiapp/ssegtw/internal/handler/routes.go#L17-L50)
- [aiapp/aigtw/internal/handler/routes.go:17-55](file://aiapp/aigtw/internal/handler/routes.go#L17-L55)
- [aiapp/ssegtw/internal/types/types.go:6-17](file://aiapp/ssegtw/internal/types/types.go#L6-L17)
- [aiapp/aigtw/internal/types/types.go:14-91](file://aiapp/aigtw/internal/types/types.go#L14-L91)
- [aiapp/ssegtw/internal/svc/servicecontext.go:23-38](file://aiapp/ssegtw/internal/svc/servicecontext.go#L23-L38)
- [aiapp/aigtw/internal/svc/servicecontext.go](file://aiapp/aigtw/internal/svc/servicecontext.go)

## 核心组件
- 应用入口与服务启动
  - 解析配置文件，打印运行环境信息，创建 REST 服务器并启用自定义 CORS，注册路由，加入服务组并启动。
- 路由与 SSE 启用
  - 在 SSE 路由组中注册两条 POST 接口，分别对应 SSE 事件流与 AI 对话流，并通过 rest.WithSSE() 启用长连接模式。
- 处理器层
  - 从请求解析出结构化参数，构造逻辑对象并调用对应逻辑方法。
- 逻辑层
  - 初始化 SSE 写入器，确定 channel（可选），注册完成信号，订阅事件通道，发送连接成功事件，启动模拟 worker 推送事件或 token，主循环转发事件并周期性发送心跳。
- 服务上下文
  - 提供 RPC 客户端、事件发射器与等待完成注册，统一注入到各逻辑层。
- SSE 写入器
  - 封装 SSE 协议写入（事件名、数据、注释与心跳），并自动触发 flush。
- 静态文件服务
  - 提供聊天界面 HTML 文件的静态服务，支持多种路径查找策略。
- **更新** antsx库模式
  - Promise：异步结果容器，支持链式调用和错误处理
  - Invoke：并行流程编排，支持超时控制和快速失败
  - Reactor：ants池调度器，带ID去重和goroutine池化
  - PendingRegistry：关联ID请求-响应匹配，支持TTL和自动过期
  - EventEmitter：Topic级别发布/订阅，支持非阻塞广播

**更新** AI 网关服务新增了完整的聊天界面功能：
- 提供美观的 Web 界面，支持深色/浅色主题切换
- 支持多语言（中英文）
- 实时流式聊天，支持深度思考过程展示
- 代码高亮和复制功能
- 响应式设计，支持移动端访问

**章节来源**
- [aiapp/ssegtw/ssegtw.go:26-59](file://aiapp/ssegtw/ssegtw.go#L26-L59)
- [aiapp/aigtw/aigtw.go:46-65](file://aiapp/aigtw/aigtw.go#L46-L65)
- [aiapp/ssegtw/internal/handler/routes.go:17-50](file://aiapp/ssegtw/internal/handler/routes.go#L17-L50)
- [aiapp/aigtw/internal/handler/routes.go:17-55](file://aiapp/aigtw/internal/handler/routes.go#L17-L55)
- [aiapp/ssegtw/internal/handler/sse/ssestreamhandler.go:18-32](file://aiapp/ssegtw/internal/handler/sse/ssestreamhandler.go#L18-L32)
- [aiapp/ssegtw/internal/handler/sse/chatstreamhandler.go:18-32](file://aiapp/ssegtw/internal/handler/sse/chatstreamhandler.go#L18-L32)
- [aiapp/ssegtw/internal/logic/sse/ssestreamlogic.go:39-116](file://aiapp/ssegtw/internal/logic/sse/ssestreamlogic.go#L39-L116)
- [aiapp/ssegtw/internal/logic/sse/chatstreamlogic.go:39-120](file://aiapp/ssegtw/internal/logic/sse/chatstreamlogic.go#L39-L120)
- [aiapp/ssegtw/internal/svc/servicecontext.go:23-38](file://aiapp/ssegtw/internal/svc/servicecontext.go#L23-L38)
- [common/ssex/writer.go:14-54](file://common/ssex/writer.go#L14-L54)
- [common/antsx/README.md:41-438](file://common/antsx/README.md#L41-L438)

## 架构总览
SSE 网关采用"请求-处理-事件发射-客户端推送"的链路，结合"等待完成注册"实现流结束的优雅控制；心跳保活确保长连接稳定。

```mermaid
sequenceDiagram
participant Client as "客户端"
participant Handler as "HTTP处理器"
participant Logic as "业务逻辑"
participant Emitter as "事件发射器"
participant Reg as "等待完成注册"
participant Writer as "SSE写入器"
Client->>Handler : "POST /sse/stream 或 /chat/stream"
Handler->>Logic : "解析请求并构造逻辑对象"
Logic->>Writer : "创建SSE写入器"
Logic->>Reg : "注册完成信号(带TTL)"
Logic->>Emitter : "订阅channel事件"
Logic->>Writer : "发送connected事件"
par "模拟worker"
Logic->>Emitter : "推送通知/进度/完成事件"
Emitter-->>Logic : "事件通道消息"
and "等待完成信号"
Reg-->>Logic : "完成信号触发"
Logic->>Emitter : "取消订阅"
end
loop "主循环"
Emitter-->>Logic : "事件消息"
Logic->>Writer : "WriteEvent/WriteData"
Writer-->>Client : "SSE事件/数据"
alt "心跳周期"
Logic->>Writer : "WriteKeepAlive"
Writer-->>Client : " : keepalive"
end
end
```

**更新** AI 网关服务的架构更加复杂，包含了完整的聊天界面：
```mermaid
sequenceDiagram
participant Browser as "浏览器"
participant ChatUI as "聊天界面"
participant API as "AI API"
participant Models as "模型列表"
participant Completions as "对话补全"
Browser->>ChatUI : "访问 /aigtw/chat"
ChatUI->>API : "GET /aigtw/v1/models"
API->>Models : "获取模型列表"
Models-->>API : "返回模型数据"
API-->>ChatUI : "渲染模型选择器"
Browser->>ChatUI : "输入消息"
ChatUI->>API : "POST /aigtw/v1/chat/completions"
API->>Completions : "处理对话请求"
Completions-->>API : "流式返回结果"
API-->>ChatUI : "实时渲染聊天内容"
```

**更新** 基于antsx库的架构模式：
```mermaid
graph TB
A["客户端请求"] --> B["HTTP处理器"]
B --> C["业务逻辑层"]
C --> D["antsx Promise"]
C --> E["antsx Invoke"]
C --> F["antsx Reactor"]
C --> G["antsx PendingRegistry"]
C --> H["antsx EventEmitter"]
D --> I["异步结果处理"]
E --> J["并行任务执行"]
F --> K["goroutine池化"]
G --> L["请求-响应匹配"]
H --> M["事件广播"]
I --> N["SSE写入器"]
J --> N
K --> N
L --> N
M --> N
N --> O["客户端推送"]
```

**图表来源**
- [aiapp/ssegtw/internal/handler/routes.go:18-36](file://aiapp/ssegtw/internal/handler/routes.go#L18-L36)
- [aiapp/aigtw/internal/handler/routes.go:30-42](file://aiapp/aigtw/internal/handler/routes.go#L30-L42)
- [aiapp/ssegtw/internal/logic/sse/ssestreamlogic.go:53-116](file://aiapp/ssegtw/internal/logic/sse/ssestreamlogic.go#L53-L116)
- [aiapp/ssegtw/internal/logic/sse/chatstreamlogic.go:58-120](file://aiapp/ssegtw/internal/logic/sse/chatstreamlogic.go#L58-L120)
- [common/ssex/writer.go:23-54](file://common/ssex/writer.go#L23-L54)
- [common/antsx/README.md:10-438](file://common/antsx/README.md#L10-L438)

## 详细组件分析

### 入口与配置
- 配置文件包含服务名称、监听地址、端口、日志路径以及 ZeroRPC 客户端配置（端点、非阻塞、超时）。
- 应用入口加载配置，打印 Go 版本，创建 REST 服务器并启用自定义 CORS（允许任意来源、凭证、常用方法与头部），注册路由并启动服务组。

**更新** AI 网关服务的入口程序还包含了静态文件服务：
- 添加了 `/aigtw/chat` 路由，用于提供聊天界面 HTML 文件
- 支持多种路径查找策略，优先查找可执行文件所在目录的 chat.html
- 如果找不到本地文件，会尝试在项目根目录和 aiapp/aigtw 目录下查找

**章节来源**
- [aiapp/ssegtw/etc/ssegtw.yaml:1-14](file://aiapp/ssegtw/etc/ssegtw.yaml#L1-L14)
- [aiapp/aigtw/etc/aigtw.yaml:1-15](file://aiapp/aigtw/etc/aigtw.yaml#L1-L15)
- [aiapp/ssegtw/ssegtw.go:26-59](file://aiapp/ssegtw/ssegtw.go#L26-L59)
- [aiapp/aigtw/aigtw.go:29-75](file://aiapp/aigtw/aigtw.go#L29-L75)
- [aiapp/ssegtw/internal/config/config.go:11-14](file://aiapp/ssegtw/internal/config/config.go#L11-L14)
- [aiapp/aigtw/internal/config/config.go:20-25](file://aiapp/aigtw/internal/config/config.go#L20-L25)

### 路由与跨域
- SSE 路由组前缀为 /ssegtw/v1/sse，启用 rest.WithSSE() 以支持长连接。
- 健康检查路由 /ssegtw/v1/ping 返回简单文本。
- 跨域头通过自定义函数动态设置，允许来源、凭证、常见方法与头部，并暴露长度与类型等响应头。

**更新** AI 网关服务的路由配置：
- 模型列表路由：GET /aigtw/v1/models
- 对话补全路由：POST /aigtw/v1/chat/completions（启用 SSE 和无限超时）
- 健康检查路由：GET /aigtw/v1/ping
- 聊天界面路由：GET /aigtw/chat（静态文件服务）

**章节来源**
- [aiapp/ssegtw/ssegtw.api:13-38](file://aiapp/ssegtw/ssegtw.api#L13-L38)
- [aiapp/aigtw/aigtw.api:14-47](file://aiapp/aigtw/aigtw.api#L14-L47)
- [aiapp/ssegtw/internal/handler/routes.go:17-50](file://aiapp/ssegtw/internal/handler/routes.go#L17-L50)
- [aiapp/aigtw/internal/handler/routes.go:17-55](file://aiapp/aigtw/internal/handler/routes.go#L17-L55)
- [aiapp/ssegtw/ssegtw.go:35-46](file://aiapp/ssegtw/ssegtw.go#L35-L46)

### SSE事件流处理逻辑
- 请求参数：SSEStreamRequest，包含可选 channel。
- 处理流程：
  - 创建 SSE 写入器，若不支持流式则直接报错。
  - 若未提供 channel，则生成唯一 ID。
  - 注册完成信号（带默认 TTL），订阅事件通道。
  - 发送 connected 事件，携带 channel。
  - 启动模拟 worker：按固定间隔推送多条通知事件，最后发送 done 事件并 Resolve 完成信号。
  - 使用独立 goroutine 等待完成信号，触发取消订阅。
  - 主循环：转发事件到客户端；若事件为空则写入纯数据；周期性发送心跳注释。

```mermaid
flowchart TD
Start(["进入 SSE 事件流"]) --> Parse["解析请求参数"]
Parse --> NewWriter["创建SSE写入器"]
NewWriter --> Channel{"是否提供channel?"}
Channel --> |否| Gen["生成唯一channel"]
Channel --> |是| Use["使用传入channel"]
Gen --> Subscribe["注册完成信号+订阅事件通道"]
Use --> Subscribe
Subscribe --> SendConnected["发送connected事件"]
SendConnected --> Worker["启动模拟worker推送通知事件"]
Worker --> Done["推送done事件并Resolve完成信号"]
Done --> Wait["等待完成信号触发取消订阅"]
Wait --> Loop{"主循环"}
Loop --> |收到事件| Write["WriteEvent/WriteData"]
Loop --> |心跳周期| KeepAlive["WriteKeepAlive"]
Loop --> |客户端断开| Exit(["退出"])
Write --> Loop
KeepAlive --> Loop
```

**图表来源**
- [aiapp/ssegtw/internal/logic/sse/ssestreamlogic.go:39-116](file://aiapp/ssegtw/internal/logic/sse/ssestreamlogic.go#L39-L116)
- [common/ssex/writer.go:23-54](file://common/ssex/writer.go#L23-L54)

**章节来源**
- [aiapp/ssegtw/internal/handler/sse/ssestreamhandler.go:18-32](file://aiapp/ssegtw/internal/handler/sse/ssestreamhandler.go#L18-L32)
- [aiapp/ssegtw/internal/logic/sse/ssestreamlogic.go:39-116](file://aiapp/ssegtw/internal/logic/sse/ssestreamlogic.go#L39-L116)
- [aiapp/ssegtw/internal/types/types.go:15-17](file://aiapp/ssegtw/internal/types/types.go#L15-L17)

### AI对话流处理逻辑
- 请求参数：ChatStreamRequest，包含可选 channel 与可选 prompt，默认值为"Hello World"。
- 处理流程：
  - 创建 SSE 写入器，确定 channel，解析 prompt。
  - 注册完成信号，订阅事件通道，发送 connected 事件。
  - 启动模拟 worker：按固定间隔逐字符推送 token，最后发送 done 并 Resolve 完成信号。
  - 等待完成信号触发取消订阅。
  - 主循环：转发事件到客户端；周期性发送心跳注释。

```mermaid
flowchart TD
Start(["进入 AI 对话流"]) --> Parse["解析请求参数"]
Parse --> NewWriter["创建SSE写入器"]
NewWriter --> Channel{"是否提供channel?"}
Channel --> |否| Gen["生成唯一channel"]
Channel --> |是| Use["使用传入channel"]
Gen --> Subscribe["注册完成信号+订阅事件通道"]
Use --> Subscribe
Subscribe --> SendConnected["发送connected事件"]
SendConnected --> Worker["启动模拟worker逐字符推送token"]
Worker --> Done["推送done事件并Resolve完成信号"]
Done --> Wait["等待完成信号触发取消订阅"]
Wait --> Loop{"主循环"}
Loop --> |收到事件| Write["WriteEvent/WriteData"]
Loop --> |心跳周期| KeepAlive["WriteKeepAlive"]
Loop --> |客户端断开| Exit(["退出"])
Write --> Loop
KeepAlive --> Loop
```

**图表来源**
- [aiapp/ssegtw/internal/logic/sse/chatstreamlogic.go:39-120](file://aiapp/ssegtw/internal/logic/sse/chatstreamlogic.go#L39-L120)
- [common/ssex/writer.go:23-54](file://common/ssex/writer.go#L23-L54)

**章节来源**
- [aiapp/ssegtw/internal/handler/sse/chatstreamhandler.go:18-32](file://aiapp/ssegtw/internal/handler/sse/chatstreamhandler.go#L18-L32)
- [aiapp/ssegtw/internal/logic/sse/chatstreamlogic.go:39-120](file://aiapp/ssegtw/internal/logic/sse/chatstreamlogic.go#L39-L120)
- [aiapp/ssegtw/internal/types/types.go:6-9](file://aiapp/ssegtw/internal/types/types.go#L6-L9)

### SSE写入器与传输协议
- 写入器封装：
  - WriteEvent：写入事件名与数据，随后 flush。
  - WriteData：写入纯数据，随后 flush。
  - WriteComment：写入注释行（客户端忽略），随后 flush。
  - WriteKeepAlive：写入心跳注释。
- 协议要点：
  - 事件名与数据均以换行分隔，事件块以空行结束。
  - 心跳通过注释行实现，客户端可忽略但保持连接活跃。

**章节来源**
- [common/ssex/writer.go:14-54](file://common/ssex/writer.go#L14-L54)

### 服务上下文与依赖
- 服务上下文包含：
  - 配置对象
  - ZeroRPC 客户端（带元数据拦截器）
  - 事件发射器（泛型事件模型）
  - 等待完成注册（泛型键，带默认 TTL）
- 依赖注入：
  - 处理器将上下文注入逻辑层，逻辑层通过事件发射器与等待完成注册协调流式推送与结束条件。

**章节来源**
- [aiapp/ssegtw/internal/svc/servicecontext.go:23-38](file://aiapp/ssegtw/internal/svc/servicecontext.go#L23-L38)

### 类型定义
- ChatStreamRequest：包含可选 channel 与可选 prompt。
- PingReply：健康检查返回结构。
- SSEStreamRequest：包含可选 channel。

**更新** AI 网关服务的类型定义更加丰富：
- ChatCompletionRequest：包含模型ID、消息列表、流式输出开关、采样参数等
- ChatCompletionResponse：包含补全ID、模型ID、选择结果、Token用量统计
- ChatDelta：包含角色、内容增量、推理内容增量
- ChatMessage：包含角色、内容、推理内容
- ListModelsResponse：包含模型列表
- ModelObject：包含模型ID、显示名称、支持流式等元数据

**章节来源**
- [aiapp/ssegtw/internal/types/types.go:6-17](file://aiapp/ssegtw/internal/types/types.go#L6-L17)
- [aiapp/aigtw/internal/types/types.go:14-91](file://aiapp/aigtw/internal/types/types.go#L14-L91)

### 聊天界面与客户端集成
**更新** 新增了完整的聊天界面功能：

- **聊天界面特性**：
  - 美观的深色/浅色主题切换
  - 多语言支持（中英文）
  - 实时流式聊天，支持深度思考过程展示
  - 代码高亮和复制功能
  - 响应式设计，支持移动端访问
  - 详细的统计信息（分块数量、耗时等）

- **客户端集成示例**：
  - 访问 `/aigtw/chat` 获取完整的聊天界面
  - 通过 `/aigtw/v1/models` 获取可用模型列表
  - 通过 `/aigtw/v1/chat/completions` 发送聊天请求
  - 支持流式和非流式两种模式

- **JavaScript 功能**：
  - 实时流式处理，支持深度思考标签解析
  - Markdown 渲染，支持代码块高亮
  - 键盘快捷键支持（Enter 发送）
  - 停止流式输出功能
  - 本地存储主题和语言偏好

**章节来源**
- [aiapp/aigtw/aigtw.go:46-65](file://aiapp/aigtw/aigtw.go#L46-L65)
- [aiapp/aigtw/internal/handler/routes.go:30-42](file://aiapp/aigtw/internal/handler/routes.go#L30-L42)
- [aiapp/aigtw/chat.html:1089-1836](file://aiapp/aigtw/chat.html#L1089-1836)

## antsx库架构模式详解

### 场景选择指南
antsx库提供了完整的场景选择指南，帮助开发者根据具体需求选择合适的架构模式：

| 场景 | 推荐工具 | 关键特征 |
|------|---------|---------|
| 阻塞调用加超时 | `Promise` | 数据直连流转，只需包装阻塞 I/O |
| 并行聚合多个调用 | `Invoke` | 多个独立调用并行执行，汇总结果 |
| 高并发任务+资源控制 | `Reactor` | goroutine 池化复用，ID 去重 |
| 异步请求-响应匹配 | `PendingRegistry` | 请求和响应走不同通道，用关联 ID 匹配 |
| 事件广播/多订阅者 | `EventEmitter` | 一对多推送，topic 级隔离 |
| MQ 异步流式推送 | `EventEmitter` + `PendingRegistry` | 解耦的生产者-消费者 + 完成信号 |

**更新** 如何判断用哪个？

```
你的数据流是怎样的？

├── 直连同步流（调用方直接拿结果）
│   ├── 单个阻塞调用需要超时 ──────────> Promise
│   ├── 多个独立调用并行聚合 ──────────> Invoke
│   └── 高并发提交需要限流 ───────────> Reactor + Promise
│
└── 解耦异步流（请求和响应走不同通道）
    ├── 一次请求 - 一次响应 ──────────> PendingRegistry
    ├── 一次请求 - 多次推送 ──────────> EventEmitter
    └── 一次请求 - 多次推送 + 完成信号 ─> EventEmitter + PendingRegistry
```

### Promise模式
Promise是antsx库中最基础的异步结果容器，适用于将阻塞调用包装为可超时、可取消的异步操作。

**典型场景**：gRPC stream 读取加超时
```
上游 AI ──SSE──> reader.Recv() [阻塞] ──> gRPC stream.Send() ──> 客户端
                     │
                     └── 如果上游挂了，Recv() 永远阻塞
                         用 Promise 包装后可以设超时中断
```

**基本用法**：
```go
p := antsx.NewPromise[string]("req-1")

go func() {
    result := doSomeWork()
    p.Resolve(result)
}()

val, err := p.Await(ctx)                    // 跟随 context 超时
val, err := p.AwaitWithTimeout(3 * time.Second) // 固定超时
```

**链式调用**：
```go
// Then: T -> (U, error)
p2 := antsx.Then(ctx, p1, func(val int) (string, error) {
    return fmt.Sprintf("result: %d", val), nil
})

// Map: T -> U（纯映射）
p3 := antsx.Map(ctx, p1, func(val int) string {
    return strconv.Itoa(val)
})

// FlatMap: T -> *Promise[U]（链接多个异步操作）
p4 := antsx.FlatMap(ctx, p1, func(val int) *antsx.Promise[string] {
    return fetchFromRemote(val)
})
```

**并发组合**：
```go
// 等待所有完成，任一失败立即取消其余
results, err := antsx.PromiseAll(ctx, p1, p2, p3)

// 竞争，返回最先完成的
val, err := antsx.PromiseRace(ctx, p1, p2)
```

**错误捕获**：
```go
p.Catch(func(err error) {
    log.Printf("任务失败: %v", err)
})
```

### Invoke模式
Invoke适用于多个独立的 RPC/IO 调用需要并行执行并汇总结果的场景。

**典型场景**：聚合用户画像
```
        ┌── getBasic()  ──> 基础信息
请求 ───┼── getStats()  ──> 统计数据  ───> 合并为完整画像
        └── getPrefs()  ──> 偏好设置
```

**Invoke用法**：
```go
results, err := antsx.Invoke(ctx,
    antsx.Task[*UserInfo]{
        Name:    "basic-info",
        Timeout: 3 * time.Second,
        Fn:      func(ctx context.Context) (*UserInfo, error) { return userRpc.GetBasic(ctx) },
    },
    antsx.Task[*UserInfo]{
        Name:    "stats",
        Timeout: 5 * time.Second,
        Fn:      func(ctx Context) (*UserInfo, error) { return userRpc.GetStats(ctx) },
    },
)
// results 按 index 排列（不是完成顺序）
// 任一失败 -> 立即取消其余，返回第一个错误
```

**InvokeCallback**：
```go
profile, err := antsx.InvokeCallback(ctx,
    []antsx.Task[*UserInfo]{
        {Name: "basic", Fn: getBasic, Timeout: 3 * time.Second},
        {Name: "stats", Fn: getStats, Timeout: 5 * time.Second},
    },
    func(infos []*UserInfo) (*AggregatedProfile, error) {
        return mergeProfiles(infos[0], infos[1]), nil
    },
)
```

**InvokeWithReactor**：
```go
reactor, _ := antsx.NewReactor(10)
defer reactor.Release()

results, err := antsx.InvokeWithReactor(ctx, reactor,
    antsx.Task[int]{Name: "t1", Fn: task1},
    antsx.Task[int]{Name: "t2", Fn: task2},
)
```

### Reactor模式
Reactor适用于高并发场景下需要控制 goroutine 数量，同时需要 ID 去重防止重复提交的场景。

**典型场景**：消息处理限流
```
高并发请求 ──> Reactor (pool size=100) ──> 有序处理
                   │
                   └── ID 去重：同一请求不会重复执行
```

**Reactor用法**：
```go
reactor, err := antsx.NewReactor(100)
defer reactor.Release()

// Submit: 带 ID 去重，返回 Promise
promise, err := antsx.Submit(ctx, reactor, "order-123", func(ctx context.Context) (string, error) {
    return processOrder(ctx, "order-123")
})
// err == ErrDuplicateID 表示同 ID 任务正在执行
val, err := promise.Await(ctx)

// Post: fire-and-forget，错误仅日志记录
antsx.Post(ctx, reactor, func(ctx context.Context) (any, error) {
    return nil, sendNotification(ctx)
})

// Go: 最轻量的提交，无 Promise/ID 开销
reactor.Go(func() {
    cleanupTempFiles()
})
```

### PendingRegistry模式
PendingRegistry适用于请求和响应走不同通道（MQ、TCP 等），需要通过关联 ID 将响应路由回对应的请求方的场景。

**典型流程**：
```
发送端                              消费端（另一个 goroutine）
  │                                    │
  ├── reg.Register(id) -> Promise      │
  ├── mq.Publish(请求)                 │
  ├── promise.Await(ctx) [等待]        │
  │                                    ├── 收到响应
  │                                    ├── reg.Resolve(id, 响应)
  │<── Promise 返回结果 ───────────────┘
```

**场景一：MQ 消息匹配**：
```go
reg := antsx.NewPendingRegistry[ResponseMsg](
    antsx.WithDefaultTTL(10 * time.Second),
)
defer reg.Close()

// --- 发送端 ---
correlationId := uuid.New().String()
promise, err := reg.Register(correlationId, 5*time.Second)
if err != nil {
    return err // ErrDuplicateID 或 ErrRegistryClosed
}

err = kafka.Publish("cmd-topic", Message{
    CorrelationId: correlationId,
    Payload:       cmdPayload,
})
if err != nil {
    reg.Reject(correlationId, err) // 发送失败，立即清理
    return err
}

reply, err := promise.Await(ctx)
// err: nil(成功) / ErrPendingExpired(TTL超时) / context.DeadlineExceeded(ctx超时)
```

**场景二：TCP sendNo 匹配**：
```go
// 发送端
sendNo := atomic.AddUint32(&seq, 1)
id := strconv.FormatUint(uint64(sendNo), 10)

promise, _ := reg.Register(id, 3*time.Second)
conn.WriteFrame(Frame{SendNo: sendNo, Data: payload})
reply, err := promise.Await(ctx)

// 接收端（read loop）
for {
    frame := conn.ReadFrame()
    id := strconv.FormatUint(uint64(frame.SendNo), 10)
    reg.Resolve(id, frame)
}
```

**RequestReply 便捷封装**：
```go
// 一行代码完成 注册 -> 发送 -> 等待：
reply, err := antsx.RequestReply(ctx, reg, correlationId, func() error {
    return kafka.Publish("cmd-topic", Message{
        CorrelationId: correlationId,
        Payload:       cmdPayload,
    })
})
// sendFn 失败会自动清理已注册的 entry
```

### EventEmitter模式
EventEmitter适用于一对多事件广播，按 topic 隔离的场景，非常适合SSE推送、实时通知等场景。

**典型流程**：
```
生产者                         消费者 A（SSE 连接 1）
  │                              │
  ├── emitter.Emit(topic, event) ├── ch, cancel := emitter.Subscribe(topic)
  │         │                    ├── for event := range ch { sendSSE(event) }
  │         ├──────────────────> │
  │         │                    消费者 B（SSE 连接 2）
  │         │                    │
  │         └──────────────────> ├── for event := range ch { sendSSE(event) }
```

**EventEmitter用法**：
```go
emitter := antsx.NewEventEmitter[SSEEvent]()
defer emitter.Close()

// 订阅（bufSize 防止慢消费者阻塞生产者）
ch, cancel := emitter.Subscribe("user-123", 32)
defer cancel()

go func() {
    for event := range ch {
        sendSSE(event)
    }
}()

// 发布（非阻塞，慢消费者消息丢弃）
emitter.Emit("user-123", SSEEvent{Event: "message", Data: "hello"})

// 查询
emitter.TopicCount()                  // 活跃 topic 数
emitter.SubscriberCount("user-123")   // 指定 topic 订阅者数
```

### 组合模式：EventEmitter + PendingRegistry
这种组合模式适用于通过 MQ 解耦的异步流式推送场景。请求通过 MQ 发给后端 Worker，Worker 逐步产生事件推回前端，最后用一个完成信号收尾。

**与 Promise 直连模式的区别**：
```
Promise 直连模式（如 gRPC stream + 上游 SSE）：
  客户端 ──请求──> 网关 ──gRPC stream──> 服务 ──HTTP SSE──> 上游 AI
                   │<── chunk ──────────<── chunk ──────<── chunk
                   │
                   └── 数据在同一条链路直接流转，Promise 只是给阻塞的 Recv() 加超时

EventEmitter + PendingRegistry 模式（如 MQ 异步流式）：
  客户端 ──请求──> HTTP Handler ──MQ──> AI Worker（独立进程）
                   │                        │
                   │<── EventEmitter ────────┤ 逐 token 推送
                   │                        │
                   │<── PendingRegistry ────┘ 完成信号
                   │
                   └── 请求和响应走不同通道，需要 EventEmitter 桥接事件 + PendingRegistry 匹配完成
```

**完整示例**：
```go
// ========== 服务初始化 ==========
emitter := antsx.NewEventEmitter[SSEEvent]()
pendingReg := antsx.NewPendingRegistry[string](antsx.WithDefaultTTL(60 * time.Second))

// ========== HTTP Handler: 客户端发起对话 ==========
func handleChat(ctx context.Context, req ChatRequest) {
    sessionId := uuid.New().String()

    // 1. 注册完成信号（60s TTL 兜底，防止 Worker 挂掉后连接永不关闭）
    done, _ := pendingReg.Register(sessionId, 60*time.Second)

    // 2. 订阅该 session 的流式事件
    ch, cancel := emitter.Subscribe(sessionId)
    defer cancel()

    // 3. 通过 MQ 发送给后端 AI Worker
    mq.Publish("ai-request", AIRequest{SessionId: sessionId, Prompt: req.Prompt})

    // 4. 等待完成信号，触发 cancel 关闭 ch
    go func() {
        done.Await(ctx)
        cancel()
    }()

    // 5. 持续推送事件给客户端，直到 ch 关闭
    for event := range ch {
        sendSSE(event)
    }
}

// ========== AI Worker: 处理推理结果（独立进程/goroutine） ==========
func handleAIResult(result AIResult) {
    for _, token := range result.Tokens {
        emitter.Emit(result.SessionId, SSEEvent{Event: "token", Data: token})
    }
    emitter.Emit(result.SessionId, SSEEvent{Event: "done", Data: ""})
    pendingReg.Resolve(result.SessionId, "completed") // 触发完成信号
}
```

**章节来源**
- [common/antsx/README.md:10-438](file://common/antsx/README.md#L10-L438)
- [common/antsx/promise.go:1-147](file://common/antsx/promise.go#L1-L147)
- [common/antsx/invoke.go:1-150](file://common/antsx/invoke.go#L1-L150)
- [common/antsx/reactor.go:1-93](file://common/antsx/reactor.go#L1-L93)
- [common/antsx/pending.go:1-184](file://common/antsx/pending.go#L1-L184)
- [common/antsx/emitter.go:1-118](file://common/antsx/emitter.go#L1-L118)
- [common/antsx/errors.go:1-10](file://common/antsx/errors.go#L1-L10)
- [common/antsx/promise_ext.go:1-132](file://common/antsx/promise_ext.go#L1-L132)

## 依赖分析
- 组件耦合
  - 处理器仅负责参数解析与逻辑委派，低耦合。
  - 逻辑层依赖服务上下文，通过事件发射器与等待完成注册解耦外部系统。
  - SSE 写入器对 http.ResponseWriter 强依赖（需要 Flusher），通过接口约束保证兼容性。
  - **更新** antsx库提供并发安全的依赖注入，支持多种架构模式的组合使用。
- 外部依赖
  - ZeroRPC 客户端用于后端服务通信（当前示例为本地演示，未实际调用）。
  - 事件发射器与等待完成注册来自通用库，提供线程安全与生命周期管理。
  - **更新** antsx库依赖 ants goroutine 池，提供高性能的并发控制。

```mermaid
classDiagram
class ServiceContext {
+Config
+ZeroRpcCli
+Emitter
+PendingReg
}
class SseStreamLogic {
+Logger
+ctx
+svcCtx
+w
+r
+SseStream(req)
}
class ChatStreamLogic {
+Logger
+ctx
+svcCtx
+w
+r
+ChatStream(req)
}
class Writer {
+WriteEvent(event,data)
+WriteData(data)
+WriteComment(comment)
+WriteKeepAlive()
}
class Promise {
+Await(ctx)
+Resolve(val)
+Reject(err)
+Catch(fn)
}
class Reactor {
+Submit(ctx,id,task)
+Post(ctx,task)
+Go(fn)
+Release()
}
class PendingRegistry {
+Register(id,ttl)
+Resolve(id,val)
+Reject(id,err)
}
class EventEmitter {
+Subscribe(topic,bufSize)
+Emit(topic,value)
+Close()
}
ServiceContext --> Writer : "被逻辑层使用"
ServiceContext --> Promise : "被逻辑层使用"
ServiceContext --> Reactor : "被逻辑层使用"
ServiceContext --> PendingRegistry : "被逻辑层使用"
ServiceContext --> EventEmitter : "被逻辑层使用"
SseStreamLogic --> ServiceContext : "依赖"
ChatStreamLogic --> ServiceContext : "依赖"
```

**图表来源**
- [aiapp/ssegtw/internal/svc/servicecontext.go:23-38](file://aiapp/ssegtw/internal/svc/servicecontext.go#L23-L38)
- [aiapp/ssegtw/internal/logic/sse/ssestreamlogic.go:20-37](file://aiapp/ssegtw/internal/logic/sse/ssestreamlogic.go#L20-L37)
- [aiapp/ssegtw/internal/logic/sse/chatstreamlogic.go:20-37](file://aiapp/ssegtw/internal/logic/sse/chatstreamlogic.go#L20-L37)
- [common/ssex/writer.go:14-54](file://common/ssex/writer.go#L14-L54)
- [common/antsx/promise.go:16-29](file://common/antsx/promise.go#L16-L29)
- [common/antsx/reactor.go:14-17](file://common/antsx/reactor.go#L14-L17)
- [common/antsx/pending.go:29-36](file://common/antsx/pending.go#L29-L36)
- [common/antsx/emitter.go:13-18](file://common/antsx/emitter.go#L13-L18)

## 性能考虑
- 心跳保活：每 30 秒发送一次心跳注释，避免代理或中间件误判连接空闲。
- 自动刷新：SSE 写入器每次写入后立即 flush，确保事件即时到达。
- 事件通道：事件发射器内部使用通道，避免阻塞写入；订阅取消后及时释放资源。
- 超时与完成信号：等待完成注册带有默认 TTL，防止长时间悬挂；完成后主动取消订阅，降低内存占用。
- 并发模型：主循环与模拟 worker 分离，避免阻塞事件转发。
- **更新** antsx库性能优化：
  - goroutine池化：Reactor通过 ants 池限制并发数量，避免资源耗尽
  - ID去重：防止重复任务提交，提高系统稳定性
  - 非阻塞广播：EventEmitter使用非阻塞发送，慢消费者消息自动丢弃
  - TTL自动过期：PendingRegistry自动清理超时请求，防止内存泄漏
  - 并发组合：PromiseAll和PromiseRace提供高效的并发控制
- **更新** AI 网关服务的性能优化：
  - 流式渲染优化：使用 requestAnimationFrame 进行节流渲染
  - 深度思考过程：支持实时显示推理过程，提升用户体验
  - 代码块高亮：使用高效的 Markdown 渲染引擎
  - 本地存储：减少重复请求，提升响应速度

## 故障排查指南
- 连接无法建立
  - 检查 CORS 设置是否正确，确认浏览器来源、凭证与方法匹配。
  - 确认服务监听地址与端口配置正确。
- SSE 不显示事件
  - 确认路由已启用 rest.WithSSE()。
  - 检查 SSE 写入器是否成功创建（ResponseWriter 是否支持 Flusher）。
- 事件丢失或延迟
  - 检查事件通道订阅是否被提前取消（完成信号触发后会取消订阅）。
  - 确认主循环未被阻塞，心跳周期是否合理。
- **更新** antsx库相关问题排查：
  - Promise超时：检查AwaitWithTimeout设置是否合理，确认任务执行时间
  - Reactor池满：检查并发任务数量，适当调整池大小
  - PendingRegistry过期：确认TTL设置，检查响应是否及时处理
  - EventEmitter阻塞：检查订阅者缓冲区大小，避免慢消费者影响生产者
- **更新** 聊天界面问题排查：
  - 确认 `/aigtw/chat` 路由能够正确找到 chat.html 文件
  - 检查 `/aigtw/v1/models` 接口是否正常返回模型列表
  - 验证 `/aigtw/v1/chat/completions` 接口的流式输出是否正常
  - 查看浏览器开发者工具的网络面板和控制台错误

**章节来源**
- [aiapp/ssegtw/sse_demo.html:558-635](file://aiapp/ssegtw/sse_demo.html#L558-L635)
- [aiapp/aigtw/chat.html:1089-1836](file://aiapp/aigtw/chat.html#L1089-1836)
- [aiapp/ssegtw/internal/handler/routes.go:34-36](file://aiapp/ssegtw/internal/handler/routes.go#L34-L36)
- [common/ssex/writer.go:14-21](file://common/ssex/writer.go#L14-L21)
- [common/antsx/README.md:10-438](file://common/antsx/README.md#L10-L438)

## 结论
SSE 网关服务通过清晰的分层设计与事件驱动机制，实现了稳定的长连接流式传输。其核心优势在于：
- 明确的 SSE 协议封装与心跳保活
- 事件发射器与等待完成注册的解耦设计
- 易于扩展的处理器与逻辑层分离
- 完备的跨域与健康检查支持
- 丰富的本地调试工具

**更新** 基于antsx库的架构模式进一步增强了系统的并发处理能力和稳定性：
- Promise模式提供可靠的异步结果管理
- Invoke模式支持高效的并行任务编排
- Reactor模式实现goroutine池化和ID去重
- PendingRegistry模式确保请求-响应的可靠匹配
- EventEmitter模式支持高效的事件广播
- 组合模式提供完整的异步流式处理解决方案

**更新** AI 网关服务进一步增强了用户体验：
- 提供完整的聊天界面，支持多语言和主题切换
- 实现实时流式聊天，支持深度思考过程展示
- 包含代码高亮和复制功能
- 响应式设计，支持移动端访问

建议在生产环境中进一步完善：
- 增加连接数与内存限制配置
- 加强错误分类与重试策略
- 引入指标采集与告警
- 对外暴露更细粒度的健康检查与指标端点
- **更新** 优化聊天界面的性能和用户体验
- **更新** 充分利用antsx库的并发模式提升系统性能

## 附录

### API 接口文档
- **SSE 网关服务**
  - 健康检查
    - 方法：GET
    - 路径：/ssegtw/v1/ping
    - 返回：PingReply
  - SSE 事件流
    - 方法：POST
    - 路径：/ssegtw/v1/sse/stream
    - 请求体：SSEStreamRequest
    - 返回：PingReply
  - AI 对话流
    - 方法：POST
    - 路径：/ssegtw/v1/sse/chat/stream
    - 请求体：ChatStreamRequest
    - 返回：PingReply

- **AI 网关服务**
  - 健康检查
    - 方法：GET
    - 路径：/aigtw/v1/ping
    - 返回：PingReply
  - 模型列表
    - 方法：GET
    - 路径：/aigtw/v1/models
    - 返回：ListModelsResponse
  - 对话补全
    - 方法：POST
    - 路径：/aigtw/v1/chat/completions
    - 请求体：ChatCompletionRequest
    - 返回：ChatCompletionResponse（流式输出）

**更新** 新增了聊天界面的静态文件服务：
- 聊天界面
  - 方法：GET
  - 路径：/aigtw/chat
  - 返回：chat.html

**章节来源**
- [aiapp/ssegtw/ssegtw.api:18-38](file://aiapp/ssegtw/ssegtw.api#L18-L38)
- [aiapp/aigtw/aigtw.api:19-47](file://aiapp/aigtw/aigtw.api#L19-L47)
- [aiapp/ssegtw/internal/types/types.go:6-17](file://aiapp/ssegtw/internal/types/types.go#L6-L17)
- [aiapp/aigtw/internal/types/types.go:14-91](file://aiapp/aigtw/internal/types/types.go#L14-L91)

### 客户端集成示例
- **SSE 网关服务**
  - 使用标准 Fetch API 与 ReadableStream 读取 SSE：
    - 选择端点：/sse/stream 或 /chat/stream
    - 可选参数：channel、prompt（对话流）
    - 读取响应体的流，按行解析 event/data/注释行，渲染事件与心跳

- **AI 网关服务**
  - 访问聊天界面：`/aigtw/chat`
  - 获取模型列表：`/aigtw/v1/models`
  - 发送聊天请求：`/aigtw/v1/chat/completions`
  - 支持流式和非流式两种模式
  - 参数包括：model、messages、stream、temperature、top_p、max_tokens 等

**更新** 聊天界面的客户端集成：
- 直接访问 `/aigtw/chat` 获取完整的聊天界面
- 支持深色/浅色主题切换
- 多语言支持（中英文）
- 实时流式聊天，支持深度思考过程展示

**更新** 基于antsx库的客户端集成示例：
- 使用Promise模式处理异步请求
- 使用Invoke模式并行执行多个任务
- 使用Reactor模式控制并发数量
- 使用PendingRegistry模式管理请求-响应匹配
- 使用EventEmitter模式处理事件广播

**章节来源**
- [aiapp/ssegtw/sse_demo.html:558-635](file://aiapp/ssegtw/sse_demo.html#L558-L635)
- [aiapp/aigtw/chat.html:1089-1836](file://aiapp/aigtw/chat.html#L1089-1836)

### 部署与监控
- **SSE 网关服务**
  - 监听地址与端口：在配置文件中设置 Host 与 Port
  - 日志路径：在配置文件中设置 Log.Path
  - ZeroRPC 客户端：配置 Endpoints、NonBlock、Timeout

- **AI 网关服务**
  - 监听地址与端口：在配置文件中设置 Host 与 Port
  - 日志路径：在配置文件中设置 Log.Path
  - AI Chat RPC 客户端：配置 Endpoints、NonBlock、Timeout
  - 静态文件：确保 chat.html 文件可被正确访问

**更新** 部署注意事项：
- 确保 chat.html 文件位于正确的路径
- 配置静态文件服务以提供聊天界面
- 监控两个服务的运行状态和性能指标
- **更新** 配置antsx库的并发参数，根据实际负载调整池大小和超时设置

**章节来源**
- [aiapp/ssegtw/etc/ssegtw.yaml:1-14](file://aiapp/ssegtw/etc/ssegtw.yaml#L1-L14)
- [aiapp/aigtw/etc/aigtw.yaml:1-15](file://aiapp/aigtw/etc/aigtw.yaml#L1-L15)

### antsx库使用最佳实践
- **Promise模式最佳实践**：
  - 合理设置超时时间，避免长时间阻塞
  - 使用Catch处理错误，防止异常传播
  - 避免在Then链中进行阻塞操作
- **Invoke模式最佳实践**：
  - 为每个任务设置合理的超时时间
  - 使用InvokeCallback进行结果聚合
  - 对大量任务使用InvokeWithReactor进行池化控制
- **Reactor模式最佳实践**：
  - 根据CPU核心数设置合适的池大小
  - 使用Submit进行带ID的任务提交
  - 使用Post进行fire-and-forget任务
- **PendingRegistry模式最佳实践**：
  - 合理设置TTL，避免内存泄漏
  - 使用RequestReply简化请求-响应流程
  - 注意并发Resolve的线程安全性
- **EventEmitter模式最佳实践**：
  - 为慢消费者设置合适的缓冲区大小
  - 使用Topic进行事件隔离
  - 及时取消订阅，避免资源泄漏

**章节来源**
- [common/antsx/README.md:41-438](file://common/antsx/README.md#L41-L438)
- [common/antsx/antsx_test.go:1-459](file://common/antsx/antsx_test.go#L1-L459)
- [common/antsx/invoke_test.go:1-306](file://common/antsx/invoke_test.go#L1-L306)
- [common/antsx/pending_test.go:1-343](file://common/antsx/pending_test.go#L1-L343)