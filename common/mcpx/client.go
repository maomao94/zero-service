package mcpx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"zero-service/common/ctxdata"
	"zero-service/common/ctxprop"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/core/timex"
)

// Client MCP 客户端，专注于连接管理和 MCP 协议完整支持
// 支持 Tools、Prompts、Resources、Logging、Progress、Sampling、Elicitation 等所有 MCP 核心功能
type Client struct {
	config Config

	connections map[string]*Connection
	mu          sync.RWMutex

	tools      []*mcp.Tool
	toolRoutes map[string]*Connection

	prompts      []*mcp.Prompt
	promptRoutes map[string]*Connection

	resources      []*mcp.Resource
	resourceRoutes map[string]*Connection

	ctx    context.Context
	cancel context.CancelFunc

	metrics *stat.Metrics

	options *mcp.ClientOptions
}

// Connection 单个 MCP 服务连接
type Connection struct {
	name         string
	endpoint     string
	serviceToken string
	client       *mcp.Client
	session      *mcp.ClientSession
	transport    mcp.Transport
	tools        []*mcp.Tool
	prompts      []*mcp.Prompt
	resources    []*mcp.Resource
	mu           sync.RWMutex

	cfg      Config
	onChange func()

	ctx    context.Context
	cancel context.CancelFunc
}

// NewClient 创建 MCP 客户端
// opts 参数可选，为 nil 时使用默认配置
// 支持配置所有 MCP handlers（CreateMessageHandler、ElicitationHandler 等）
func NewClient(cfg Config, opts ...*mcp.ClientOptions) *Client {
	var clientOpts *mcp.ClientOptions
	if len(opts) > 0 {
		clientOpts = opts[0]
	}

	if cfg.RefreshInterval <= 0 {
		cfg.RefreshInterval = DefaultRefreshInterval
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = DefaultConnectTimeout
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		config:         cfg,
		connections:    make(map[string]*Connection),
		toolRoutes:     make(map[string]*Connection),
		promptRoutes:   make(map[string]*Connection),
		resourceRoutes: make(map[string]*Connection),
		metrics:        stat.NewMetrics("mcpx"),
		ctx:            ctx,
		cancel:         cancel,
		options:        clientOpts,
	}

	names := make(map[string]bool)
	for i, sc := range cfg.Servers {
		name := sc.Name
		if name == "" {
			name = fmt.Sprintf("mcp%d", i)
		}
		if names[name] {
			logx.WithContext(c.ctx).Errorf("[mcpx] duplicate server name %q, skipping", name)
			continue
		}
		names[name] = true

		connCtx, connCancel := context.WithCancel(ctx)
		conn := &Connection{
			name:         name,
			endpoint:     sc.Endpoint,
			serviceToken: sc.ServiceToken,
			cfg:          cfg,
			onChange:     c.rebuildAll,
			ctx:          connCtx,
			cancel:       connCancel,
		}

		c.connections[name] = conn
		go conn.run(c.buildClientOptions())
	}

	logx.WithContext(c.ctx).Infof("[mcpx] started, servers=%d", len(c.connections))
	return c
}

