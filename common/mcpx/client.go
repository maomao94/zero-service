package mcpx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
	"zero-service/common/ctxprop"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/core/timex"
)

// Client 管理多个 MCP server 连接，聚合工具，路由调用。
type Client struct {
	conns      []*serverConn
	mu         sync.RWMutex
	tools      []*mcp.Tool
	toolRoutes map[string]*serverConn
	metrics    *stat.Metrics
	ctx        context.Context
	cancel     context.CancelFunc
}

// serverConn 单个 MCP server 的连接。
type serverConn struct {
	name          string
	endpoint      string
	serviceToken  string
	useStreamable bool
	client        *mcp.Client
	session       *mcp.ClientSession
	tools         []*mcp.Tool
	mu            sync.RWMutex
	cfg           Config
	onChange      func()
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewClient 创建 mcpx 客户端，非阻塞，后台自动连接。
func NewClient(cfg Config) *Client {
	if cfg.RefreshInterval <= 0 {
		cfg.RefreshInterval = DefaultRefreshInterval
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = DefaultConnectTimeout
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		toolRoutes: make(map[string]*serverConn),
		metrics:    stat.NewMetrics("mcpx"),
		ctx:        ctx,
		cancel:     cancel,
	}

	logger := newLogxLogger()
	names := make(map[string]bool)

	for i, sc := range cfg.Servers {
		name := sc.Name
		if name == "" {
			name = fmt.Sprintf("mcp%d", i)
		}
		if names[name] {
			logx.Errorf("[mcpx] duplicate server name %q, skipping", name)
			continue
		}
		names[name] = true

		connCtx, connCancel := context.WithCancel(ctx)
		conn := &serverConn{
			name:          name,
			endpoint:      sc.Endpoint,
			serviceToken:  sc.ServiceToken,
			useStreamable: sc.UseStreamable,
			cfg:           cfg,
			onChange:      c.rebuildTools,
			ctx:           connCtx,
			cancel:        connCancel,
		}
		conn.client = mcp.NewClient(&mcp.Implementation{
			Name:    "mcpx-" + name,
			Version: "1.0.0",
		}, &mcp.ClientOptions{
			Logger:    logger,
			KeepAlive: cfg.RefreshInterval,
			ToolListChangedHandler: func(ctx context.Context, _ *mcp.ToolListChangedRequest) {
				if err := conn.refreshTools(); err != nil {
					logx.Errorf("[mcpx] %s refresh tools: %v", conn.name, err)
					return
				}
				conn.onChange()
			},
		})

		c.conns = append(c.conns, conn)
		go conn.run()
	}

	logx.Infof("[mcpx] started, servers=%d", len(c.conns))
	return c
}

// Tools 返回聚合工具列表（带 serverName__ 前缀）。
func (c *Client) Tools() []*mcp.Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tools
}

// HasTools 是否有可用工具。
func (c *Client) HasTools() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.tools) > 0
}

// CallTool 路由到对应 server 调用工具。
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (string, error) {
	start := timex.Now()

	c.mu.RLock()
	conn, ok := c.toolRoutes[name]
	c.mu.RUnlock()
	if !ok {
		c.metrics.AddDrop()
		return "", fmt.Errorf("[mcpx] tool %q not found", name)
	}

	realName := name
	if idx := strings.Index(name, ToolNameSeparator); idx >= 0 {
		realName = name[idx+len(ToolNameSeparator):]
	}

	result, err := conn.callTool(ctx, realName, args)
	if err != nil {
		c.metrics.AddDrop()
		return "", err
	}

	c.metrics.Add(stat.Task{Duration: timex.Since(start)})
	return result, nil
}

// Close 关闭所有连接。
func (c *Client) Close() {
	c.cancel()
	for _, conn := range c.conns {
		conn.close()
	}
}

// rebuildTools 聚合所有 server 的工具。
func (c *Client) rebuildTools() {
	var all []*mcp.Tool
	routes := make(map[string]*serverConn)

	for _, conn := range c.conns {
		for _, t := range conn.getTools() {
			prefixed := conn.name + ToolNameSeparator + t.Name
			all = append(all, &mcp.Tool{
				Name:        prefixed,
				Description: t.Description,
				InputSchema: t.InputSchema,
				Annotations: t.Annotations,
			})
			routes[prefixed] = conn
		}
	}

	c.mu.Lock()
	c.tools = all
	c.toolRoutes = routes
	c.mu.Unlock()
}

// --- serverConn ---

