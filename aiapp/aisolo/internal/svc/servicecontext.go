package svc

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/aiapp/aisolo/internal/agent"
	"zero-service/aiapp/aisolo/internal/config"
	"zero-service/aiapp/aisolo/internal/router"
	"zero-service/common/einox/memory"
	exinoModel "zero-service/common/einox/model"
)

// ServiceContext 服务上下文
type ServiceContext struct {
	Config config.Config

	// 智能路由器
	Router router.Router

	// 记忆存储
	MemoryStorage memory.Storage

	// Agent 管理器
	AgentManager *agent.Manager

	// ChatModel
	ChatModel model.BaseChatModel

	// DefaultAgent 默认 Agent
	DefaultAgent adk.Agent
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	ctx := context.Background()
	svc := &ServiceContext{
		Config: c,
	}

	// 初始化记忆存储
	svc.initMemoryStorage()

	// 初始化智能路由器
	svc.initRouter()

	// 初始化 ChatModel
	svc.initChatModel(ctx)

	// 初始化 Agent 管理器
	svc.initAgentManager(ctx)

	return svc
}

// initMemoryStorage 初始化记忆存储
func (s *ServiceContext) initMemoryStorage() {
	// 使用内存存储
	s.MemoryStorage = memory.NewMemoryStorage()
	logx.Info("MemoryStorage initialized: in-memory")
}

// initRouter 初始化智能路由器
func (s *ServiceContext) initRouter() {
	s.Router = router.NewTwoLevelRouter(
		router.WithSimpleThreshold(s.Config.Router.SimpleThreshold),
	)
}

// initChatModel 初始化 ChatModel
func (s *ServiceContext) initChatModel(ctx context.Context) {
	// 从配置获取默认模型
	defaultModel := s.Config.DefaultModel
	if defaultModel == "" {
		defaultModel = "deepseek-v3-2-251201"
	}

	// 构建模型配置
	cfg := exinoModel.Config{
		Provider: exinoModel.ProviderArk,
		Model:    defaultModel,
	}

	// 从 Providers 配置获取 API Key
	for _, p := range s.Config.Providers {
		if p.Name == "ark" {
			cfg.APIKey = p.ApiKey
			if p.Endpoint != "" {
				cfg.BaseURL = p.Endpoint
			}
		}
	}

	// 从 Models 配置获取详细配置
	for _, m := range s.Config.Models {
		if m.Id == defaultModel {
			cfg.Provider = exinoModel.Provider(m.Provider)
			cfg.Model = m.BackendModel
			if m.MaxTokens > 0 {
				cfg.MaxTokens = m.MaxTokens
			}
		}
	}

	chatModel, err := exinoModel.NewChatModel(ctx, cfg)
	if err != nil {
		logx.Errorf("init chat model failed: %v", err)
		return
	}

	s.ChatModel = chatModel
	logx.Infof("ChatModel initialized: %s", defaultModel)
}

// initAgentManager 初始化 Agent 管理器
func (s *ServiceContext) initAgentManager(ctx context.Context) {
	if s.ChatModel == nil {
		logx.Info("ChatModel not initialized, skip AgentManager initialization")
		return
	}

	manager, err := agent.NewManager(&s.Config, s.ChatModel)
	if err != nil {
		logx.Errorf("init agent manager failed: %v", err)
		return
	}

	s.AgentManager = manager
	logx.Infof("AgentManager initialized with %d agents", len(manager.List()))
}

// GetAgent 获取 Agent
func (s *ServiceContext) GetAgent(agentType string) adk.Agent {
	if s.AgentManager == nil {
		return nil
	}
	wrappedAgent := s.AgentManager.GetByType(agentType)
	if wrappedAgent == nil {
		return nil
	}
	return wrappedAgent.GetADKAgent()
}

// Close 关闭资源
func (s *ServiceContext) Close() error {
	return nil
}
