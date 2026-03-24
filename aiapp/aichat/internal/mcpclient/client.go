package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"zero-service/aiapp/aichat/internal/provider"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zeromicro/go-zero/core/logx"
)

// McpClient 封装 MCP SDK 客户端连接和工具操作
type McpClient struct {
	client  *mcp.Client
	session *mcp.ClientSession
	tools   []*mcp.Tool
	mu      sync.RWMutex
}

// NewMcpClient 通过 SSE 连接到 MCP Server，初始化后缓存工具列表
func NewMcpClient(ctx context.Context, endpoint string) (*McpClient, error) {
	mc := &McpClient{}

	mc.client = mcp.NewClient(&mcp.Implementation{
		Name:    "aichat-mcp-client",
		Version: "1.0.0",
	}, &mcp.ClientOptions{
		ToolListChangedHandler: func(ctx context.Context, req *mcp.ToolListChangedRequest) {
			if err := mc.refreshTools(ctx); err != nil {
				logx.Errorf("refresh mcp tools failed: %v", err)
			}
		},
	})

	transport := &mcp.SSEClientTransport{Endpoint: endpoint}
	session, err := mc.client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect mcp server: %w", err)
	}
	mc.session = session

	// 初始化工具列表
	if err := mc.refreshTools(ctx); err != nil {
		session.Close()
		return nil, fmt.Errorf("list mcp tools: %w", err)
	}

	logx.Infof("mcp client connected to %s, tools: %d", endpoint, len(mc.tools))
	return mc, nil
}

// refreshTools 刷新工具列表缓存
func (c *McpClient) refreshTools(ctx context.Context) error {
	var tools []*mcp.Tool
	for tool, err := range c.session.Tools(ctx, nil) {
		if err != nil {
			return err
		}
		tools = append(tools, tool)
	}

	c.mu.Lock()
	c.tools = tools
	c.mu.Unlock()

	logx.Infof("mcp tools refreshed, count: %d", len(tools))
	return nil
}

// ToOpenAITools 将 MCP 工具转换为 OpenAI function calling 格式
func (c *McpClient) ToOpenAITools() []provider.ToolDef {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.tools) == 0 {
		return nil
	}

	defs := make([]provider.ToolDef, len(c.tools))
	for i, t := range c.tools {
		defs[i] = provider.ToolDef{
			Type: "function",
			Function: provider.ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		}
	}
	return defs
}

// CallTool 调用 MCP 工具，返回文本结果
func (c *McpClient) CallTool(ctx context.Context, name string, args map[string]any) (string, error) {
	result, err := c.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return "", fmt.Errorf("call tool %q: %w", name, err)
	}

	// 将 Content 拼接为文本
	var sb strings.Builder
	for _, c := range result.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(tc.Text)
		}
	}
	return sb.String(), nil
}

// Close 关闭 MCP 连接
func (c *McpClient) Close() error {
	if c.session != nil {
		return c.session.Close()
	}
	return nil
}

// ParseArgs 解析 LLM 返回的 JSON 字符串参数为 map
func ParseArgs(argsJSON string) map[string]any {
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return map[string]any{}
	}
	return args
}
