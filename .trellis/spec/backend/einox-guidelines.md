# EinoX AI Agent 框架规范

> `common/einox` 是基于 CloudWeGo Eino / Eino ADK 的 AI Agent 封装。提供 Agent 工厂、工具管理、模型适配、运行时、中间件和协议事件。

## When to read

- 新增 Agent 类型、工具、中间件或模型提供商。
- 修改 `aiapp/aisolo` 或 `aiapp/aichat` 的 Agent 构造或运行时逻辑。
- 对接新的 AI 后端（OpenAI 兼容、火山 ARK 等）。
- 使用中断（Interrupt）类中间件（审批、选择、表单、文本输入、信息确认）。

## 包结构

| 子包 | 职责 | source |
|------|------|--------|
| `agent/` | Agent 工厂（ChatModel、Deep、PlanExecute、Supervisor、Sequential、Parallel、Loop） | `common/einox/agent/` |
| `model/` | ChatModel 多厂商适配（OpenAI、DeepSeek、Ollama、Qwen、ARK、Claude） | `common/einox/model/chatmodel.go` |
| `tool/` | 工具注册（Kit）和能力白名单（Policy），以及 MCP 工具适配 | `common/einox/tool/` |
| `tool/builtin/` | 内置工具实现（compute/io/human capability） | `common/einox/tool/builtin/` |
| `middleware/` | 中断类中间件（审批、选择、文本输入、表单、信息确认） | `common/einox/middleware/` |
| `runtime/` | 独立运行时（Generate/Stream + RAG + Tool Registry） | `common/einox/runtime/` |
| `protocol/` | SSE 流式事件协议（Event/Adapter/Emitter） | `common/einox/protocol/` |
| `memory/` | 会话记忆存储（JSONL + GORM） | `common/einox/memory/` |
| `checkpoint/` | ADK 断点存储（JSONL + GORM） | `common/einox/checkpoint/` |
| `knowledge/` | 知识库 RAG（Milvus/Redis/GORM/Memory Store + Embedder） | `common/einox/knowledge/` |
| `metrics/` | 运行时指标收集 | `common/einox/metrics/` |
| `fsrestrict/` | Deep Agent 本地文件系统沙箱 | `common/einox/fsrestrict/` |

## Agent 工厂

### 工厂类型

所有工厂函数签名为 `func NewXxxAgent(ctx context.Context, ..., opts ...agent.Option) (*agent.Agent, error)`，统一返回 `agent.Agent` 包装。

| 工厂 | 用途 | source |
|------|------|--------|
| `agent.New` | 基础 ChatModel Agent（ReAct 工具调用） | `agent/agent.go:36` |
| `agent.NewChatModelAgent` | ChatModel + Option 便捷构造 | `agent/factory.go:24` |
| `agent.NewDeepAgent` | Deep Agent（预构建：WriteTodos + 可选 FileSystem + SubAgents） | `agent/factory.go:30` |
| `agent.NewPlanExecuteAgent` | 计划-执行 Agent | `agent/factory.go:91` |
| `agent.NewSupervisorAgent` | 多 Agent 协作（Supervisor 模式） | `agent/factory.go:150` |
| `agent.NewSequentialAgent` | 顺序工作流 Agent | `agent/factory.go:236` |
| `agent.NewParallelAgent` | 并行工作流 Agent | `agent/factory.go:272` |
| `agent.NewLoopAgent` | 循环工作流 Agent | `agent/factory.go:309` |

### Agent 结构

`agent/agent.go:28-33`：

```go
type Agent struct {
    name     string
    adkAgent adk.Agent
    runner   *adk.Runner
    opts     options
}
```

暴露的最小 API：
- `Runner() *adk.Runner` — 业务层执行 Run/Resume
- `ModelOptions() []model.Option` — 模型运行时选项（调用方通过 `adk.WithChatModelOptions` 传入）
- `Name() string` / `GetAgent() adk.Agent` / `Stop(ctx context.Context) error`

### Skill 中间件

`agent/agent.go:130-167` — 所有 Agent 工厂通过 `buildSkillHandlers` 自动加载文件系统 Skill：
- 目录来源：`WithSkillsDir(dir)` 或 `EINO_EXT_SKILLS_DIR` 环境变量
- 使用 `skill.NewBackendFromFilesystem` + `skill.NewMiddleware`
- 模型先只看元数据，命中后经验文件系统加载正文

### 选项体系

`agent/agent_option.go` — 遵循 Client Option 构造配置边界规范：

```go
type Option func(*options)
type options struct { ... } // 不直接改运行态 Agent

// 常用选项
WithName, WithDescription, WithInstruction
WithModel(model)                // 模型（BaseChatModel 或 ToolCallingChatModel）
WithTools(tools ...)            // 工具列表
WithSubAgents(agents ...)       // Workflow/Supervisor/Deep 子 Agent
WithHandler/WithHandlers        // ChatModelAgentMiddleware
WithMiddleware/WithMiddlewares  // AgentMiddleware
WithSkillsDir(dir)              // Skills 目录
WithMaxIterations(n)            // Deep/PlanExecute/Loop 最大迭代
WithCheckPointStore(store)      // ADK CheckPointStore
WithEnableFileSystem/WithDeepFilesystem  // Deep 文件系统沙箱
```

### Workflow 协调子 Agent

`agent/factory.go:215-230` — Workflow 类型（Sequential/Parallel/Loop）的 `workflowSubAgents`：
- 若配置了 tools/skills/handlers/middlewares 且提供 WithModel，自动前置协调子 Agent
- 协调子 Agent 的 name 自动加 `/coordinator` 后缀

