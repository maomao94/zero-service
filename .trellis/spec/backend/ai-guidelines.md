# AI 会话、工具与 MCP 规范

## 适用范围

修改 `common/einox`、`common/mcpx`、`aiapp/aichat`、`aiapp/aisolo`、`aiapp/mcpserver` 或 AI 会话/工具执行时读取。

## 工具能力与运行时

- 工具由 `common/einox/tool` registry 按 capability 注册，运行时按 policy 和当前会话能力过滤；不要把所有已注册工具无条件暴露给模型。
- 工具 schema、名称和参数是模型可见契约；变更前检查 prompt、agent、MCP route 和测试，不只修改 Go handler。
- Lite/runtime runner 通过协议事件输出模型、工具调用和结果；新增事件保持顺序和结束语义，不把内部 callback 直接暴露。
- 工具迭代必须有上限并响应 context，避免模型/工具循环无限运行。工具 panic/error 转成明确事件或错误，不吞掉。

### einox 子包速览

| 子包 | 职责 |
|------|------|
| `agent/` | Agent 工厂、配置选项（基于 eino） |
| `runtime/` | Runner 执行器，协调模型调用、工具执行、RAG 检索 |
| `tool/` | 工具注册（capability/policy/mcp adapter）、内置工具（compute/io/human） |
| `model/` | ChatModel 封装与选项 |
| `memory/` | 会话记忆存储（GORM/JSONL/内存） |
| `knowledge/` | 知识库服务（GORM/Redis/Milvus 存储、embedding、分块） |
| `checkpoint/` | 执行检查点持久化（GORM/JSONL），支持 resume |
| `protocol/` | 协议事件定义与 codec |
| `middleware/` | 中间件（如审批流 approval） |
| `metrics/` | 执行指标采集 |
| `fsrestrict/` | 文件系统访问控制，限制工具只能访问会话 workspace |

mcpx 额外能力：
- `async_result.go` — 异步工具结果存储接口与实现
- `auth.go` — JWT/ServiceToken 双认证验证器
- `config.go` — 多 Server 配置、SSE/Streamable transport 切换
- `wrapper.go` — HTTP handler 包装（CORS、超时）

依据：`common/einox/runtime/runner.go`、`common/einox/tool/kit.go`、`common/einox/tool/policy.go`、`common/mcpx/client.go`、`common/mcpx/server.go`、`common/mcpx/auth.go`、`common/einox` 测试。

## 会话执行所有权

- aisolo 会话至少区分 `IDLE`、`RUNNING`、`INTERRUPTED`；开始执行前通过持久化/CAS 取得运行所有权，不能仅检查后内存赋值。
- 保存消息或启动模型失败时恢复可重试状态；成功/中断/失败路径都必须释放当前执行所有权。
- Resume 由 interrupt ID/持久化 checkpoint 定位，不能只依赖进程内指针；旧执行回调不得覆盖已恢复的新执行。
- 会话 workspace 限制在分配目录内，所有文件路径规范化并检查越界；工具不能访问任意宿主路径。

依据：`aiapp/aisolo/internal/turn/executor.go`、`aiapp/aisolo/internal/turn` 测试、会话 store 实现。

## MCP 边界

- `common/mcpx` client 管理 connection、tool route 和异步响应；业务服务通过已有 client/registry 使用工具，不复制连接管理。
- server 的认证 wrapper 同时覆盖 SSE/Streamable 等已启用 transport；新增 endpoint 不能绕过统一认证和 context 注入。
- MCP transport error、tool protocol error 和工具业务 error 分层返回，网关再映射 OpenAI-compatible 或 HTTP 响应。
- 连接断开/重连、请求超时和 server shutdown 必须完成 pending 请求并释放 goroutine。

依据：`common/mcpx/client.go`、`common/mcpx/server.go`、`aiapp/mcpserver`。

## 反模式

- 把 registry 中所有工具直接交给模型，绕过 capability/policy。
- 用进程内 bool 作为分布式/并发会话所有权。
- 执行失败后会话永久停留 `RUNNING`。
- 将用户提供的相对路径直接拼接到宿主目录。
- 只给一种 MCP transport 加认证，其他入口裸露。

## 验证

- 工具测试覆盖 capability 过滤、schema、超时、panic、最大迭代和事件顺序。
- 会话测试覆盖并发 acquire、保存失败、模型失败、中断、resume、迟到回调和 workspace 越界。
- MCP 测试覆盖各 transport 的认证、断连、超时、未知 tool/route、关闭 pending 和错误映射。