// buildClientOptions 构建客户端选项，合并用户配置和默认值
func (c *Client) buildClientOptions() *mcp.ClientOptions {
	opts := &mcp.ClientOptions{
		Logger:    newLogxLogger(),
		KeepAlive: c.config.RefreshInterval,
	}

	if c.options != nil {
		if c.options.Logger != nil {
			opts.Logger = c.options.Logger
		}

		// CreateMessageHandler: 处理采样请求（sampling/createMessage）
		// 用于客户端作为采样服务器，响应来自服务器的采样请求
		if c.options.CreateMessageHandler != nil {
			opts.CreateMessageHandler = c.options.CreateMessageHandler
		}

		// ElicitationHandler: 处理诱导请求（elicitation/create）
		// 用于客户端作为诱导服务器，响应来自服务器的诱导请求
		if c.options.ElicitationHandler != nil {
			opts.ElicitationHandler = c.options.ElicitationHandler
		}

		// Capabilities: 客户端能力配置
		// 用于声明客户端支持的能力（如 roots、sampling、elicitation 等）
		if c.options.Capabilities != nil {
			opts.Capabilities = c.options.Capabilities
		}

		// ElicitationCompleteHandler: 处理诱导完成通知（notifications/elicitation/complete）
		// 用于接收诱导操作完成的通知
		if c.options.ElicitationCompleteHandler != nil {
			opts.ElicitationCompleteHandler = c.options.ElicitationCompleteHandler
		}

		// ToolListChangedHandler: 处理工具列表变更通知（notifications/tools/list_changed）
		// 当服务器工具列表发生变化时触发，用于刷新本地工具缓存
		if c.options.ToolListChangedHandler != nil {
			opts.ToolListChangedHandler = c.options.ToolListChangedHandler
		}

		// PromptListChangedHandler: 处理提示模板列表变更通知（notifications/prompts/list_changed）
		// 当服务器提示模板列表发生变化时触发，用于刷新本地提示模板缓存
		if c.options.PromptListChangedHandler != nil {
			opts.PromptListChangedHandler = c.options.PromptListChangedHandler
		}

		// ResourceListChangedHandler: 处理资源列表变更通知（notifications/resources/list_changed）
		// 当服务器资源列表发生变化时触发，用于刷新本地资源缓存
		if c.options.ResourceListChangedHandler != nil {
			opts.ResourceListChangedHandler = c.options.ResourceListChangedHandler
		}

		// ResourceUpdatedHandler: 处理资源更新通知（notifications/resources/updated）
		// 当特定资源内容发生变化时触发，用于刷新该资源的内容
		if c.options.ResourceUpdatedHandler != nil {
			opts.ResourceUpdatedHandler = c.options.ResourceUpdatedHandler
		}

		// LoggingMessageHandler: 处理日志消息通知（notifications/message）
		// 用于接收来自服务器的日志消息
		if c.options.LoggingMessageHandler != nil {
			opts.LoggingMessageHandler = c.options.LoggingMessageHandler
		}

		// ProgressNotificationHandler: 处理进度通知（notifications/progress）
		// 用于接收长时间运行操作的进度更新
		if c.options.ProgressNotificationHandler != nil {
			opts.ProgressNotificationHandler = c.options.ProgressNotificationHandler
		}
	}

	// 默认处理工具列表变更：刷新所有连接的工具列表
	if opts.ToolListChangedHandler == nil {
		opts.ToolListChangedHandler = func(ctx context.Context, req *mcp.ToolListChangedRequest) {
			c.refreshTools()
		}
	}

	// 默认处理提示模板列表变更：刷新所有连接的提示模板列表
	if opts.PromptListChangedHandler == nil {
		opts.PromptListChangedHandler = func(ctx context.Context, req *mcp.PromptListChangedRequest) {
			c.refreshPrompts()
		}
	}

	// 默认处理资源列表变更：刷新所有连接的资源列表
	if opts.ResourceListChangedHandler == nil {
		opts.ResourceListChangedHandler = func(ctx context.Context, req *mcp.ResourceListChangedRequest) {
			c.refreshResources()
		}
	}

	return opts
}

// Tools 获取所有工具（带 serverName__ 前缀）
func (c *Client) Tools() []*mcp.Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tools
}

// HasTools 检查是否有可用工具
func (c *Client) HasTools() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.tools) > 0
}

// Prompts 获取所有提示模板（带 serverName__ 前缀）
func (c *Client) Prompts() []*mcp.Prompt {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.prompts
}

// HasPrompts 检查是否有可用提示模板
func (c *Client) HasPrompts() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.prompts) > 0
}

// Resources 获取所有资源
func (c *Client) Resources() []*mcp.Resource {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.resources
}

