package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// MCPCaller 抽象 MCP 工具调用能力，方便测试和替换。
type MCPCaller interface {
	CallTool(ctx context.Context, name string, args map[string]any) (string, error)
}

// MCPTool 将 MCP 客户端的远程工具包装为 Eino InvokableTool。
type MCPTool struct {
	caller MCPCaller
	name   string
	desc   string
}

// NewMCPTool 创建一个 MCP 工具包装器。
func NewMCPTool(caller MCPCaller, name, desc string) *MCPTool {
	return &MCPTool{caller: caller, name: name, desc: desc}
}

// Info 返回工具元信息。
func (t *MCPTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.name,
		Desc: t.desc,
	}, nil
}

// InvokableRun 调用远程 MCP 工具。
func (t *MCPTool) InvokableRun(ctx context.Context, args string, _ ...tool.Option) (string, error) {
	var parsed map[string]any
	if args != "" {
		if err := json.Unmarshal([]byte(args), &parsed); err != nil {
			return "", fmt.Errorf("mcp tool %q: parse args: %w", t.name, err)
		}
	}
	result, err := t.caller.CallTool(ctx, t.name, parsed)
	if err != nil {
		return "", fmt.Errorf("mcp tool %q: %w", t.name, err)
	}
	return result, nil
}

var _ tool.InvokableTool = (*MCPTool)(nil)
