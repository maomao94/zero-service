package svc

import (
	"context"
	"time"

	"zero-service/aiapp/aisolo/internal/agent"
	"zero-service/aiapp/aisolo/internal/config"
	"zero-service/aiapp/aisolo/internal/roles"
	"zero-service/aiapp/aisolo/internal/router"
	"zero-service/aiapp/aisolo/internal/tool"
	"zero-service/common/einox/memory"
	exinoModel "zero-service/common/einox/model"

	"github.com/cloudwego/eino/components/model"
	"github.com/zeromicro/go-zero/core/logx"
)

// ServiceContext 服务上下文
type ServiceContext struct {
	Config config.Config

	// ChatModel
	ChatModel model.BaseChatModel

	// 记忆存储
	MemoryStorage memory.Storage

	// 角色管理器
	RoleManager *roles.RoleManager

	// Agent池
	AgentPool *agent.AgentPool

	// 请求路由器
	Router *router.Router

	// 工具管理器
	ToolManager *tool.ToolManager
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	ctx := context.Background()
	svc := &ServiceContext{
		Config: c,
	}

	// 初始化记忆存储
	svc.initMemoryStorage()

	// 初始化 ChatModel
	svc.initChatModel(ctx)

	// 初始化角色管理器
	svc.initRoleManager()

	// 初始化Agent池
	svc.initAgentPool()

	// 初始化请求路由器
	svc.initRouter()

	// 初始化工具管理器
	svc.initToolManager(ctx)

	return svc
}

// initMemoryStorage 初始化记忆存储
func (s *ServiceContext) initMemoryStorage() {
	s.MemoryStorage = memory.NewMemoryStorage()
	logx.Info("MemoryStorage initialized: in-memory")
}

// initChatModel 初始化 ChatModel
func (s *ServiceContext) initChatModel(ctx context.Context) {
	cfg := exinoModel.Config{
		Provider: exinoModel.Provider(s.Config.Model.Provider),
		Model:    s.Config.Model.Model,
		APIKey:   s.Config.Model.APIKey,
	}

	if s.Config.Model.BaseURL != "" {
		cfg.BaseURL = s.Config.Model.BaseURL
	}

	chatModel, err := exinoModel.NewChatModel(ctx, cfg)
	if err != nil {
		logx.Errorf("init chat model failed: %v", err)
		return
	}

	s.ChatModel = chatModel
	logx.Infof("ChatModel initialized: %s", s.Config.Model.Model)
}

// initRoleManager 初始化角色管理器
func (s *ServiceContext) initRoleManager() {
	if s.ChatModel == nil {
		logx.Info("ChatModel not initialized, skip RoleManager initialization")
		return
	}

	// 根据配置决定是否启用 skills 和内置工具
	var rm *roles.RoleManager
	if s.Config.Skills.Enabled && s.Config.Skills.Dir != "" {
		rm = roles.NewRoleManagerWithSkillsAndTools(s.ChatModel, s.Config.Skills.Dir)
		logx.Infof("RoleManager initialized with %d roles + skills + builtin tools, skills dir: %s", len(roles.BuiltinRoles), s.Config.Skills.Dir)
	} else {
		rm = roles.NewRoleManagerWithBuiltinTools(s.ChatModel)
		logx.Infof("RoleManager initialized with %d roles + builtin tools", len(roles.BuiltinRoles))
	}
	s.RoleManager = rm
}

// initAgentPool 初始化Agent池
func (s *ServiceContext) initAgentPool() {
	if s.RoleManager == nil {
		logx.Info("RoleManager not initialized, skip AgentPool initialization")
		return
	}

	maxIdle := 10
	maxLive := 1 * time.Hour
	if s.Config.Agent.PoolMaxIdle > 0 {
		maxIdle = s.Config.Agent.PoolMaxIdle
	}
	if s.Config.Agent.PoolMaxLive > 0 {
		maxLive = s.Config.Agent.PoolMaxLive
	}

	s.AgentPool = agent.NewAgentPool(s.RoleManager, maxIdle, maxLive)
	logx.Infof("AgentPool initialized: maxIdle=%d, maxLive=%v", maxIdle, maxLive)
}

// initRouter 初始化请求路由器
func (s *ServiceContext) initRouter() {
	if s.AgentPool == nil {
		logx.Info("AgentPool not initialized, skip Router initialization")
		return
	}

	s.Router = router.NewRouter(s.AgentPool)
	logx.Info("Router initialized")
}

// initToolManager 初始化工具管理器
func (s *ServiceContext) initToolManager(ctx context.Context) {
	if !s.Config.Tools.Enabled {
		logx.Info("Tools disabled, skip ToolManager initialization")
		return
	}

	registry := tool.NewRegistry()
	config := tool.ToolConfig{
		Timeout:        30 * time.Second,
		MaxRetries:     3,
		MaxConcurrency: 10,
	}
	if s.Config.Tools.Timeout > 0 {
		config.Timeout = s.Config.Tools.Timeout
	}
	if s.Config.Tools.MaxRetries > 0 {
		config.MaxRetries = s.Config.Tools.MaxRetries
	}
	if s.Config.Tools.MaxConcurrency > 0 {
		config.MaxConcurrency = s.Config.Tools.MaxConcurrency
	}

	s.ToolManager = tool.NewToolManager(registry, config)
	logx.Infof("ToolManager initialized: timeout=%v, maxRetries=%d, maxConcurrency=%d",
		config.Timeout, config.MaxRetries, config.MaxConcurrency)
}

// Close 关闭资源
func (s *ServiceContext) Close() error {
	if s.AgentPool != nil {
		s.AgentPool.Cleanup()
	}
	return nil
}