// HasResources 检查是否有可用资源
func (c *Client) HasResources() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.resources) > 0
}

// CallTool 调用指定工具
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (string, error) {
	start := timex.Now()

	c.mu.RLock()
	conn, ok := c.toolRoutes[name]
	c.mu.RUnlock()
	if !ok {
		c.metrics.AddDrop()
		return "", fmt.Errorf("[mcpx] tool %q not found", name)
	}

	result, err := conn.callTool(ctx, name, args)
	if err != nil {
		c.metrics.AddDrop()
		return "", err
	}

	c.metrics.Add(stat.Task{Duration: timex.Since(start)})
	return result, nil
}

// GetPrompt 获取指定提示模板内容
func (c *Client) GetPrompt(ctx context.Context, name string, args map[string]string) (*mcp.GetPromptResult, error) {
	c.mu.RLock()
	conn, ok := c.promptRoutes[name]
	c.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("[mcpx] prompt %q not found", name)
	}

	return conn.getPrompt(ctx, name, args)
}

// ReadResource 读取指定资源内容
func (c *Client) ReadResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	c.mu.RLock()
	conn, ok := c.resourceRoutes[uri]
	c.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("[mcpx] resource %q not found", uri)
	}

	return conn.readResource(ctx, uri)
}

// Close 关闭客户端，释放所有连接
func (c *Client) Close() {
	c.cancel()
	for _, conn := range c.connections {
		conn.cancel()
	}
}

// GetConnectionState 获取所有连接的状态信息
func (c *Client) GetConnectionState() map[string]ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	states := make(map[string]ConnectionState)
	for name, conn := range c.connections {
		states[name] = conn.getConnectionState()
	}
	return states
}

// refreshAll 刷新所有连接的 tools、prompts、resources
func (c *Client) refreshAll() {
	for _, conn := range c.connections {
		conn.refreshAll()
	}
	c.rebuildAll()
}

// refreshTools 刷新所有连接的工具列表
func (c *Client) refreshTools() {
	for _, conn := range c.connections {
		if err := conn.loadTools(); err != nil {
			logx.WithContext(conn.ctx).Errorf("[mcpx] %s refresh tools failed: %v", conn.name, err)
		}
	}
	c.rebuildTools()
}

// refreshPrompts 刷新所有连接的提示模板列表
func (c *Client) refreshPrompts() {
	for _, conn := range c.connections {
		if err := conn.loadPrompts(); err != nil {
			logx.WithContext(conn.ctx).Errorf("[mcpx] %s refresh prompts failed: %v", conn.name, err)
		}
	}
	c.rebuildPrompts()
}

// refreshResources 刷新所有连接的资源列表
func (c *Client) refreshResources() {
	for _, conn := range c.connections {
		if err := conn.loadResources(); err != nil {
			logx.WithContext(conn.ctx).Errorf("[mcpx] %s refresh resources failed: %v", conn.name, err)
		}
	}
	c.rebuildResources()
}

// rebuildAll 重建所有路由（tools、prompts、resources）
func (c *Client) rebuildAll() {
	c.rebuildTools()
	c.rebuildPrompts()
	c.rebuildResources()
}

// rebuildTools 重建工具路由
func (c *Client) rebuildTools() {
	var allTools []*mcp.Tool
	toolRoutes := make(map[string]*Connection)

	for _, conn := range c.connections {
		for _, t := range conn.getTools() {
			prefixed := conn.name + ToolNameSeparator + t.Name
			allTools = append(allTools, &mcp.Tool{
				Name:        prefixed,
				Description: t.Description,
				InputSchema: t.InputSchema,
				Annotations: t.Annotations,
			})
			toolRoutes[prefixed] = conn
		}
	}

	c.mu.Lock()
	c.tools = allTools
	c.toolRoutes = toolRoutes
	c.mu.Unlock()
}

