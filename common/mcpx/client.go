package mcpx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
	"zero-service/common/tool"

	"zero-service/common/antsx"
	"zero-service/common/ctxprop"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/core/timex"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
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

	// 配置选项（使用 SDK 类型）
	options *mcp.ClientOptions

	// 进度事件发射器，按 token 分发进度
	progressEmitter *antsx.EventEmitter[ProgressInfo]

	// Reactor goroutine 池，用于异步工具调用
	reactor *antsx.Reactor

	// 异步结果存储
	asyncResultStore AsyncResultStore
}

// ClientOption Client 配置选项
type ClientOption func(*Client)

// WithAsyncResultStore 设置异步结果存储
func WithAsyncResultStore(store AsyncResultStore) ClientOption {
	return func(c *Client) {
		c.asyncResultStore = store
	}
}

// WithReactor 设置 goroutine 池
func WithReactor(reactor *antsx.Reactor) ClientOption {
	return func(c *Client) {
		c.reactor = reactor
	}
}

// WithOptions 设置完整配置选项（使用 SDK 的 *mcp.ClientOptions）
func WithOptions(opts *mcp.ClientOptions) ClientOption {
	return func(c *Client) {
		c.options = opts
	}
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

	// 引用 Client，用于访问 progressEmitter
	clientRef *Client
}

// ProgressInfo 进度信息
type ProgressInfo struct {
	Token    string  // ProgressToken
	Progress float64 // 当前进度
	Total    float64 // 总进度
	Message  string  // 进度消息
}

// Percent 计算进度百分比，返回 0-100 的整数
func (p *ProgressInfo) Percent() int {
	// total > 1 表示真正的进度值，计算百分比
	if p.Total > 1 {
		return int(p.Progress / p.Total * 100)
	}
	// progress 本身就是百分比
	if p.Progress > 0 && p.Progress <= 100 {
		return int(p.Progress)
	}
	// 不合理情况，默认 100
	return 100
}

// ProgressCallback 进度回调函数
type ProgressCallback func(info *ProgressInfo)

// CallToolWithProgressRequest 带进度通知的工具调用请求
type CallToolWithProgressRequest struct {
	Token      string
	Name       string           // 工具名称
	Args       map[string]any   // 工具参数
	OnProgress ProgressCallback // 进度回调
}

