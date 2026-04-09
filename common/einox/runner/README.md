# Solo Runner - Eino Agent 封装

SoloRunner 封装了 Eino Agent 的调用，提供了简单易用的接口来执行 Agent 查询、流式输出和聊天对话。

## 主要特性

- **简洁的 API**：封装了底层的 adk.Runner，提供简单易用的方法
- **流式支持**：支持流式查询和流式输出到 A2UI
- **历史记录**：可选支持对话历史记录和自动保存
- **灵活配置**：通过选项模式配置各种功能
- **Go-zero 风格**：集成了 logx.Logger，方便日志记录

## 使用示例

### 基本使用

```go
import (
    "context"
    "github.com/cloudwego/eino/adk"
    "zero-service/common/einox/runner"
)

ctx := context.Background()

agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:        "MyAgent",
    Description: "My AI Assistant",
    Instruction: "You are a helpful assistant.",
    Model:       myChatModel,
})
if err != nil {
    return err
}

sr, err := runner.NewSoloRunner(ctx, agent)
if err != nil {
    return err
}
```

### 配置选项

```go
sr, err := runner.NewSoloRunner(ctx, agent,
    runner.WithEnableStreaming(true),
    runner.WithEnableHistory(true),
    runner.WithMaxHistory(20),
)
```

### 执行查询

```go
iter, err := sr.Query(ctx, "Hello, how are you?")
if err != nil {
    return err
}

for {
    event, ok := iter.Next()
    if !ok {
        break
    }
    if event.Output != nil {
        msg, _ := event.Output.MessageOutput.GetMessage()
        fmt.Println(msg.Content)
    }
}
```

### 流式查询

```go
messages := []*schema.Message{
    schema.UserMessage("Hello!"),
}

iter, err := sr.QueryStream(ctx, "session-123", messages)
```

### 聊天对话（支持历史）

```go
result, err := sr.Chat(ctx, "user-1", "session-1", "Hello, how are you?")
if err != nil {
    return err
}
fmt.Println(result.Response)
```

### 流式输出到 A2UI

```go
messages := []*schema.Message{
    schema.UserMessage("Hello!"),
}

var buf bytes.Buffer
response, err := sr.StreamToA2UI(ctx, &buf, "session-123", messages)
```

## API 文档

### NewSoloRunner

创建新的 SoloRunner 实例。

```go
func NewSoloRunner(ctx context.Context, agent adk.Agent, opts ...RunnerOption) (*SoloRunner, error)
```

### Query

执行单轮查询，返回事件迭代器。

```go
func (r *SoloRunner) Query(ctx context.Context, input string) (*adk.AsyncIterator[*adk.AgentEvent], error)
```

### QueryStream

流式查询，支持会话历史。

```go
func (r *SoloRunner) QueryStream(ctx context.Context, sessionID string, messages []*schema.Message) (*adk.AsyncIterator[*adk.AgentEvent], error)
```

### Chat

聊天对话，自动处理历史记录。

```go
func (r *SoloRunner) Chat(ctx context.Context, userID, sessionID, input string) (*ChatResult, error)
```

### StreamToA2UI

流式输出到 A2UI 格式。

```go
func (r *SoloRunner) StreamToA2UI(ctx context.Context, w io.Writer, sessionID string, messages []*schema.Message) (string, error)
```

## 配置选项

- `WithEnableStreaming(bool)`: 启用流式输出
- `WithEnableHistory(bool)`: 启用历史记录
- `WithMaxHistory(int)`: 最大历史记录数

## 相关文档

- [Eino ADK 文档](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/)
- [A2UI 组件](../a2ui/README.md)
- [Memory 组件](../memory/README.md)