// rebuildPrompts 重建提示模板路由
func (c *Client) rebuildPrompts() {
	var allPrompts []*mcp.Prompt
	promptRoutes := make(map[string]*Connection)

	for _, conn := range c.connections {
		for _, p := range conn.getPrompts() {
			prefixed := conn.name + ToolNameSeparator + p.Name
			allPrompts = append(allPrompts, &mcp.Prompt{
				Name:        prefixed,
				Description: p.Description,
				Arguments:   p.Arguments,
			})
			promptRoutes[prefixed] = conn
		}
	}

	c.mu.Lock()
	c.prompts = allPrompts
	c.promptRoutes = promptRoutes
	c.mu.Unlock()
}

// rebuildResources 重建资源路由
func (c *Client) rebuildResources() {
	var allResources []*mcp.Resource
	resourceRoutes := make(map[string]*Connection)

	for _, conn := range c.connections {
		for _, r := range conn.getResources() {
			allResources = append(allResources, &mcp.Resource{
				URI:         r.URI,
				Name:        r.Name,
				Description: r.Description,
				MIMEType:    r.MIMEType,
			})
			resourceRoutes[r.URI] = conn
		}
	}

	c.mu.Lock()
	c.resources = allResources
	c.resourceRoutes = resourceRoutes
	c.mu.Unlock()
}

// run 连接管理主循环，负责建立连接、处理断线重连
func (conn *Connection) run(opts *mcp.ClientOptions) {
	for {
		session := conn.tryConnect(opts)
		if session != nil {
			_ = session.Wait()
			logx.WithContext(conn.ctx).Errorf("[mcpx] %s disconnected", conn.name)
			conn.mu.Lock()
			conn.session = nil
			conn.tools = nil
			conn.prompts = nil
			conn.resources = nil
			conn.mu.Unlock()
			conn.onChange()
		}

		t := time.NewTimer(conn.cfg.RefreshInterval)
		select {
		case <-conn.ctx.Done():
			t.Stop()
			return
		case <-t.C:
		}
	}
}

// tryConnect 尝试建立 MCP 连接
func (conn *Connection) tryConnect(opts *mcp.ClientOptions) *mcp.ClientSession {
	httpClient := &http.Client{Transport: &ctxHeaderTransport{base: http.DefaultTransport, serviceToken: conn.serviceToken}}

	var transport mcp.Transport

	if conn.useStreamable() {
		transport = &mcp.StreamableClientTransport{
			Endpoint:   conn.endpoint,
			HTTPClient: httpClient,
		}
	} else {
		transport = &mcp.SSEClientTransport{
			Endpoint:   conn.endpoint,
			HTTPClient: httpClient,
		}
	}

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "mcpx-" + conn.name,
		Version: "2.0.0",
	}, opts)

	session, err := client.Connect(conn.ctx, transport, nil)
	if err != nil {
		logx.WithContext(conn.ctx).Errorf("[mcpx] %s connect: %v", conn.name, err)
		return nil
	}

	conn.mu.Lock()
	conn.client = client
	conn.session = session
	conn.transport = transport
	conn.mu.Unlock()

	if err := conn.loadAll(); err != nil {
		logx.WithContext(conn.ctx).Errorf("[mcpx] %s load resources: %v", conn.name, err)
		_ = session.Close()
		return nil
	}

	conn.onChange()
	logx.WithContext(conn.ctx).Infof("[mcpx] %s connected, endpoint=%s, tools=%d, prompts=%d, resources=%d",
		conn.name, conn.endpoint, len(conn.tools), len(conn.prompts), len(conn.resources))
	return session
}

// loadAll 加载所有 MCP 资源（tools、prompts、resources）
func (conn *Connection) loadAll() error {
	if err := conn.loadTools(); err != nil {
		return err
	}
	if err := conn.loadPrompts(); err != nil {
		return err
	}
	if err := conn.loadResources(); err != nil {
		return err
	}
	return nil
}