// NewClient 创建 MCP 客户端
// 使用 ClientOption 配置选项（如 WithAsyncResultStore、WithReactor）
func NewClient(cfg Config, opts ...ClientOption) *Client {
	if cfg.RefreshInterval <= 0 {
		cfg.RefreshInterval = DefaultRefreshInterval
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = DefaultConnectTimeout
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		config:           cfg,
		connections:      make(map[string]*Connection),
		toolRoutes:       make(map[string]*Connection),
		promptRoutes:     make(map[string]*Connection),
		resourceRoutes:   make(map[string]*Connection),
		metrics:          stat.NewMetrics("mcpx"),
		ctx:              ctx,
		cancel:           cancel,
		options:          &mcp.ClientOptions{},
		progressEmitter:  antsx.NewEventEmitter[ProgressInfo](),
		asyncResultStore: NewEmptyAsyncResultStore(),
	}
	c.reactor, _ = antsx.NewReactor(1000)

	// 应用 Option 配置
	for _, opt := range opts {
		opt(c)
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
			clientRef:    c,
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

	// 进度通知处理器（内部实现）
	opts.ProgressNotificationHandler = func(ctx context.Context, req *mcp.ProgressNotificationClientRequest) {
		token := fmt.Sprintf("%v", req.Params.ProgressToken)
		c.progressEmitter.Emit(token, ProgressInfo{
			Token:    token,
			Progress: req.Params.Progress,
			Total:    req.Params.Total,
			Message:  req.Params.Message,
		})
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

// CallToolWithProgress 带进度通知的工具调用
// 进度会通过 SSE EventEmitter 推送到浏览器
func (c *Client) CallToolWithProgress(ctx context.Context, req *CallToolWithProgressRequest) (string, error) {
	start := timex.Now()

	c.mu.RLock()
	conn, ok := c.toolRoutes[req.Name]
	c.mu.RUnlock()
	if !ok {
		c.metrics.AddDrop()
		return "", fmt.Errorf("[mcpx] tool %q not found", req.Name)
	}

	logx.Infof("[mcpx] CallToolWithProgress: name=%s, token=%s, hasCallback=%v", req.Name, req.Token, req.OnProgress != nil)

	result, err := conn.callToolWithProgress(ctx, req)
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

	if err := conn.loadAllWithRetry(); err != nil {
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

// loadAllWithRetry 带重试机制的加载所有 MCP 资源
// 在 session 初始化期间，tools/list 可能会失败，需要重试
func (conn *Connection) loadAllWithRetry() error {
	maxRetries := 3
	retryDelay := 100 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		err := conn.loadAll()
		if err == nil {
			return nil
		}

		if i < maxRetries-1 {
			logx.WithContext(conn.ctx).Debugf("[mcpx] %s load resources failed (attempt %d/%d): %v, retrying in %v...",
				conn.name, i+1, maxRetries, err, retryDelay)
			time.Sleep(retryDelay)
			retryDelay *= 2
		}
	}

	return fmt.Errorf("load resources failed after %d retries: %w", maxRetries, conn.loadAll())
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

	// 从 ctx 收集用户上下文，注入 trace 到 _meta
	meta := ctxprop.CollectFromCtx(ctx)
	otel.GetTextMapPropagator().Inject(ctx, NewMapMetaCarrier(meta))
	params.SetMeta(meta)
	result, err := session.CallTool(ctx, params)
	if err != nil {
		return "", fmt.Errorf("[mcpx] call %q on %s: %w", realName, conn.name, err)
	}

	return conn.formatToolResult(result), nil
}

// callToolWithProgress 带进度通知的工具调用
func (conn *Connection) callToolWithProgress(ctx context.Context, req *CallToolWithProgressRequest) (string, error) {
	conn.mu.RLock()
	session := conn.session
	conn.mu.RUnlock()
	if session == nil {
		return "", fmt.Errorf("[mcpx] %s not connected", conn.name)
	}

	realName := stripServerPrefix(req.Name)

	params := &mcp.CallToolParams{Name: realName, Arguments: req.Args}
	token := req.Token
	if len(token) == 0 {
		// 生成唯一的 progress token
		token, _ = tool.SimpleUUID()
	}

	// 订阅进度事件
	var cancel func()
	var progressSR *antsx.StreamReader[ProgressInfo]
	if req.OnProgress != nil {
		progressSR, cancel = conn.clientRef.progressEmitter.Subscribe(ctx, token)
		defer cancel()
	}

	meta := ctxprop.CollectFromCtx(ctx)
	otel.GetTextMapPropagator().Inject(ctx, NewMapMetaCarrier(meta))
	params.SetMeta(meta)
	if req.OnProgress != nil && progressSR != nil {
		params.SetProgressToken(token)
		threading.GoSafe(func() {
			logx.Infof("[mcpx] progress listener started: token=%s", token)
			for {
				info, err := progressSR.Recv()
				if err != nil {
					break
				}
				logx.Infof("[mcpx] progress received: token=%s, progress=%.0f/%.0f, msg=%s", token, info.Progress, info.Total, info.Message)
				req.OnProgress(&info)
			}
			logx.Infof("[mcpx] progress listener stopped: token=%s", token)
		})
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

// CallToolAsync 异步调用工具，立即返回 task_id
// 工具在后台 goroutine 执行，进度通过 TaskObserver 观察
func (c *Client) CallToolAsync(ctx context.Context, req *CallToolAsyncRequest) (string, error) {
	taskID, _ := tool.SimpleUUID()

	if c.asyncResultStore != nil {
		c.asyncResultStore.Save(ctx, &AsyncToolResult{
			TaskID:    taskID,
			Status:    "pending",
			Progress:  0,
			Total:     100,
			CreatedAt: time.Now().UnixMilli(),
			UpdatedAt: time.Now().UnixMilli(),
		})
	}

	// 创建独立 context，继承原 ctx 的值（ctxprop、OTel trace 等）
	// 但不继承 cancel，请求结束后原 ctx 会被 cancel，这里创建的不受影响
	asyncCtx := context.WithoutCancel(ctx)

	// 后台执行
	threading.GoSafe(func() {
		// 调用同步方法（带进度）
		res, callErr := c.CallToolWithProgress(asyncCtx, &CallToolWithProgressRequest{
			Token: taskID,
			Name:  req.Name,
			Args:  req.Args,
			OnProgress: func(info *ProgressInfo) {
				// 1. 保存进度到 store（幂等，先创建任务）
				if c.asyncResultStore != nil {
					if !c.asyncResultStore.Exists(asyncCtx, taskID) {
						total := 100.0
						if info.Total > 1 {
							total = info.Total
						}
						c.asyncResultStore.Save(asyncCtx, &AsyncToolResult{
							TaskID:    taskID,
							Status:    "pending",
							Progress:  0,
							Total:     total,
							CreatedAt: time.Now().UnixMilli(),
							UpdatedAt: time.Now().UnixMilli(),
						})
					}
					progress := 0.0
					if info.Total > 0 {
						progress = info.Progress / info.Total * 100
					}
					c.asyncResultStore.UpdateProgress(asyncCtx, taskID, progress, 100, info.Message)
				}
				// 2. 通知业务方
				if req.TaskObserver != nil {
					progress := 0.0
					total := 100.0
					if info.Total > 0 {
						progress = info.Progress / info.Total * 100
					}
					req.TaskObserver.OnProgress(taskID, progress, total, info.Message)
				}
			},
		})

		// 构建最终结果
		finalResult := &AsyncToolResult{
			TaskID:    taskID,
			Status:    "pending",
			Progress:  0,
			CreatedAt: time.Now().UnixMilli(),
			UpdatedAt: time.Now().UnixMilli(),
		}

		var finalMessage string
		if callErr != nil {
			finalResult.Status = "failed"
			finalResult.Error = callErr.Error()
			finalResult.Progress = 0
			finalMessage = callErr.Error()
		} else {
			finalResult.Status = "completed"
			finalResult.Result = res
			finalResult.Progress = 100
			finalMessage = "异步任务完成"
		}

		// 获取已有消息历史并追加完成消息
		if c.asyncResultStore != nil {
			if existing, err := c.asyncResultStore.Get(asyncCtx, taskID); err == nil && existing != nil {
				finalResult.Messages = existing.Messages
			}
			finalResult.Messages = append(finalResult.Messages, ProgressMessage{
				Progress: 100,
				Total:    100,
				Message:  finalMessage,
				Time:     time.Now().UnixMilli(),
			})
			c.asyncResultStore.Save(asyncCtx, finalResult)
		}

		// 回调通知
		if req.TaskObserver != nil {
			req.TaskObserver.OnComplete(taskID, finalMessage, finalResult)
		}
	})

	return taskID, nil
}

// CallToolAsyncAwait 异步调用工具，返回 Promise 可同步等待
// - 任务提交到 Reactor 池执行（goroutine 复用）
// - 进度通过 TaskObserver 推送
// - 结果通过 Promise 返回，同时自动保存到 asyncResultStore
// 返回 (taskID, Promise, error)
func (c *Client) CallToolAsyncAwait(ctx context.Context, req *CallToolAsyncRequest) (string, *antsx.Promise[string], error) {
	taskID, _ := tool.SimpleUUID()

	if c.asyncResultStore != nil {
		c.asyncResultStore.Save(ctx, &AsyncToolResult{
			TaskID:    taskID,
			Status:    "pending",
			Progress:  0,
			Total:     100,
			CreatedAt: time.Now().UnixMilli(),
			UpdatedAt: time.Now().UnixMilli(),
		})
	}

	// 创建 Promise
	promise := antsx.NewPromise[string]()

	// 保存原始 observer
	origObserver := req.TaskObserver

	// 提交到 Reactor 池执行
	_, err := antsx.Submit(ctx, c.reactor, func(ctx context.Context) (string, error) {
		// 调用同步方法（带进度回调）
		result, callErr := c.CallToolWithProgress(ctx, &CallToolWithProgressRequest{
			Token: taskID,
			Name:  req.Name,
			Args:  req.Args,
			OnProgress: func(info *ProgressInfo) {
				// 1. 保存进度到 store（幂等，先创建任务）
				if c.asyncResultStore != nil {
					if !c.asyncResultStore.Exists(ctx, taskID) {
						total := 100.0
						if info.Total > 1 {
							total = info.Total
						}
						c.asyncResultStore.Save(ctx, &AsyncToolResult{
							TaskID:    taskID,
							Status:    "pending",
							Progress:  0,
							Total:     total,
							CreatedAt: time.Now().UnixMilli(),
							UpdatedAt: time.Now().UnixMilli(),
						})
					}
					c.asyncResultStore.UpdateProgress(ctx, taskID, float64(info.Percent()), 100, info.Message)
				}
				// 2. 通知业务方
				if origObserver != nil {
					origObserver.OnProgress(taskID, float64(info.Percent()), 100, info.Message)
				}
			},
		})

		// 构建结果
		finalResult := &AsyncToolResult{
			TaskID:    taskID,
			Status:    "pending",
			Progress:  0,
			CreatedAt: time.Now().UnixMilli(),
			UpdatedAt: time.Now().UnixMilli(),
		}

		var finalMessage string
		if callErr != nil {
			finalResult.Status = "failed"
			finalResult.Error = callErr.Error()
			finalResult.Progress = 0
			finalMessage = callErr.Error()
		} else {
			finalResult.Status = "completed"
			finalResult.Result = result
			finalResult.Progress = 100
			finalMessage = "异步任务完成"
		}

		// 获取已有消息历史并追加完成消息
		if c.asyncResultStore != nil {
			if existing, err := c.asyncResultStore.Get(ctx, taskID); err == nil && existing != nil {
				finalResult.Messages = existing.Messages
			}
			finalResult.Messages = append(finalResult.Messages, ProgressMessage{
				Progress: 100,
				Total:    100,
				Message:  finalMessage,
				Time:     time.Now().UnixMilli(),
			})
			c.asyncResultStore.Save(ctx, finalResult)
		}

		// 回调通知
		if origObserver != nil {
			origObserver.OnComplete(taskID, finalMessage, finalResult)
		}

		// resolve Promise
		if callErr != nil {
			promise.Reject(callErr)
		} else {
			promise.Resolve(result)
		}

		return result, callErr
	})

	return taskID, promise, err
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

var _ propagation.TextMapCarrier = (*MapMetaCarrier)(nil)

// MapMetaCarrier MCP _meta 的 TextMapCarrier 实现，用于 OTel 链路注入
type MapMetaCarrier struct {
	meta map[string]any
}

// NewMapMetaCarrier 创建 MapMetaCarrier
func NewMapMetaCarrier(meta map[string]any) MapMetaCarrier {
	return MapMetaCarrier{meta: meta}
}

// Get 获取值
func (c MapMetaCarrier) Get(key string) string {
	if v, ok := c.meta[key].(string); ok {
		return v
	}
	return ""
}

// Set 设置值
func (c MapMetaCarrier) Set(key string, value string) {
	c.meta[key] = value
}

// Keys 返回所有键
func (c MapMetaCarrier) Keys() []string {
	keys := make([]string, 0, len(c.meta))
	for k := range c.meta {
		keys = append(keys, k)
	}
	return keys
}

// RoundTrip 实现 http.RoundTripper 接口，在请求中注入服务 Token
func (t *ctxHeaderTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// 注入 trace 等上下文到 Header
	//ctxprop.InjectToHTTPHeader(r.Context(), r.Header)

	// 设置服务 Token（连接和调用都用服务 Token）
	if t.serviceToken != "" {
		r.Header.Set("Authorization", "Bearer "+t.serviceToken)
	}

	return t.base.RoundTrip(r)
}