func (sc *serverConn) run() {
	for {
		session := sc.tryConnect()
		if session != nil {
			_ = session.Wait()
			logx.Errorf("[mcpx] %s disconnected", sc.name)
			sc.mu.Lock()
			sc.session = nil
			sc.tools = nil
			sc.mu.Unlock()
			sc.onChange()
		}
		t := time.NewTimer(sc.cfg.RefreshInterval)
		select {
		case <-sc.ctx.Done():
			t.Stop()
			return
		case <-t.C:
		}
	}
}

func (sc *serverConn) tryConnect() *mcp.ClientSession {
	httpClient := &http.Client{Transport: &ctxHeaderTransport{base: http.DefaultTransport, serviceToken: sc.serviceToken}}

	var transport mcp.Transport
	if sc.useStreamable {
		transport = &mcp.StreamableClientTransport{
			Endpoint:   sc.endpoint,
			HTTPClient: httpClient,
		}
	} else {
		transport = &mcp.SSEClientTransport{
			Endpoint:   sc.endpoint,
			HTTPClient: httpClient,
		}
	}

	session, err := sc.client.Connect(sc.ctx, transport, nil)
	if err != nil {
		logx.Errorf("[mcpx] %s connect: %v", sc.name, err)
		return nil
	}

	toolCtx, toolCancel := context.WithTimeout(sc.ctx, sc.cfg.ConnectTimeout)
	defer toolCancel()

	var tools []*mcp.Tool
	for t, err := range session.Tools(toolCtx, nil) {
		if err != nil {
			logx.Errorf("[mcpx] %s load tools: %v", sc.name, err)
			_ = session.Close()
			return nil
		}
		tools = append(tools, t)
	}

	sc.mu.Lock()
	sc.session = session
	sc.tools = tools
	sc.mu.Unlock()
	sc.onChange()

	logx.Infof("[mcpx] %s connected, endpoint=%s, tools=%d", sc.name, sc.endpoint, len(tools))
	return session
}

func (sc *serverConn) refreshTools() error {
	sc.mu.RLock()
	session := sc.session
	sc.mu.RUnlock()
	if session == nil {
		return nil
	}

	var tools []*mcp.Tool
	for t, err := range session.Tools(sc.ctx, nil) {
		if err != nil {
			return err
		}
		tools = append(tools, t)
	}

	sc.mu.Lock()
	sc.tools = tools
	sc.mu.Unlock()
	return nil
}

func (sc *serverConn) getTools() []*mcp.Tool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.tools
}

func (sc *serverConn) callTool(ctx context.Context, name string, args map[string]any) (string, error) {
	sc.mu.RLock()
	session := sc.session
	sc.mu.RUnlock()
	if session == nil {
		return "", fmt.Errorf("[mcpx] %s not connected", sc.name)
	}

	params := &mcp.CallToolParams{Name: name, Arguments: args}

	// Inject user context into _meta for per-message auth (SSE transport).
	if meta := ctxprop.CollectFromCtx(ctx); len(meta) > 0 {
		params.SetMeta(meta)
	}

	result, err := session.CallTool(ctx, params)
	if err != nil {
		return "", fmt.Errorf("[mcpx] call %q on %s: %w", name, sc.name, err)
	}

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

func (sc *serverConn) close() {
	sc.cancel()
	sc.mu.RLock()
	s := sc.session
	sc.mu.RUnlock()
	if s != nil {
		_ = s.Close()
	}
}

// ParseArgs 解析 LLM 返回的 JSON 字符串参数为 map。
func ParseArgs(argsJSON string) map[string]any {
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return map[string]any{}
	}
	return args
}

// ctxHeaderTransport 是自定义 http.RoundTripper，
// 从请求 context 提取用户上下文，注入为 HTTP 头。
type ctxHeaderTransport struct {
	base         http.RoundTripper
	serviceToken string
}

func (t *ctxHeaderTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	ctxprop.InjectToHTTPHeader(r.Context(), r.Header)
	authType := "user"
	// Authorization 降级：ctx 中无用户 JWT 时使用 ServiceToken
	if r.Header.Get("Authorization") == "" && t.serviceToken != "" {
		r.Header.Set("Authorization", "Bearer "+t.serviceToken)
		authType = "system"
	}
	logx.Debugf("[mcpx] transport: authType=%s, method=%s, path=%s, token=%s", authType, r.Method, r.URL.Path, r.Header.Get("Authorization"))
	return t.base.RoundTrip(r)
}