// loadTools 从服务器加载工具列表
func (conn *Connection) loadTools() error {
	conn.mu.RLock()
	session := conn.session
	conn.mu.RUnlock()
	if session == nil {
		return fmt.Errorf("no session")
	}

	ctx, cancel := context.WithTimeout(conn.ctx, conn.cfg.ConnectTimeout)
	defer cancel()

	var tools []*mcp.Tool
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			return fmt.Errorf("load tools failed: %w", err)
		}
		tools = append(tools, tool)
	}

	conn.mu.Lock()
	conn.tools = tools
	conn.mu.Unlock()

	return nil
}

// loadPrompts 从服务器加载提示模板列表
func (conn *Connection) loadPrompts() error {
	conn.mu.RLock()
	session := conn.session
	conn.mu.RUnlock()
	if session == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(conn.ctx, conn.cfg.ConnectTimeout)
	defer cancel()

	var prompts []*mcp.Prompt
	for prompt, err := range session.Prompts(ctx, nil) {
		if err != nil {
			return fmt.Errorf("load prompts failed: %w", err)
		}
		prompts = append(prompts, prompt)
	}

	conn.mu.Lock()
	conn.prompts = prompts
	conn.mu.Unlock()

	return nil
}

// loadResources 从服务器加载资源列表
func (conn *Connection) loadResources() error {
	conn.mu.RLock()
	session := conn.session
	conn.mu.RUnlock()
	if session == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(conn.ctx, conn.cfg.ConnectTimeout)
	defer cancel()

	var resources []*mcp.Resource
	for resource, err := range session.Resources(ctx, nil) {
		if err != nil {
			return fmt.Errorf("load resources failed: %w", err)
		}
		resources = append(resources, resource)
	}

	conn.mu.Lock()
	conn.resources = resources
	conn.mu.Unlock()

	return nil
}

// refreshAll 刷新所有 MCP 资源
func (conn *Connection) refreshAll() {
	if err := conn.loadAll(); err != nil {
		logx.WithContext(conn.ctx).Errorf("[mcpx] %s refresh failed: %v", conn.name, err)
		return
	}
	conn.onChange()
	logx.WithContext(conn.ctx).Infof("[mcpx] %s refreshed: tools=%d, prompts=%d, resources=%d",
		conn.name, len(conn.tools), len(conn.prompts), len(conn.resources))
}

// getTools 获取工具列表（线程安全）
func (conn *Connection) getTools() []*mcp.Tool {
	conn.mu.RLock()
	defer conn.mu.RUnlock()
	return conn.tools
}

// getPrompts 获取提示模板列表（线程安全）
func (conn *Connection) getPrompts() []*mcp.Prompt {
	conn.mu.RLock()
	defer conn.mu.RUnlock()
	return conn.prompts
}

// getResources 获取资源列表（线程安全）
func (conn *Connection) getResources() []*mcp.Resource {
	conn.mu.RLock()
	defer conn.mu.RUnlock()
	return conn.resources
}

// callTool 调用指定工具
func (conn *Connection) callTool(ctx context.Context, name string, args map[string]any) (string, error) {
	conn.mu.RLock()
	session := conn.session
	conn.mu.RUnlock()
	if session == nil {
		return "", fmt.Errorf("[mcpx] %s not connected", conn.name)
	}

	realName := stripServerPrefix(name)

	params := &mcp.CallToolParams{Name: realName, Arguments: args}

	if meta := ctxprop.CollectFromCtx(ctx); len(meta) > 0 {
		if meta[ctxdata.CtxAuthTypeKey] == nil {
			meta[ctxdata.CtxAuthTypeKey] = "user"
		}
		params.SetMeta(meta)
	} else {
		params.SetMeta(map[string]any{ctxdata.CtxAuthTypeKey: "service"})
	}

	result, err := session.CallTool(ctx, params)
	if err != nil {
		return "", fmt.Errorf("[mcpx] call %q on %s: %w", realName, conn.name, err)
	}

	return conn.formatToolResult(result), nil
}

