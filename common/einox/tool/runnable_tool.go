package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// RunnableToolConfig Runnable 转换为 Tool 的配置
type RunnableToolConfig struct {
	Name  string // 工具名称
	Desc  string // 工具描述
	Param string // 参数名称（默认 "input"）
}

// NewRunnableTool 将 Runnable（Graph/Chain/Workflow）转换为 Agent 可调用的 Tool
//
// 支持的类型：
//   - compose.Runnable[I, O]
//   - compose.Graph
//   - compose.Chain
//   - compose.Workflow
//
// 示例：
//
//	// 创建 Graph
//	g := compose.NewGraph[string, string]()
//	g.AddLambdaNode("process", compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
//	    return strings.ToUpper(input), nil
//	}))
//	g.AddEdge(compose.START, "process")
//	g.AddEdge("process", compose.END)
//
//	compiled, _ := g.Compile(ctx)
//
//	// 转换为 Tool
//	runnableTool := tool.NewRunnableTool(ctx, compiled, &tool.RunnableToolConfig{
//	    Name:  "text_processor",
//	    Desc:  "处理文本，转为大写",
//	    Param: "text",
//	})
//
//	// 在 Agent 中使用
//	agent, _ := agent.New(ctx,
//	    agent.WithTools(runnableTool),
//	)
func NewRunnableTool[I, O any](ctx context.Context, runnable compose.Runnable[I, O], cfg *RunnableToolConfig) (tool.BaseTool, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if cfg.Name == "" {
		return nil, fmt.Errorf("tool name is required")
	}

	paramName := cfg.Param
	if paramName == "" {
		paramName = "input"
	}

	return &runnableTool[I, O]{
		name:      cfg.Name,
		desc:      cfg.Desc,
		paramName: paramName,
		runnable:  runnable,
	}, nil
}

// runnableTool 将 Runnable 包装为 Tool
type runnableTool[I, O any] struct {
	name      string
	desc      string
	paramName string
	runnable  compose.Runnable[I, O]
}

func (t *runnableTool[I, O]) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.name,
		Desc: t.desc,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			t.paramName: {
				Type:     getInputType[I](),
				Desc:     "输入参数",
				Required: true,
			},
		}),
	}, nil
}

func (t *runnableTool[I, O]) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 解析输入参数
	var input I
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		// 如果解析失败，尝试从 input 字段获取
		var wrapper struct {
			Input I `json:"input"`
		}
		if unmarshalErr := json.Unmarshal([]byte(argumentsInJSON), &wrapper); unmarshalErr != nil {
			return "", fmt.Errorf("parse input: %w", err)
		}
		input = wrapper.Input
	}

	// 调用 Runnable
	output, err := t.runnable.Invoke(ctx, input)
	if err != nil {
		return "", fmt.Errorf("invoke runnable: %w", err)
	}

	// 序列化为 JSON
	result, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("marshal output: %w", err)
	}

	return string(result), nil
}

// getInputType 根据类型获取 schema.DataType
func getInputType[T any]() schema.DataType {
	switch any(*new(T)).(type) {
	case string:
		return schema.String
	case int, int64:
		return schema.Integer
	case float64:
		return schema.Number
	case bool:
		return schema.Boolean
	case []string:
		return schema.Array
	case map[string]any:
		return schema.Object
	default:
		return schema.String // 默认使用 string
	}
}

// Ensure runnableTool implements tool.InvokableTool
var _ tool.InvokableTool = (*runnableTool[string, string])(nil)

// StringRunnableTool 便捷构造函数，用于 string -> string 的简单 Runnable
func StringRunnableTool(ctx context.Context, runnable compose.Runnable[string, string], name, desc string) (tool.BaseTool, error) {
	return NewRunnableTool(ctx, runnable, &RunnableToolConfig{
		Name: name,
		Desc: desc,
	})
}

// MapRunnableTool 用于 map[string]any -> map[string]any 的 Runnable
func MapRunnableTool(ctx context.Context, runnable compose.Runnable[map[string]any, map[string]any], name, desc string) (tool.BaseTool, error) {
	return NewRunnableTool(ctx, runnable, &RunnableToolConfig{
		Name: name,
		Desc: desc,
	})
}
