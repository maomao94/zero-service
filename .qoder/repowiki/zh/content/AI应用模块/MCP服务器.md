# MCP服务器

<cite>
**本文档引用的文件**
- [mcpserver.go](file://aiapp/mcpserver/mcpserver.go)
- [mcpserver.yaml](file://aiapp/mcpserver/etc/mcpserver.yaml)
- [config.go](file://aiapp/mcpserver/internal/config/config.go)
- [servicecontext.go](file://aiapp/mcpserver/internal/svc/servicecontext.go)
- [registry.go](file://aiapp/mcpserver/internal/tools/registry.go)
- [echo.go](file://aiapp/mcpserver/internal/tools/echo.go)
- [modbus.go](file://aiapp/mcpserver/internal/tools/modbus.go)
- [testprogress.go](file://aiapp/mcpserver/internal/tools/testprogress.go)
- [wrapper.go](file://common/mcpx/wrapper.go)
- [server.go](file://common/mcpx/server.go)
- [auth.go](file://common/mcpx/auth.go)
- [client.go](file://common/mcpx/client.go)
- [ctxprop.go](file://common/ctxprop/ctx.go)
- [ctxData.go](file://common/ctxdata/ctxData.go)
- [aichat.yaml](file://aiapp/aichat/etc/aichat.yaml)
- [servicecontext.go](file://aiapp/aichat/internal/svc/servicecontext.go)
- [go.mod](file://go.mod)
- [README.md](file://README.md)
</cite>

## 更新摘要
**所做更改**
- **新增SSE连接超时配置**：在MCP配置中新增SseTimeout字段，支持24小时的SSE连接超时设置
- **新增工具执行超时配置**：在MCP配置中新增MessageTimeout字段，支持5分钟的工具执行超时设置
- **增强超时配置管理**：通过registerRoutes方法分别配置SSE连接超时和工具执行超时
- **优化长时间连接支持**：通过SseTimeout配置支持更灵活的部署场景，增强长时间连接支持
- **改进工具调用性能**：通过MessageTimeout配置优化工具执行超时控制
- **扩展配置灵活性**：新增的超时配置为不同部署场景提供更精细的控制能力

## 目录
1. [简介](#简介)
2. [项目结构](#项目结构)
3. [核心组件](#核心组件)
4. [架构总览](#架构总览)
5. [详细组件分析](#详细组件分析)
6. [双层认证系统](#双层认证系统)
7. [Streamable HTTP传输协议](#streamable-http传输协议)
8. [SSE传输协议改进](#sse传输协议改进)
9. [超时配置管理](#超时配置管理)
10. [上下文传播机制](#上下文传播机制)
11. [工具实现详解](#工具实现详解)
12. [CallToolWrapper现代化工具处理](#calltoolwrapper现代化工具处理)
13. [日志增强功能](#日志增强功能)
14. [配置管理增强](#配置管理增强)
15. [多服务器连接管理](#多服务器连接管理)
16. [JWT密钥配置增强](#jwt密钥配置增强)
17. [工具名前缀路由](#工具名前缀路由)
18. [日志配置优化](#日志配置优化)
19. [依赖分析](#依赖分析)
20. [性能考量](#性能考量)
21. [故障排查指南](#故障排查指南)
22. [结论](#结论)
23. [附录](#附录)

## 简介
本文件为MCP（Model Context Protocol）服务器的技术文档，围绕在本仓库中的MCP服务器实现进行系统化说明。该实现基于go-zero框架和最新的MCP协议规范，提供模块化的MCP服务器示例，包含：

- **现代化工具处理机制**：采用CallToolWrapper统一包装所有工具，提供更好的日志记录、上下文提取和错误处理能力
- **双层认证系统**：支持JWT令牌和连接级服务令牌的双重验证机制
- **Streamable HTTP传输协议**：实现2025年3月26日规范的流式HTTP传输
- **SSE传输协议改进**：增强的Server-Sent Events支持，包含认证上下文提取机制
- **超时配置管理**：新增SSE连接超时和工具执行超时配置，支持更灵活的部署场景
- **上下文传播机制**：实现HTTP头部与上下文的双向映射和自动传播
- **模块化架构**：采用go-zero合规的internal目录结构，包含config、svc、tools等子模块
- **增强配置管理**：支持Auth配置段落、useStreamable标志和超时配置
- **服务上下文**：集中管理服务依赖和服务生命周期
- **工具注册系统**：统一的工具注册机制，支持多个工具的动态注册
- **增强工具实现**：包含echo回显工具、Modbus协议工具和testprogress进度通知工具，支持上下文传播
- **日志增强功能**：改进的调试日志输出，同时捕获token和username信息
- **响应消息增强**：在响应消息中支持中文用户名显示
- **与AI生态的深度集成**：支持与Claude Code、Copilot等AI代理的技能集成
- **现代化配置结构**：采用简化的配置层次，提供更好的可读性和维护性
- **多服务器连接管理**：支持多MCP服务器连接和工具名前缀路由
- **JWT密钥配置增强**：支持多JWT密钥配置，提升安全性和密钥轮换能力
- **认证上下文提取**：新增的SSE传输认证上下文提取功能，确保工具调用的一致性
- **进度通知功能**：新增的testprogress工具支持实时进度更新，用于演示长时间任务的进度反馈

本仓库中MCP服务器属于aiapp子模块，当前实现展示了如何通过go-zero的mcp包快速搭建模块化的MCP服务，并注册多个工具供外部AI代理调用。

**章节来源**
- [mcpserver.go:19-38](file://aiapp/mcpserver/mcpserver.go#L19-L38)
- [mcpserver.yaml:1-29](file://aiapp/mcpserver/etc/mcpserver.yaml#L1-L29)

## 项目结构
MCP服务器位于aiapp/mcpserver目录，采用完全模块化的go-zero合规布局：

```
aiapp/mcpserver/
├── mcpserver.go              # 服务器入口点
├── etc/
│   └── mcpserver.yaml        # 服务器配置文件（含Auth配置和超时配置）
└── internal/
    ├── config/
    │   └── config.go         # 配置结构定义
    ├── svc/
    │   └── servicecontext.go # 服务上下文管理
    └── tools/
        ├── echo.go           # echo工具实现（使用CallToolWrapper）
        ├── modbus.go         # Modbus工具实现（使用CallToolWrapper）
        ├── testprogress.go   # testprogress工具实现（使用CallToolWrapper）
        └── registry.go       # 工具注册中心
```

**章节来源**
- [README.md:59-108](file://README.md#L59-L108)
- [mcpserver.go:1-41](file://aiapp/mcpserver/mcpserver.go#L1-L41)
- [mcpserver.yaml:1-29](file://aiapp/mcpserver/etc/mcpserver.yaml#L1-L29)

## 核心组件

### 配置管理模块
- **Config结构**：继承mcpx.McpServerConf，扩展BridgeModbusRpcConf用于Modbus服务调用
- **配置加载**：通过conf.MustLoad加载YAML配置到Config结构
- **配置验证**：包含MCP基础配置、Auth配置、Modbus RPC客户端配置和超时配置
- **环境标识**：支持Mode开发环境标识，便于环境特定配置管理
- **超时配置**：新增SseTimeout（SSE连接超时）和MessageTimeout（工具执行超时）配置

### 服务上下文模块
- **ServiceContext结构**：包含Config和BridgeModbusCli客户端
- **依赖注入**：通过NewServiceContext集中管理服务依赖
- **客户端初始化**：基于BridgeModbusRpcConf创建gRPC客户端

### 工具注册中心
- **RegisterAll函数**：统一注册所有工具，包括新增的testprogress工具
- **工具注册机制**：支持动态工具注册和管理
- **工具隔离**：每个工具独立实现，便于维护和扩展

**章节来源**
- [config.go:8-12](file://aiapp/mcpserver/internal/config/config.go#L8-L12)
- [servicecontext.go:10-24](file://aiapp/mcpserver/internal/svc/servicecontext.go#L10-L24)
- [registry.go:9-15](file://aiapp/mcpserver/internal/tools/registry.go#L9-L15)

## 架构总览
MCP服务器在本仓库中的角色是作为AI代理的工具提供方，其模块化架构如下：

```mermaid
graph TB
subgraph "MCP服务器模块化架构"
CFG["配置管理<br/>internal/config/config.go"]
CTX["服务上下文<br/>internal/svc/servicecontext.go"]
REG["工具注册中心<br/>internal/tools/registry.go"]
ECHO["Echo工具<br/>internal/tools/echo.go"]
MODBUS["Modbus工具<br/>internal/tools/modbus.go"]
TESTPROGRESS["TestProgress工具<br/>internal/tools/testprogress.go"]
ENTRY["入口点<br/>mcpserver.go"]
CONF["配置文件<br/>etc/mcpserver.yaml"]
AUTH["双层认证系统<br/>common/mcpx/auth.go"]
STREAM["Streamable传输<br/>common/mcpx/server.go"]
WRAPPER["CallToolWrapper<br/>common/mcpx/wrapper.go"]
CTXPROP["上下文传播<br/>common/ctxprop/ctx.go"]
CTXDATA["上下文数据<br/>common/ctxdata/ctxData.go"]
SSEAUTH["SSE认证处理<br/>common/mcpx/sse_auth.go"]
MSERVER["多服务器配置<br/>common/mcpx/config.go"]
MCPCFG["MCP配置结构<br/>common/mcpx/config.go"]
ENDPOINT["工具名前缀<br/>ToolNameSeparator"]
LOGENHANCE["日志增强<br/>统一错误处理"]
RESPONSE["响应增强<br/>中文用户名"]
DEBUGLOG["调试日志<br/>Level: debug"]
PROGRESS["进度通知<br/>ProgressSender接口"]
TIMEOUT["超时配置<br/>SseTimeout/MessageTimeout"]
end
ENTRY --> CFG
ENTRY --> CTX
ENTRY --> REG
REG --> ECHO
REG --> MODBUS
REG --> TESTPROGRESS
CTX --> MODBUS
CFG --> CONF
AUTH --> STREAM
STREAM --> WRAPPER
WRAPPER --> CTXPROP
CTXPROP --> CTXDATA
CTXDATA --> SSEAUTH
ENTRY --> LOGENHANCE
LOGENHANCE --> RESPONSE
LOGENHANCE --> DEBUGLOG
MSERVER --> MCPCFG
MCPCFG --> ENDPOINT
TESTPROGRESS --> PROGRESS
PROGRESS --> WRAPPER
TIMEOUT --> STREAM
```

**图表来源**
- [mcpserver.go:28-33](file://aiapp/mcpserver/mcpserver.go#L28-L33)
- [config.go:8-12](file://aiapp/mcpserver/internal/config/config.go#L8-L12)
- [servicecontext.go:15-24](file://aiapp/mcpserver/internal/svc/servicecontext.go#L15-L24)
- [registry.go:10-15](file://aiapp/mcpserver/internal/tools/registry.go#L10-L15)
- [wrapper.go:23-70](file://common/mcpx/wrapper.go#L23-L70)

## 详细组件分析

### 配置与启动流程
- **配置项**
  - Name/Host/Port：服务基本监听信息
  - Mode: dev：开发环境标识
  - Mcp.UseStreamable: true：启用Streamable HTTP传输协议
  - Mcp.SseTimeout: 24h：SSE连接超时配置
  - Mcp.MessageTimeout: 300s：工具执行超时配置（5分钟）
  - Log：日志配置，包含Encoding、Path、Level: debug、KeepDays
  - Auth：认证配置段落，包含JwtSecrets和ServiceToken
  - BridgeModbusRpcConf：Modbus服务RPC客户端配置
- **启动流程**
  - 解析配置文件路径
  - 加载配置并打印Go版本
  - 禁用统计日志以优化启动性能
  - 创建服务上下文
  - 创建MCP服务器实例（使用mcpx.NewMcpServer）
  - 统一注册所有工具（包括新增的testprogress工具）
  - 启动服务

```mermaid
sequenceDiagram
participant CLI as "命令行"
participant Main as "mcpserver.go"
participant Conf as "配置加载"
participant Log as "日志配置"
participant Ctx as "服务上下文"
participant Reg as "工具注册"
participant Srv as "MCP服务器"
CLI->>Main : 传入配置文件路径
Main->>Conf : conf.MustLoad(配置文件)
Main->>Main : 打印Go版本
Main->>Log : logx.DisableStat()
Main->>Ctx : NewServiceContext(c)
Main->>Srv : mcpx.NewMcpServer(c.McpServerConf)
Main->>Reg : RegisterAll(server.Server(), svcCtx)
Reg->>Reg : RegisterEcho(server)
Reg->>Reg : RegisterModbus(server, svcCtx)
Reg->>Reg : RegisterTestProgress(server)
Main->>Srv : Start()
Srv-->>CLI : 服务就绪
```

**图表来源**
- [mcpserver.go:19-40](file://aiapp/mcpserver/mcpserver.go#L19-L40)
- [config.go:8-12](file://aiapp/mcpserver/internal/config/config.go#L8-L12)
- [servicecontext.go:15-24](file://aiapp/mcpserver/internal/svc/servicecontext.go#L15-L24)
- [registry.go:10-15](file://aiapp/mcpserver/internal/tools/registry.go#L10-L15)

**章节来源**
- [mcpserver.go:19-40](file://aiapp/mcpserver/mcpserver.go#L19-L40)
- [mcpserver.yaml:1-29](file://aiapp/mcpserver/etc/mcpserver.yaml#L1-L29)

### 工具注册与调用流程
- **统一注册机制**：RegisterAll函数统一管理所有工具的注册，包括新增的testprogress工具
- **工具隔离设计**：每个工具独立实现，便于维护和扩展
- **参数验证**：每个工具都有明确的参数结构定义
- **错误处理**：工具调用包含完善的错误处理机制
- **上下文传播**：所有工具现在使用CallToolWrapper进行统一的上下文处理

```mermaid
flowchart TD
Start(["收到工具调用"]) --> Check{"工具类型？"}
Check --> |echo| Echo["Echo工具处理"]
Check --> |read_holding_registers| Modbus1["Modbus读保持寄存器"]
Check --> |read_coils| Modbus2["Modbus读线圈"]
Check --> |test_progress| TestProgress["TestProgress工具处理"]
Echo --> Wrapper["CallToolWrapper包装器"]
Wrapper --> DebugLog["调试日志记录：捕获token和username"]
DebugLog --> Format1["格式化回显消息含中文用户名"]
Format1 --> Return1["返回结果"]
Modbus1 --> Wrapper2["CallToolWrapper包装器"]
Wrapper2 --> Extract["自动上下文提取"]
Extract --> Call1["调用BridgeModbusCli"]
Call1 --> Format2["格式化寄存器结果"]
Format2 --> Return2["返回结果"]
Modbus2 --> Wrapper3["CallToolWrapper包装器"]
Wrapper3 --> Extract2["自动上下文提取"]
Extract2 --> Call2["调用BridgeModbusCli"]
Call2 --> Format3["格式化线圈结果"]
Format3 --> Return3["返回结果"]
TestProgress --> Wrapper4["CallToolWrapper包装器"]
Wrapper4 --> Progress["进度发送器提取"]
Progress --> Loop["循环发送进度通知"]
Loop --> Sleep["等待间隔时间"]
Sleep --> Loop
Loop --> Complete["完成处理"]
Complete --> Return4["返回结果"]
Return1 --> End(["结束"])
Return2 --> End
Return3 --> End
Return4 --> End
```

**图表来源**
- [registry.go:10-15](file://aiapp/mcpserver/internal/tools/registry.go#L10-L15)
- [echo.go:25-42](file://aiapp/mcpserver/internal/tools/echo.go#L25-L42)
- [modbus.go:35-69](file://aiapp/mcpserver/internal/tools/modbus.go#L35-L69)
- [testprogress.go:27-69](file://aiapp/mcpserver/internal/tools/testprogress.go#L27-L69)
- [wrapper.go:31-70](file://common/mcpx/wrapper.go#L31-L70)

**章节来源**
- [registry.go:9-15](file://aiapp/mcpserver/internal/tools/registry.go#L9-L15)
- [echo.go:15-42](file://aiapp/mcpserver/internal/tools/echo.go#L15-L42)
- [modbus.go:28-129](file://aiapp/mcpserver/internal/tools/modbus.go#L28-L129)
- [testprogress.go:13-69](file://aiapp/mcpserver/internal/tools/testprogress.go#L13-L69)

## 双层认证系统

### 认证架构
MCP服务器实现了双层认证系统，提供灵活的安全机制：

```mermaid
graph LR
subgraph "双层认证系统"
A["客户端请求"] --> B["NewDualTokenVerifier"]
B --> C{"检查ServiceToken"}
C --> |匹配| D["连接级认证成功"]
C --> |不匹配| E{"检查JWT密钥"}
E --> |匹配| F["调用级认证成功"]
E --> |不匹配| G["认证失败"]
D --> H["返回TokenInfo(type=service)"]
F --> I["返回TokenInfo(包含JWT声明)"]
G --> J["返回ErrInvalidToken"]
end
```

**图表来源**
- [auth.go:15-48](file://common/mcpx/auth.go#L15-L48)

### 认证流程
1. **ServiceToken验证**：使用常量时间比较算法验证连接级服务令牌
2. **JWT验证**：如果ServiceToken验证失败，尝试解析JWT令牌
3. **TokenInfo生成**：根据验证结果生成相应的TokenInfo
4. **过期时间处理**：JWT令牌设置合理的过期时间

### 配置要求
- **JwtSecrets**：支持多个JWT密钥，提高安全性
- **ServiceToken**：连接级服务令牌，用于内部服务间通信
- **常量时间比较**：防止时序攻击

**章节来源**
- [auth.go:15-48](file://common/mcpx/auth.go#L15-L48)
- [mcpserver.yaml:14-22](file://aiapp/mcpserver/etc/mcpserver.yaml#L14-L22)

## Streamable HTTP传输协议

### 协议支持
MCP服务器支持两种传输协议：

```mermaid
graph TB
subgraph "传输协议选择"
A["Mcp.UseStreamable"] --> B{"UseStreamable=true?"}
B --> |是| C["setupStreamableTransport()"]
B --> |否| D["setupSSETransport()"]
C --> E["NewStreamableHTTPHandler"]
D --> F["newAuthSSEHandler"]
E --> G["MessageEndpoint /mcp/message"]
F --> H["SseEndpoint /sse"]
end
```

**图表来源**
- [server.go:64-110](file://common/mcpx/server.go#L64-L110)

### Streamable HTTP特性
- **2025-03-26规范**：支持最新的MCP协议规范
- **独立POST请求**：每次工具调用都是独立的HTTP POST请求
- **DELETE方法支持**：支持Streamable HTTP的DELETE方法
- **超时配置**：支持messageTimeout超时设置

### SSE传输对比
- **SSE协议**：基于Server-Sent Events的长连接
- **Streamable协议**：基于HTTP的流式传输
- **适用场景**：Streamable更适合工具调用场景，SSE适合持续连接场景

**章节来源**
- [server.go:64-140](file://common/mcpx/server.go#L64-L140)
- [mcpserver.yaml:6](file://aiapp/mcpserver/etc/mcpserver.yaml#L6)

## SSE传输协议改进

### SSE传输架构
MCP服务器新增了专门的SSE传输处理机制，解决了认证上下文提取的问题：

```mermaid
graph LR
subgraph "SSE传输改进架构"
A["客户端SSE连接"] --> B["newAuthSSEHandler"]
B --> C["创建authSSESession"]
C --> D["生成sessionID"]
D --> E["创建authSSETransport"]
E --> F["server.Connect()"]
F --> G["authSSEConn.Read()"]
G --> H["注入RequestExtra"]
H --> I["TokenInfo + Header"]
I --> J["工具处理器"]
end
```

**图表来源**
- [server.go:93-103](file://common/mcpx/server.go#L93-L103)

### SSE认证处理机制
1. **会话管理**：每个SSE连接创建独立的authSSESession
2. **POST请求认证**：在POST请求中捕获TokenInfo和HTTP头部
3. **会话上下文**：将认证信息存储在会话中供后续读取
4. **请求额外信息**：在工具调用时注入RequestExtra到请求中

### 会话生命周期管理
- **会话创建**：GET请求创建新的SSE会话，生成随机sessionID
- **会话存储**：使用互斥锁保护会话映射表
- **会话清理**：连接断开时自动清理会话资源
- **并发安全**：支持多会话并发处理

**章节来源**
- [server.go:93-103](file://common/mcpx/server.go#L93-L103)

## 超时配置管理

### 超时配置架构
**更新** MCP服务器现在支持精细化的超时配置管理：

```mermaid
graph LR
subgraph "超时配置管理"
A["Mcp.SseTimeout: 24h"] --> B["SSE连接超时配置"]
A --> C["Mcp.MessageTimeout: 300s"]
C --> D["工具执行超时配置"]
B --> E["registerRoutes()"]
D --> E
E --> F["rest.WithSSE()"]
E --> G["rest.WithTimeout()"]
F --> H["SSE连接处理"]
G --> I["工具调用处理"]
H --> J["长时间连接支持"]
I --> K["工具执行超时控制"]
end
```

**图表来源**
- [server.go:125-143](file://common/mcpx/server.go#L125-L143)
- [mcpserver.yaml:7-8](file://aiapp/mcpserver/etc/mcpserver.yaml#L7-L8)

### SSE连接超时配置
- **SseTimeout字段**：新增的SSE连接超时配置，支持24小时超时
- **配置位置**：在Mcp配置段落中设置
- **应用场景**：支持长时间连接场景，如实时数据流和进度通知
- **默认行为**：通过rest.WithSSE()和rest.WithTimeout()应用配置

### 工具执行超时配置
- **MessageTimeout字段**：新增的工具执行超时配置，默认5分钟
- **配置位置**：在Mcp配置段落中设置
- **应用场景**：控制工具调用的最大执行时间
- **性能优化**：防止长时间阻塞影响服务器性能

### 超时配置应用机制
1. **SSE连接超时**：通过registerRoutes方法配置SSE端点的超时处理
2. **工具执行超时**：通过registerRoutes方法配置POST端点的超时处理
3. **统一管理**：所有超时配置通过McpServerConf结构统一管理
4. **灵活配置**：支持不同场景下的超时时间调整

**章节来源**
- [server.go:125-143](file://common/mcpx/server.go#L125-L143)
- [mcpserver.yaml:7-8](file://aiapp/mcpserver/etc/mcpserver.yaml#L7-L8)

## 上下文传播机制

### 上下文传播架构
MCP服务器实现了完整的上下文传播机制，支持SSE和Streamable两种传输协议：

```mermaid
graph LR
subgraph "上下文传播流程"
A["客户端请求"] --> B["ctxHeaderTransport"]
B --> C["HTTP头部注入"]
C --> D["MCP服务器接收"]
D --> E["CallToolWrapper"]
E --> F["ExtractCtxFromHeader"]
F --> G["context.Context注入"]
G --> H["工具处理器"]
H --> I["统一错误处理"]
I --> J["性能监控"]
J --> K["自动日志记录"]
K --> L["进度发送器注入"]
L --> M["ProgressSender接口"]
end
```

**图表来源**
- [client.go:684-705](file://common/mcpx/client.go#L684-L705)
- [wrapper.go:31-70](file://common/mcpx/wrapper.go#L31-L70)

### CallToolWrapper现代化处理机制
**更新** MCP服务器现在使用CallToolWrapper进行统一的工具处理：

```mermaid
graph LR
subgraph "CallToolWrapper现代化处理"
A["原始工具处理器"] --> B["CallToolWrapper包装器"]
B --> C["统一日志记录"]
C --> D["自动上下文提取"]
D --> E["性能监控"]
E --> F["错误处理"]
F --> G["进度发送器注入"]
G --> H["ProgressSender接口"]
H --> I["返回结果"]
end
```

**图表来源**
- [wrapper.go:31-70](file://common/mcpx/wrapper.go#L31-L70)

### 上下文提取机制
1. **Streamable传输**：从req.Extra.Header和TokenInfo提取上下文
2. **SSE传输**：从req.Params._meta提取用户上下文
3. **SSE直连JWT**：从连接级TokenInfo提取上下文
4. **自动传播**：将提取的上下文注入到工具处理器的context中

### 进度发送器注入机制
- **上下文存储**：通过WithProgressSender将进度发送器存入context
- **上下文获取**：通过GetProgressSender从context获取进度发送器
- **自动注入**：在CallToolWrapper中自动检测_meta中的进度token并注入
- **接口设计**：ProgressSender接口提供SendProgress方法用于进度通知

### 错误处理和日志记录
- **统一错误处理**：所有工具调用都经过CallToolWrapper的错误处理
- **性能监控**：自动记录工具调用的执行时间和参数
- **日志记录**：成功和失败的工具调用都会被记录
- **参数序列化**：自动序列化工具参数用于日志记录

**章节来源**
- [wrapper.go:23-80](file://common/mcpx/wrapper.go#L23-L80)
- [client.go:684-705](file://common/mcpx/client.go#L684-L705)
- [ctxprop.go:12-78](file://common/ctxprop/ctx.go#L12-L78)
- [ctxData.go:5-67](file://common/ctxdata/ctxData.go#L5-L67)

## 工具实现详解

### Echo工具实现
- **参数结构**：包含message（必填）和prefix（可选）参数
- **功能特性**：支持自定义前缀的回显功能
- **日志增强**：**新增** 通过CallToolWrapper提供统一的日志记录，同时捕获token和username信息
- **响应格式**：返回TextContent格式的文本内容，**更新** 在响应消息中支持中文用户名显示
- **使用场景**：测试工具调用链路和验证MCP协议实现

### Modbus工具实现
- **读保持寄存器工具**：支持Function Code 0x03，返回多种数值表示
- **读线圈工具**：支持Function Code 0x01，返回线圈开关状态
- **参数验证**：包含地址范围和数量限制验证
- **结果格式化**：提供JSON格式的结果输出
- **错误处理**：完善的RPC调用错误处理机制
- **上下文传播**：使用CallToolWrapper进行统一的上下文处理

### TestProgress工具实现
- **参数结构**：包含steps（总步数）、interval（每步间隔毫秒）和message（可选消息）
- **功能特性**：模拟长时间运行的任务并发送进度更新
- **默认值处理**：steps默认5步，interval默认500ms
- **进度发送**：使用GetProgressSender获取进度发送器并发送实时进度
- **循环处理**：通过for循环模拟任务执行过程
- **日志记录**：记录每次进度发送的详细信息
- **响应格式**：返回TextContent格式的完成消息
- **上下文传播**：使用CallToolWrapper进行统一的上下文处理

```mermaid
graph LR
subgraph "TestProgress工具架构"
A["TestProgressArgs"] --> B["RegisterTestProgress"]
B --> C["CallToolWrapper包装器"]
C --> D["progressHandler函数"]
D --> E["steps参数验证"]
E --> F["interval参数验证"]
F --> G["获取进度发送器"]
G --> H["循环发送进度"]
H --> I["time.Sleep间隔"]
I --> H
H --> J["返回完成结果"]
A2["参数结构"] --> B
A3["默认值处理"] --> D
A4["进度发送器"] --> G
A5["循环控制"] --> H
end
```

**图表来源**
- [testprogress.go:13-18](file://aiapp/mcpserver/internal/tools/testprogress.go#L13-L18)
- [testprogress.go:20-69](file://aiapp/mcpserver/internal/tools/testprogress.go#L20-L69)
- [wrapper.go:31-70](file://common/mcpx/wrapper.go#L31-L70)

**章节来源**
- [echo.go:9-42](file://aiapp/mcpserver/internal/tools/echo.go#L9-L42)
- [modbus.go:14-129](file://aiapp/mcpserver/internal/tools/modbus.go#L14-L129)
- [testprogress.go:13-69](file://aiapp/mcpserver/internal/tools/testprogress.go#L13-L69)

## CallToolWrapper现代化工具处理

### CallToolWrapper架构
**更新** MCP服务器现在使用CallToolWrapper提供现代化的工具处理机制：

```mermaid
graph LR
subgraph "CallToolWrapper现代化处理机制"
A["原始工具处理器"] --> B["CallToolWrapper(h)"]
B --> C["统一日志记录"]
C --> D["性能监控"]
D --> E["自动上下文提取"]
E --> F["进度发送器注入"]
F --> G["错误处理"]
G --> H["返回结果"]
end
```

**图表来源**
- [wrapper.go:31-70](file://common/mcpx/wrapper.go#L31-L70)

### 日志记录增强
- **统一日志格式**：所有工具调用都经过统一的日志记录机制
- **执行时间监控**：自动记录工具调用的执行时间
- **参数序列化**：自动序列化工具参数用于日志记录
- **错误分类**：区分成功和失败的日志记录
- **上下文信息**：记录工具名称和执行上下文

### 上下文提取机制
- **Streamable传输路径**：从req.Extra.Header和TokenInfo提取上下文
- **SSE传输路径**：从req.Params._meta提取用户上下文
- **SSE直连JWT路径**：从连接级TokenInfo提取上下文
- **自动传播**：将提取的上下文注入到工具处理器的context中

### 进度发送器注入机制
- **上下文存储**：通过WithProgressSender将进度发送器存入context
- **上下文获取**：通过GetProgressSender从context获取进度发送器
- **自动注入**：在CallToolWrapper中自动检测_meta中的进度token并注入
- **接口设计**：ProgressSender接口提供SendProgress方法用于进度通知

### 错误处理机制
- **统一错误处理**：所有工具调用都经过CallToolWrapper的错误处理
- **性能监控**：自动记录工具调用的执行时间和参数
- **日志记录**：成功和失败的工具调用都会被记录
- **参数序列化**：自动序列化工具参数用于日志记录

**章节来源**
- [wrapper.go:23-80](file://common/mcpx/wrapper.go#L23-L80)

## 日志增强功能

### 日志增强架构
**更新** MCP服务器的echo工具现在通过CallToolWrapper具备了增强的日志功能：

```mermaid
graph LR
subgraph "日志增强功能"
A["Echo工具调用"] --> B["CallToolWrapper包装器"]
B --> C["统一日志记录"]
C --> D["token: %s,username: %s"]
D --> E["返回增强响应"]
E --> F["中文用户名显示"]
end
```

**图表来源**
- [echo.go:25-42](file://aiapp/mcpserver/internal/tools/echo.go#L25-L42)
- [wrapper.go:37-44](file://common/mcpx/wrapper.go#L37-L44)

### 日志增强特性
- **统一日志记录**：通过CallToolWrapper提供统一的日志记录机制
- **token捕获**：通过ctxdata.GetAuthorization(ctx)获取用户认证令牌
- **username捕获**：通过ctxdata.GetUserName(ctx)获取用户名信息
- **调试输出**：使用logx.WithContext(ctx).WithDuration记录详细信息
- **格式化输出**：采用统一的日志格式"token: %s,username: %s"
- **中文支持**：响应消息中直接显示中文用户名

### 日志输出格式
- **调试级别**：使用debug级别日志，便于开发和调试
- **统一格式**：所有日志输出采用相同的格式规范
- **性能监控**：记录工具调用的执行时间和参数
- **敏感信息**：Authorization令牌在日志中可能需要脱敏处理

### 响应消息增强
- **中文用户名**：在响应消息中直接显示中文用户名
- **用户友好**：提升用户体验，无需额外的用户名转换
- **国际化支持**：支持中文用户的本地化体验

**章节来源**
- [echo.go:25-42](file://aiapp/mcpserver/internal/tools/echo.go#L25-L42)
- [wrapper.go:37-44](file://common/mcpx/wrapper.go#L37-L44)
- [ctxData.go:47-59](file://common/ctxdata/ctxData.go#L47-L59)

## 配置管理增强

### 认证配置
- **JwtSecrets**：支持多个JWT密钥，提高安全性
- **ServiceToken**：连接级服务令牌，用于内部服务间通信
- **配置验证**：支持空配置，无认证时跳过认证中间件

### 传输协议配置
- **UseStreamable**：启用Streamable HTTP传输协议
- **MessageTimeout**：工具调用消息超时时间（默认300秒，现配置为5分钟）
- **Cors**：允许的跨域来源列表

### 超时配置
- **SseTimeout**：SSE连接超时时间（默认24小时）
- **MessageTimeout**：工具执行超时时间（默认30秒，现配置为5分钟）
- **配置应用**：通过registerRoutes方法分别应用到SSE和POST端点

### 配置文件结构更新
- **YAML结构**：支持多层嵌套配置
- **配置层次**：Name、Host、Port、Mode、Mcp、Log、Auth、BridgeModbusRpcConf等配置项
- **配置验证**：确保配置项的完整性和正确性

**章节来源**
- [mcpserver.yaml:5-29](file://aiapp/mcpserver/etc/mcpserver.yaml#L5-L29)

## 多服务器连接管理

### 服务器配置结构
MCP客户端SDK支持多服务器连接管理，通过ServerConfig结构实现：

```mermaid
graph TB
subgraph "多服务器连接管理"
A["Config.Servers"] --> B["ServerConfig数组"]
B --> C["Name: 工具名前缀"]
B --> D["Endpoint: MCP服务器端点"]
B --> E["ServiceToken: 连接级鉴权令牌"]
B --> F["UseStreamable: 传输协议选择"]
G["客户端连接"] --> H["serverConn结构"]
H --> I["tryConnect(): 建立连接"]
H --> J["onChange(): 状态变更通知"]
I --> K["session.Tools(): 加载工具"]
K --> L["工具路由映射"]
end
```

**图表来源**
- [server.go:11-22](file://common/mcpx/server.go#L11-L22)
- [client.go:208-247](file://common/mcpx/client.go#L208-L247)

### 服务器连接流程
1. **配置加载**：从Config.Servers数组加载多个服务器配置
2. **连接建立**：每个ServerConfig创建对应的serverConn实例
3. **工具加载**：连接成功后加载服务器上的工具列表
4. **状态管理**：断开连接时自动重连，支持RefreshInterval配置
5. **工具路由**：为每个工具添加服务器Name前缀，避免命名冲突

### AI聊天应用集成
AI聊天应用通过Mcpx配置实现多服务器连接：

```mermaid
graph LR
subgraph "AI聊天应用配置"
A["aichat.yaml"] --> B["Mcpx.Servers数组"]
B --> C["ServerConfig配置"]
C --> D["Name: mcpserver"]
C --> E["Endpoint: http://localhost:13003/message"]
C --> F["ServiceToken: 内部令牌"]
C --> G["UseStreamable: true"]
H["服务上下文"] --> I["mcpx.NewClient(mcpCfg)"]
I --> J["多服务器客户端"]
J --> K["工具聚合"]
end
```

**图表来源**
- [aichat.yaml:8-17](file://aiapp/aichat/etc/aichat.yaml#L8-L17)
- [servicecontext.go:24-28](file://aiapp/aichat/internal/svc/servicecontext.go#L24-L28)

**章节来源**
- [server.go:11-22](file://common/mcpx/server.go#L11-L22)
- [client.go:208-247](file://common/mcpx/client.go#L208-L247)
- [aichat.yaml:8-17](file://aiapp/aichat/etc/aichat.yaml#L8-L17)
- [servicecontext.go:24-28](file://aiapp/aichat/internal/svc/servicecontext.go#L24-L28)

## JWT密钥配置增强

### 多密钥支持架构
MCP服务器增强了JWT认证系统的安全性，支持多JWT密钥配置：

```mermaid
graph LR
subgraph "JWT密钥配置增强"
A["JwtSecrets数组"] --> B["多个JWT密钥"]
B --> C["密钥轮换支持"]
B --> D["并行验证机制"]
C --> E["平滑密钥切换"]
D --> F["提高系统安全性"]
G["认证验证"] --> H["NewDualTokenVerifier"]
H --> I["遍历JwtSecrets验证"]
I --> J["支持历史密钥"]
end
```

**图表来源**
- [mcpserver.yaml:15-17](file://aiapp/mcpserver/etc/mcpserver.yaml#L15-L17)
- [auth.go:15-48](file://common/mcpx/auth.go#L15-L48)

### 密钥配置示例
配置文件中的JWT密钥配置示例：

```yaml
Auth:
  JwtSecrets:
    - "629c6233-1a76-471b-bd25-b87208762219"
  ServiceToken: "mcp-internal-service-token-2026"
```

### 密钥轮换策略
- **历史密钥支持**：系统同时验证当前和历史JWT密钥
- **平滑过渡**：新旧密钥并存期间，确保认证连续性
- **安全升级**：定期轮换JWT密钥，提升系统安全性

**章节来源**
- [mcpserver.yaml:15-17](file://aiapp/mcpserver/etc/mcpserver.yaml#L15-L17)
- [auth.go:15-48](file://common/mcpx/auth.go#L15-L48)

## 工具名前缀路由

### 前缀路由机制
MCP客户端SDK通过工具名前缀实现多服务器工具路由隔离：

```mermaid
graph TB
subgraph "工具名前缀路由"
A["ToolNameSeparator: '__'"] --> B["工具名前缀分隔符"]
B --> C["服务器Name前缀"]
C --> D["工具名组合"]
D --> E["echo__mcpserver"]
D --> F["read_holding_registers__mcpserver"]
D --> G["test_progress__mcpserver"]
G --> H["test_progress工具"]
end
```

**图表来源**
- [server.go:8](file://common/mcpx/server.go#L8)
- [client.go:178-182](file://common/mcpx/client.go#L178-L182)

### 前缀生成规则
- **分隔符**：使用双下划线"__"作为分隔符
- **命名规则**：工具名 = 原工具名 + "__" + 串联符服务器Name
- **自动处理**：SDK自动为每个工具添加服务器前缀
- **路由隔离**：不同服务器的同名工具不会冲突

### 实际应用示例
- **单服务器**：echo → echo
- **多服务器**：echo → echo__mcpserver, read_holding_registers__mcpserver, test_progress__mcpserver
- **工具调用**：AI代理通过前缀路由调用指定服务器的工具

**章节来源**
- [server.go:8](file://common/mcpx/server.go#L8)
- [client.go:178-182](file://common/mcpx/client.go#L178-L182)

## 日志配置优化

### 日志级别调整
**更新** MCP服务器采用了更详细的日志记录策略，将日志级别从info调整为debug：

```mermaid
graph TB
subgraph "日志配置优化"
A["配置文件"] --> B["Log.Level: debug"]
A --> C["Log.Encoding: plain"]
A --> D["Log.Path: /opt/logs/mcpserver"]
A --> E["Log.KeepDays: 300"]
F["启动流程"] --> G["logx.DisableStat()"]
G --> H["禁用统计日志"]
B --> I["启用详细调试日志"]
C --> J["纯文本格式"]
D --> K["标准日志路径"]
E --> L["长期保留策略"]
```

**图表来源**
- [mcpserver.go:25-26](file://aiapp/mcpserver/mcpserver.go#L25-L26)
- [mcpserver.yaml:7-12](file://aiapp/mcpserver/etc/mcpserver.yaml#L7-L12)

### 日志级别调整特性
- **Debug级别**：提供更详细的调试信息，便于问题诊断和开发调试
- **纯文本格式**：使用plain编码，便于日志分析和处理
- **标准路径**：日志文件保存在/opt/logs/mcpserver目录
- **长期保留**：保留300天的日志，支持长期审计需求
- **启动优化**：禁用统计日志，减少启动阶段的性能开销

### 启动流程优化
- **统计日志禁用**：在服务启动时调用logx.DisableStat()，避免统计信息干扰
- **Go版本显示**：启动时显示Go运行时版本信息
- **资源清理**：服务停止时自动清理资源

**章节来源**
- [mcpserver.go:25-26](file://aiapp/mcpserver/mcpserver.go#L25-L26)
- [mcpserver.yaml:7-12](file://aiapp/mcpserver/etc/mcpserver.yaml#L7-L12)

## 依赖分析
- **go-zero mcp包**：核心MCP服务器功能
- **go-zero zrpc包**：RPC客户端通信支持
- **modelcontextprotocol/go-sdk**：MCP协议SDK
- **项目模块依赖**：go.mod中声明的github.com/zeromicro/go-zero v1.10.0
- **Modbus服务依赖**：app/bridgemodbus模块提供Modbus协议支持
- **第三方依赖**：grid-x/modbus用于Modbus协议实现

```mermaid
graph TB
M["go.mod<br/>require github.com/zeromicro/go-zero v1.10.0"] --> P["mcpserver.go<br/>导入mcp包"]
P --> S["MCP服务器实例"]
P --> Z["zrpc客户端"]
Z --> B["BridgeModbusCli"]
B --> BM["bridgemodbus服务"]
M --> SDK["modelcontextprotocol/go-sdk"]
SDK --> AUTH["认证系统"]
SDK --> STREAM["传输协议"]
SDK --> CTX["上下文传播"]
SDK --> MSERVER["多服务器管理"]
SDK --> PREFIX["工具名前缀"]
SDK --> SSEAUTH["SSE认证处理"]
SDK --> LOGENHANCE["日志增强"]
SDK --> RESPONSE["响应增强"]
SDK --> WRAPPER["CallToolWrapper"]
SDK --> PROGRESS["进度通知"]
SDK --> TIMEOUT["超时配置"]
```

**图表来源**
- [go.mod:50](file://go.mod#L50)
- [mcpserver.go:12-14](file://aiapp/mcpserver/mcpserver.go#L12-L14)
- [servicecontext.go:18-23](file://aiapp/mcpserver/internal/svc/servicecontext.go#L18-L23)
- [wrapper.go:1-15](file://common/mcpx/wrapper.go#L1-L15)

**章节来源**
- [go.mod:50](file://go.mod#L50)
- [mcpserver.go:12-14](file://aiapp/mcpserver/mcpserver.go#L12-L14)
- [servicecontext.go:18-23](file://aiapp/mcpserver/internal/svc/servicecontext.go#L18-L23)
- [wrapper.go:1-15](file://common/mcpx/wrapper.go#L1-L15)

## 性能考量
- **模块化优势**：清晰的职责分离，便于性能优化和资源管理
- **服务上下文复用**：统一的服务上下文管理，避免重复初始化
- **工具注册优化**：统一注册机制，减少工具加载开销
- **RPC调用优化**：Modbus工具通过RPC调用，支持连接池和超时控制
- **认证优化**：双层认证系统支持常量时间比较，防止时序攻击
- **传输协议优化**：Streamable HTTP协议适合工具调用场景，减少连接开销
- **上下文传播优化**：高效的头部映射和上下文注入机制
- **SSE传输优化**：新增的SSE认证处理机制确保认证上下文的一致性
- **日志管理优化**：启动前可选择关闭统计日志，降低启动阶段开销
- **环境配置优化**：开发环境标识便于调试和性能分析
- **日志级别优化**：debug级别提供详细信息，同时通过禁用统计日志优化性能
- **多服务器连接优化**：支持连接池和自动重连，提升系统可用性
- **JWT密钥优化**：多密钥支持允许平滑密钥轮换，不影响系统运行
- **工具路由优化**：前缀路由避免工具名冲突，提升工具管理效率
- **日志增强优化**：**新增** 通过CallToolWrapper提供统一的日志记录机制，提升调试效率
- **响应增强优化**：**新增** 中文用户名显示提升用户体验，无需额外处理开销
- **CallToolWrapper优化**：**新增** 统一的工具处理机制，提供更好的性能监控和错误处理
- **上下文提取优化**：**新增** 三种认证路径的自动上下文提取，提升工具调用可靠性
- **进度通知优化**：**新增** 基于上下文的进度发送器，支持实时进度更新
- **testprogress工具优化**：**新增** 可配置的步骤数和间隔时间，支持长时间任务的进度反馈
- **超时配置优化**：**新增** SSE连接超时和工具执行超时配置，支持更灵活的部署场景
- **长时间连接支持**：**新增** 通过SseTimeout配置支持24小时的SSE连接超时
- **工具执行性能**：**新增** 通过MessageTimeout配置优化5分钟的工具执行超时控制

**章节来源**
- [mcpserver.go:25-26](file://aiapp/mcpserver/mcpserver.go#L25-L26)
- [mcpserver.yaml:4](file://aiapp/mcpserver/etc/mcpserver.yaml#L4)
- [servicecontext.go:15-24](file://aiapp/mcpserver/internal/svc/servicecontext.go#L15-L24)
- [wrapper.go:31-70](file://common/mcpx/wrapper.go#L31-L70)
- [testprogress.go:27-69](file://aiapp/mcpserver/internal/tools/testprogress.go#L27-L69)

## 故障排查指南
- **配置加载失败**
  - 确认配置文件路径正确，且etc/mcpserver.yaml存在
  - 检查配置项格式是否符合YAML规范
  - 验证BridgeModbusRpcConf配置的RPC服务可达性
  - 检查Auth配置的JwtSecrets和ServiceToken格式
  - **新增** 验证SseTimeout和MessageTimeout配置格式
- **认证失败**
  - 检查JWT令牌是否在JwtSecrets中配置
  - 验证ServiceToken是否正确配置
  - 确认客户端发送的Authorization头部格式
  - 查看认证日志中的具体错误信息
- **工具调用异常**
  - 检查工具schema定义与参数传递是否一致
  - 查看参数解析与处理逻辑，定位错误分支
  - 对于Modbus工具，检查Modbus设备连接状态
  - 验证上下文传播是否正常工作
  - **新增** 检查CallToolWrapper是否正确包装工具处理器
  - **新增** 检查testprogress工具的进度发送器是否正确注入
- **传输协议问题**
  - 确认UseStreamable配置与客户端兼容
  - 检查MessageTimeout和Cors配置
  - 验证Streamable HTTP和SSE端点配置
- **SSE传输问题**
  - **新增** 检查SSE会话ID生成和管理
  - **新增** 验证POST请求中的认证信息捕获
  - **新增** 确认SSE认证处理机制正常工作
  - **新增** 检查会话上下文的TokenInfo提取
- **RPC调用失败**
  - 检查bridgemodbus服务是否正常运行
  - 验证Modbus设备地址和参数范围
  - 查看RPC超时和连接池配置
- **上下文传播问题**
  - 检查HTTP头部是否正确注入
  - 验证context键值对的映射关系
  - 确认CallToolWrapper包装器是否正确使用
  - **新增** 验证三种认证路径的上下文提取机制
  - **新增** 检查进度发送器的上下文注入机制
- **多服务器连接问题**
  - 检查Config.Servers配置格式
  - 验证服务器Endpoint可达性
  - 确认ServiceToken配置正确
  - 查看服务器连接日志
- **工具名前缀问题**
  - 检查ToolNameSeparator配置
  - 验证工具名前缀生成规则
  - 确认工具路由映射正确
- **JWT密钥问题**
  - 检查JwtSecrets数组格式
  - 验证密钥长度和格式
  - 确认密钥轮换过程
- **日志问题**
  - 检查Log.Encoding配置的编码格式是否支持
  - 验证Log.Path目录的可写权限
  - 确认开发环境标识Mode: dev的正确性
  - 验证日志级别是否为debug以获取详细信息
  - **新增** 检查CallToolWrapper的日志记录功能
  - **新增** 验证统一日志输出格式
- **启动性能问题**
  - 确认logx.DisableStat()调用是否正确执行
  - 检查日志文件权限和磁盘空间
  - 验证配置文件的语法正确性
- **响应消息问题**
  - **新增** 检查中文用户名显示是否正常
  - **新增** 验证响应消息格式是否符合预期
  - **新增** 确认用户名编码和字符集支持
- **CallToolWrapper问题**
  - **新增** 检查工具处理器是否正确使用CallToolWrapper包装
  - **新增** 验证上下文提取机制是否正常工作
  - **新增** 确认日志记录和错误处理功能正常
  - **新增** 检查进度发送器的注入和提取机制
- **日志级别问题**
  - **新增** 确认日志级别已调整为debug以获取详细信息
  - **新增** 验证debug级别日志的输出格式和内容
- **testprogress工具问题**
  - **新增** 检查steps参数是否为正数
  - **新增** 验证interval参数是否大于0
  - **新增** 确认进度发送器是否正确获取
  - **新增** 检查循环执行和进度发送逻辑
  - **新增** 验证默认值处理机制
- **超时配置问题**
  - **新增** 检查SseTimeout配置是否为有效的24小时格式
  - **新增** 验证MessageTimeout配置是否为有效的300秒格式
  - **新增** 确认超时配置是否正确应用到SSE和POST端点
  - **新增** 验证超时配置对服务器性能的影响

**章节来源**
- [mcpserver.go:22-23](file://aiapp/mcpserver/mcpserver.go#L22-L23)
- [mcpserver.yaml:7-8](file://aiapp/mcpserver/etc/mcpserver.yaml#L7-L8)
- [mcpserver.yaml:10-18](file://aiapp/mcpserver/etc/mcpserver.yaml#L10-L18)
- [wrapper.go:31-70](file://common/mcpx/wrapper.go#L31-L70)
- [testprogress.go:27-69](file://aiapp/mcpserver/internal/tools/testprogress.go#L27-L69)

## 结论
本MCP服务器在本仓库中提供了高度模块化的实现范例，展示了如何基于go-zero构建企业级的MCP服务。当前实现具有以下特点：

- **现代化工具处理机制**：采用CallToolWrapper统一包装所有工具，提供更好的日志记录、上下文提取和错误处理能力
- **双层认证系统**：支持JWT令牌和连接级服务令牌的双重验证机制
- **Streamable HTTP传输协议**：实现2025年3月26日规范的流式HTTP传输
- **SSE传输协议改进**：新增的SSE认证处理机制，确保认证上下文的一致性
- **超时配置管理**：**新增** 支持SSE连接超时和工具执行超时配置，提供更灵活的部署场景
- **上下文传播机制**：实现HTTP头部与上下文的双向映射和自动传播
- **模块化架构**：完全符合go-zero合规布局，便于维护和扩展
- **增强配置管理**：支持Auth配置段落、useStreamable标志和超时配置
- **现代化配置结构**：采用简化的配置层次，提供更好的可读性和维护性
- **优化日志配置**：debug级别日志提供详细信息，同时通过禁用统计日志优化启动性能
- **工具丰富性**：包含基础的echo工具、实用的Modbus工具和新增的testprogress进度通知工具
- **服务集成**：与bridgemodbus服务深度集成，提供工业协议支持
- **配置灵活**：支持MCP配置和RPC配置的统一管理
- **多服务器连接管理**：支持多MCP服务器连接和工具名前缀路由
- **JWT密钥配置增强**：支持多JWT密钥配置，提升安全性和密钥轮换能力
- **认证上下文提取**：新增的SSE传输认证上下文提取功能，确保工具调用的一致性
- **日志增强功能**：**新增** 通过CallToolWrapper提供统一的日志记录机制，提升调试效率
- **响应消息增强**：**新增** 在响应消息中支持中文用户名显示，提升用户体验
- **日志级别优化**：**新增** 将日志级别调整为debug，提供更详细的调试信息用于开发和故障排除
- **进度通知功能**：**新增** testprogress工具支持实时进度更新，用于演示长时间任务的进度反馈
- **CallToolWrapper现代化**：**新增** 工具注册系统已从WithCtxProp迁移到CallToolWrapper，提供更好的日志记录、上下文提取和错误处理能力
- **统一工具包装器**：**新增** 所有工具现在使用CallToolWrapper进行统一的上下文处理和日志记录
- **增强的错误处理**：**新增** CallToolWrapper提供统一的错误处理和性能监控
- **改进的日志记录**：**新增** 自动记录工具调用的成功和失败情况，包含执行时间和参数信息
- **现代化的上下文传播**：**新增** 通过CallToolWrapper实现更可靠的上下文提取和传播机制
- **三种认证路径支持**：**新增** 支持Streamable传输、SSE + mcpx.Client和SSE直连JWT三种认证路径
- **自动上下文提取**：**新增** 根据不同的传输协议自动提取和传播用户上下文
- **详细调试日志**：**新增** 将日志级别调整为debug，提供更详细的调试信息用于开发和故障排除
- **超时配置优化**：**新增** SSE连接超时（24小时）和工具执行超时（5分钟）配置，支持更灵活的部署场景
- **长时间连接支持**：**新增** 通过SseTimeout配置增强长时间连接支持能力
- **工具执行性能优化**：**新增** 通过MessageTimeout配置优化工具执行超时控制

**新增** 本次更新特别增强了MCP服务器的超时配置管理能力，通过以下改进提升了系统的灵活性和稳定性：

- **新增SseTimeout配置**：支持24小时的SSE连接超时设置，适用于长时间连接场景
- **新增MessageTimeout配置**：支持5分钟的工具执行超时设置，默认30秒，现配置为5分钟
- **超时配置应用**：通过registerRoutes方法分别配置SSE连接超时和工具执行超时
- **灵活部署支持**：为不同部署场景提供更精细的超时控制能力
- **性能优化**：通过合理的超时配置平衡系统性能和用户体验
- **长时间连接增强**：支持更长时间的SSE连接，适用于实时数据流和进度通知场景

后续演进方向：
- 增强工具安全策略（如鉴权、限流）
- 完善监控与可观测性指标
- 扩展更多工业协议工具
- 增加工具版本管理和热更新机制
- 优化日志配置和环境特定的配置管理
- 支持多服务器连接和负载均衡
- 实现更精细的工具路由和权限控制
- 增密集成自动化密钥轮换工具
- **新增** 扩展CallToolWrapper功能到其他工具类型
- **新增** 增强响应消息的国际化支持
- **新增** 优化CallToolWrapper的性能监控和错误处理机制
- **新增** 利用debug级别日志进行更深入的故障诊断和性能分析
- **新增** 实现更完善的进度通知机制和实时反馈功能
- **新增** 扩展testprogress工具的功能，支持更多类型的进度通知场景
- **新增** 优化超时配置管理，支持动态超时调整和监控
- **新增** 增强长时间连接的稳定性和可靠性保障

## 附录

### 配置项说明
- **Name**：服务名称
- **Host**：监听主机
- **Port**：监听端口
- **Mode**：开发环境标识（dev）
- **Mcp.UseStreamable**：启用Streamable HTTP传输协议
- **Mcp.SseTimeout**：**新增** SSE连接超时配置（默认24小时）
- **Mcp.MessageTimeout**：**新增** 工具执行超时配置（默认300秒，现配置为5分钟）
- **Log.Encoding**：日志编码格式（plain）
- **Log.Path**：日志文件路径（/opt/logs/mcpserver）
- **Log.Level**：日志级别（debug）
- **Log.KeepDays**：日志保留天数（300）
- **Auth.JwtSecrets**：JWT密钥数组
- **Auth.ServiceToken**：服务令牌
- **Auth.ClaimMapping**：**新增** JWT声明映射配置
- **BridgeModbusRpcConf.Endpoints**：Modbus服务RPC端点
- **BridgeModbusRpcConf.NonBlock**：非阻塞模式
- **BridgeModbusRpcConf.Timeout**：RPC调用超时时间

**章节来源**
- [mcpserver.yaml:1-29](file://aiapp/mcpserver/etc/mcpserver.yaml#L1-L29)

### 工具参数说明

#### Echo工具参数
- **message**：必填字符串，要回显的消息
- **prefix**：可选字符串，添加到回显消息前的前缀，默认"Echo: "
- **日志增强**：**新增** 通过CallToolWrapper提供统一的日志记录，自动捕获并记录token和username信息
- **响应增强**：**新增** 在响应消息中显示中文用户名

#### Modbus工具参数

##### 读保持寄存器参数
- **modbusCode**：可选字符串，Modbus配置编码，空则使用默认配置
- **address**：必填整数，起始寄存器地址
- **quantity**：必填整数，读取数量(1-125)

##### 读线圈参数
- **modbusCode**：可选字符串，Modbus配置编码，空则使用默认配置
- **address**：必填整数，起始线圈地址
- **quantity**：必填整数，读取数量(1-2000)

#### TestProgress工具参数
- **steps**：可选整数，总步数，默认5步
- **interval**：可选整数，每步间隔毫秒，默认500ms
- **message**：可选字符串，进度消息，默认"处理中..."
- **进度发送**：**新增** 使用GetProgressSender获取进度发送器并发送实时进度
- **循环处理**：**新增** 通过for循环模拟任务执行过程
- **日志记录**：**新增** 记录每次进度发送的详细信息

### 多服务器配置示例

#### MCP服务器配置
```yaml
Name: mcpserver
Host: 0.0.0.0
Port: 13003
Mode: dev
Mcp:
  UseStreamable: true
  SseTimeout: 24h           # SSE 连接超时
  MessageTimeout: 300s      # 工具执行超时，默认 30s，改为 5 分钟
Log:
  Encoding: plain
  Path: /opt/logs/mcpserver
  Level: debug
  KeepDays: 300
Auth:
  JwtSecrets:
    - "629c6233-1a76-471b-bd25-b87208762219"
  ServiceToken: "mcp-internal-service-token-2026"
  ClaimMapping:
    user-id: "user_id"
    user-name: "user_name"
    dept-code: "dept_code"
BridgeModbusRpcConf:
  Endpoints:
    - 127.0.0.1:25004
  NonBlock: true
  Timeout: 10000
```

#### AI聊天应用多服务器配置
```yaml
Name: aichat.rpc
ListenOn: 0.0.0.0:23001
Mode: dev
Timeout: 60000
StreamTimeout: 600s
StreamIdleTimeout: 90s
MaxToolRounds: 10
Mcpx:
  Servers:
    - Name: "mcpserver"
      Endpoint: "http://localhost:13003/message"
      UseStreamable: true
      ServiceToken: "mcp-internal-service-token-2026"
  RefreshInterval: 30s
  ConnectTimeout: 10s
Log:
  Encoding: plain
  Path: /opt/logs/aichat
  Level: info
  KeepDays: 300
```

### 最佳实践
- **使用模块化结构**：遵循go-zero合规布局，便于团队协作
- **配置分离**：将MCP配置和RPC配置分离管理
- **环境配置**：合理使用Mode开发环境标识
- **认证配置**：正确配置JwtSecrets和ServiceToken
- **传输协议选择**：根据使用场景选择合适的传输协议
- **上下文传播**：确保上下文传播机制正常工作
- **工具注册**：通过统一的RegisterAll函数管理工具注册
- **错误处理**：为每个工具实现完善的错误处理机制
- **性能优化**：利用服务上下文复用资源，减少初始化开销
- **日志管理**：合理配置日志级别和保留策略，平衡调试需求和存储成本
- **多服务器管理**：合理规划服务器数量和命名，避免工具名冲突
- **JWT密钥管理**：定期轮换密钥，确保系统安全
- **工具路由设计**：合理设计工具名前缀，便于多服务器工具管理
- **SSE传输优化**：充分利用新增的SSE认证处理机制，确保工具调用的一致性
- **日志增强实践**：**新增** 利用CallToolWrapper提供的统一日志记录功能提升调试效率
- **响应增强实践**：**新增** 利用中文用户名显示提升用户体验
- **上下文传播最佳实践**：**新增** 确保三种认证路径下的上下文提取和传播机制正常工作
- **CallToolWrapper最佳实践**：**新增** 正确使用CallToolWrapper包装工具处理器，确保统一的日志记录和错误处理
- **现代化工具处理**：**新增** 采用CallToolWrapper替代传统的WithCtxProp，提升工具处理的可靠性和可维护性
- **调试日志优化**：**新增** 利用debug级别日志进行深入的故障诊断和性能分析
- **进度通知最佳实践**：**新增** 正确使用GetProgressSender获取进度发送器，确保进度通知的准确性和及时性
- **testprogress工具最佳实践**：**新增** 合理设置steps和interval参数，确保进度通知的用户体验
- **上下文传播最佳实践**：**新增** 确保进度发送器的上下文注入和提取机制正常工作
- **超时配置最佳实践**：**新增** 合理配置SseTimeout和MessageTimeout，平衡连接稳定性和性能表现
- **长时间连接最佳实践**：**新增** 利用24小时SseTimeout配置支持长时间连接场景
- **工具执行超时最佳实践**：**新增** 通过5分钟MessageTimeout配置优化工具执行超时控制

**章节来源**
- [mcpserver.go:17](file://aiapp/mcpserver/mcpserver.go#L17)
- [mcpserver.go:25-26](file://aiapp/mcpserver/mcpserver.go#L25-L26)
- [registry.go:9-15](file://aiapp/mcpserver/internal/tools/registry.go#L9-L15)
- [mcpserver.yaml:4-7](file://aiapp/mcpserver/etc/mcpserver.yaml#L4-L7)
- [aichat.yaml:8-17](file://aiapp/aichat/etc/aichat.yaml#L8-L17)
- [wrapper.go:23-70](file://common/mcpx/wrapper.go#L23-L70)
- [testprogress.go:13-69](file://aiapp/mcpserver/internal/tools/testprogress.go#L13-L69)