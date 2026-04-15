package tool

import (
	"context"
	"testing"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRunnableTool(t *testing.T) {
	ctx := context.Background()

	// 创建简单的 Lambda
	lambda := compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		return "processed: " + input, nil
	})

	// 创建 Graph 并添加 Lambda 节点
	g := compose.NewGraph[string, string]()
	err := g.AddLambdaNode("process", lambda)
	require.NoError(t, err)

	err = g.AddEdge(compose.START, "process")
	require.NoError(t, err)
	err = g.AddEdge("process", compose.END)
	require.NoError(t, err)

	// 编译获取 Runnable
	runnable, err := g.Compile(ctx)
	require.NoError(t, err)

	// 创建 Tool
	tool, err := NewRunnableTool(ctx, runnable, &RunnableToolConfig{
		Name: "text_processor",
		Desc: "处理文本",
	})
	require.NoError(t, err)
	require.NotNil(t, tool)

	// 测试 Info
	info, err := tool.Info(ctx)
	require.NoError(t, err)
	assert.Equal(t, "text_processor", info.Name)
	assert.Equal(t, "处理文本", info.Desc)
}

func TestStringRunnableTool(t *testing.T) {
	ctx := context.Background()

	// 创建简单的 Lambda
	lambda := compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		return "echo: " + input, nil
	})

	// 创建 Graph
	g := compose.NewGraph[string, string]()
	err := g.AddLambdaNode("echo", lambda)
	require.NoError(t, err)
	err = g.AddEdge(compose.START, "echo")
	require.NoError(t, err)
	err = g.AddEdge("echo", compose.END)
	require.NoError(t, err)

	// 编译获取 Runnable
	runnable, err := g.Compile(ctx)
	require.NoError(t, err)

	// 创建 Tool
	toolInstance, err := StringRunnableTool(ctx, runnable, "echo", "回显输入")
	require.NoError(t, err)

	// 类型断言为 InvokableTool
	invokableTool, ok := toolInstance.(einotool.InvokableTool)
	require.True(t, ok)

	// 测试调用
	result, err := invokableTool.InvokableRun(ctx, `"test input"`)
	require.NoError(t, err)
	assert.Contains(t, result, "test input")
}

func TestRunnableTool_MissingConfig(t *testing.T) {
	ctx := context.Background()

	// 创建简单的 Graph
	g := compose.NewGraph[string, string]()
	lambda := compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		return input, nil
	})
	err := g.AddLambdaNode("process", lambda)
	require.NoError(t, err)
	_ = g.AddEdge(compose.START, "process")
	_ = g.AddEdge("process", compose.END)

	runnable, err := g.Compile(ctx)
	require.NoError(t, err)

	// 测试空配置
	_, err = NewRunnableTool(ctx, runnable, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")

	// 测试空名称
	_, err = NewRunnableTool(ctx, runnable, &RunnableToolConfig{
		Name: "",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tool name is required")
}
