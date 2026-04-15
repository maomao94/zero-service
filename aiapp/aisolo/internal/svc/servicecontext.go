package svc

import (
	"context"

	"zero-service/aiapp/aisolo/internal/config"
	"zero-service/aiapp/aisolo/internal/roles"
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
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
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

// Close 关闭资源
func (s *ServiceContext) Close() error {
	return nil
}