## ChatModel 多厂商适配

`model/chatmodel.go` — 统一 Config + 厂商工厂：

```go
type Config struct {
    Provider       Provider // openai, deepseek, ollama, qwen, ark
    APIKey, BaseURL, Model string
    Temperature    float64
    TemperatureSet bool     // 支持 temperature=0
    MaxTokens      int
    // Ollama
    OllamaURL string
    // ARK
    ArkRegion string // cn-beijing (默认), cn-shanghai
}
```

- **ARK (火山引擎)**：支持 DeepSeek、豆包等模型，`model/chatmodel.go:169`
- **Claude**：暂不支持 eino-ext，需通过 OpenAI 兼容代理
- 返回统一 `model.BaseChatModel`，支持并发安全的 `WithTools` 工具绑定

## 工具管理（Kit + Policy）

### Kit

`tool/kit.go:41-43` — 按 Capability 分桶的工具注册表，并发安全：

| Capability | 说明 | builtin 实现 |
|-----------|------|-------------|
| `CapCompute` | 纯计算，无副作用 | `tool/builtin/compute.go` |
| `CapIO` | 外部副作用 | `tool/builtin/io.go` |
| `CapHuman` | 人机交互中断 | `tool/builtin/human.go` |

核心 API：
```go
k := tool.NewKit()
k.Register(cap, baseTool)       // 注册，重复名字返回 error
k.MustRegister(cap, baseTool)   // init/启动期用，失败 panic
k.ByCapability(cap)              // 按 capability 过滤
k.All()                          // 全部工具（按名升序）
k.Select(names...)               // 按名精确取
```

### Policy

`tool/policy.go` — 模式级工具白名单：

```go
p := tool.NewPolicy().
    AllowCapabilities(tool.CapCompute, tool.CapHuman).
    AllowNames("fetch_url")   // 额外放行个别 io 工具
tools := p.Apply(kit)          // 返回过滤后列表
```

空策略（无 AllowCapabilities 也无 AllowNames）默认放行全部。

### MCP Tool 适配

`tool/mcp.go` — 将 MCP 服务器的工具列表转换为 eino `tool.BaseTool`，统一工具入口。

## 中间件（中断）

`middleware/types.go` — 5 种中断交互，schema.Register 自动注册序列化：

| 中断类型 | Info 结构 | Result 结构 | 说明 |
|---------|----------|------------|------|
| Approval | `ApprovalInfo` | `ApprovalResult` | 二选一审批（approve/deny） |
| Single/Multi Select | `SelectInfo` | `SelectResult` | 单选/多选 |
| Free Text | `TextInputInfo` | `TextInputResult` | 自由文本输入 |
| Form Input | `FormInputInfo` | `FormInputResult` | 结构化表单 |
| Info Ack | `InfoAckInfo` | `InfoAckResult` | 展示信息 + 确认 |

所有 info/result 结构体需在 init() 中调用 `schema.Register[*T]()`。

## 运行时（Runtime）

`runtime/runner.go` — 独立于 ADK 的轻量运行时，适合不需要复杂 Agent 编排的场景：

```go
runner, _ := runtime.NewRunner(chatModel,
    runtime.WithTools(toolRegistry),
    runtime.WithRetriever(retriever, topK),
    runtime.WithMaxToolIterations(8),
)

events, err := runner.Generate(ctx, runtime.Request{
    System:  "You are a helpful assistant",
    Input:   "Hello",
    History: []*schema.Message{...},
    RAG:     runtime.RAGRequest{Retriever: myRetriever, TopK: 5},
})
```

- `Generate(ctx, req)` — 非流式生成（含工具调用循环 + RAG 检索）
- `Stream(ctx, req)` — 流式生成（SSE 事件流）
- 工具调用循环最大迭代 8 次（默认），可通过 `WithMaxToolIterations` 调整

## 协议事件（Protocol）

`protocol/event.go` — 统一 SSE 流式事件协议：

事件序列：`turn.start → message.start → message.delta* → message.end → tool.call.start/end → interrupt → turn.end`

事件类型枚举：`EventTurnStart/End`、`EventMessageStart/Delta/End`、`EventToolCallStart/End`、`EventInterrupt`、`EventError`

`protocol/adapter.go` — ADK AgentEvent 到 protocol.Event 的映射。

## 存储（Memory + Checkpoint + Knowledge）

| 存储层 | 后端 | source |
|--------|------|--------|
| Memory | JSONL、GORM | `memory/jsonl_storage.go`、`memory/gormx_storage.go` |
| Checkpoint | JSONL、GORM | `checkpoint/jsonl_store.go`、`checkpoint/gormx_store.go` |
| Knowledge | Milvus、Redis、GORM、Memory | `knowledge/store_milvus.go`、`knowledge/store_redis.go`、`knowledge/store_gorm.go`、`knowledge/store_memory.go` |

统一接口：
- `memory.Storage` — `Put/Get/Delete/List`
- `checkpoint.Store` — 实现 `adk.CheckPointStore`
- `knowledge.Store` — `Store/Search/Delete`；`knowledge.Embedder` — `Embed(ctx, text) ([]float64, error)`

## 反模式

- 不要在 Agent 构造选项里直接修改 `adk.Runner` 或 `adk.Agent` 内部字段。
- 不要绕过 `agent.New*` 工厂直接创建 `adk.ChatModelAgent`。
- 不要在工作流 Agent 中重复实现工具/技能路由——用 `WithTools` + `buildSkillHandlers` 统一注入。
- Middleware info/result 结构体必须 `schema.Register`，否则序列化会 panic。
- 模型温度设为 0 时必须同时设 `TemperatureSet: true`。