// getPrompt 获取指定提示模板内容
func (conn *Connection) getPrompt(ctx context.Context, name string, args map[string]string) (*mcp.GetPromptResult, error) {
	conn.mu.RLock()
	session := conn.session
	conn.mu.RUnlock()
	if session == nil {
		return nil, fmt.Errorf("[mcpx] %s not connected", conn.name)
	}

	realName := name
	if idx := strings.Index(name, ToolNameSeparator); idx >= 0 {
		realName = name[idx+len(ToolNameSeparator):]
	}

	params := &mcp.GetPromptParams{Name: realName, Arguments: args}

	result, err := session.GetPrompt(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("[mcpx] get prompt %q on %s: %w", realName, conn.name, err)
	}

	return result, nil
}

// readResource 读取指定资源内容
func (conn *Connection) readResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	conn.mu.RLock()
	session := conn.session
	conn.mu.RUnlock()
	if session == nil {
		return nil, fmt.Errorf("[mcpx] %s not connected", conn.name)
	}

	params := &mcp.ReadResourceParams{URI: uri}

	result, err := session.ReadResource(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("[mcpx] read resource %q on %s: %w", uri, conn.name, err)
	}

	return result, nil
}

// formatToolResult 格式化工具调用结果为字符串
func (conn *Connection) formatToolResult(result *mcp.CallToolResult) string {
	var sb strings.Builder
	for _, c := range result.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(tc.Text)
		}
	}
	return sb.String()
}

// ConnectionState 连接状态信息
type ConnectionState struct {
	Status        string
	ConnectedAt   time.Time
	LastError     error
	ToolCount     int
	PromptCount   int
	ResourceCount int
	IsConnected   bool
}

// getConnectionState 获取连接状态（线程安全）
func (conn *Connection) getConnectionState() ConnectionState {
	conn.mu.RLock()
	defer conn.mu.RUnlock()

	state := ConnectionState{
		Status: "disconnected",
	}

	if conn.session != nil {
		state.Status = "connected"
		state.ConnectedAt = time.Now()
		state.ToolCount = len(conn.tools)
		state.PromptCount = len(conn.prompts)
		state.ResourceCount = len(conn.resources)
		state.IsConnected = true
	}

	return state
}

// useStreamable 检查是否使用 Streamable 传输协议
func (conn *Connection) useStreamable() bool {
	for _, sc := range conn.cfg.Servers {
		if sc.Name == conn.name {
			return sc.UseStreamable
		}
	}
	return false
}

// ParseArgs 解析 JSON 格式的参数字符串
func ParseArgs(argsJSON string) map[string]any {
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return map[string]any{}
	}
	return args
}

// stripServerPrefix 从名称中移除服务器前缀（serverName__）
func stripServerPrefix(name string) string {
	if idx := strings.Index(name, ToolNameSeparator); idx >= 0 {
		return name[idx+len(ToolNameSeparator):]
	}
	return name
}

// ctxHeaderTransport 自定义 HTTP 传输，用于注入用户上下文和鉴权信息
type ctxHeaderTransport struct {
	base         http.RoundTripper
	serviceToken string
}

// RoundTrip 实现 http.RoundTripper 接口，在请求中注入用户上下文和鉴权信息
func (t *ctxHeaderTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	ctxprop.InjectToHTTPHeader(r.Context(), r.Header)

	var authType string
	token := r.Header.Get("Authorization")

	if token != "" {
		authType = "user"
	} else if t.serviceToken != "" {
		r.Header.Set("Authorization", "Bearer "+t.serviceToken)
		authType = "service"
	} else {
		authType = "none"
	}

	r.Header.Set("X-Auth-Type", authType)
	logx.WithContext(r.Context()).Debugf("[mcpx] transport: authType=%s, method=%s, path=%s",
		authType, r.Method, r.URL.Path)
	return t.base.RoundTrip(r)
}
