// Package agent 提供构建 AI Agent 的极简封装。
//
// 设计对齐 eino-examples/quickstart/chatwitheino:
//   - 一个 Agent 持有 adk.Agent + *adk.Runner;
//   - 通过 NewChatModelAgent / NewDeepAgent 创建;
//   - 不管理消息存储 (memory.Storage) 或会话, 这些由上层 aisolo 负责;
//   - 只暴露最小 API: Runner() / Stop() / Name() / GetAgent().
//
// 关于状态/健康/指标:
//   - 这些不属于本封装的职责, 由上层服务统一通过 logx/Prometheus 等通用工具收集,
//     以避免在每个 Agent 实例上维护多余的 mutex/atomic 字段。
package agent

import (
	"context"
	"fmt"
	"os"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/zeromicro/go-zero/core/logx"
)

// Agent 是对 adk.Agent + adk.Runner 的极简封装, 提供给业务层使用。
type Agent struct {
	name     string
	adkAgent adk.Agent
	runner   *adk.Runner
	opts     options
}

// New 基于 options 创建 ChatModel 风格的 Agent。
func New(ctx context.Context, opts ...Option) (*Agent, error) {
	cfg := newOptions(opts...)
	if cfg.model == nil {
		return nil, fmt.Errorf("agent: model is required")
	}

	chatAgent, err := buildChatModelAgent(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("agent: create chat agent: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           chatAgent,
		EnableStreaming: true,
		CheckPointStore: cfg.checkpointStore,
	})

	return &Agent{
		name:     cfg.name,
		adkAgent: chatAgent,
		runner:   runner,
		opts:     *cfg,
	}, nil
}

// Runner 返回底层 adk.Runner, 业务层使用它执行 Run/Resume。
func (a *Agent) Runner() *adk.Runner { return a.runner }

// GetAgent 返回底层 adk.Agent, 通常不直接使用。
func (a *Agent) GetAgent() adk.Agent { return a.adkAgent }

// Name 返回 Agent 名称。
func (a *Agent) Name() string { return a.name }

// Stop 释放 Agent 占用的资源。当前 adk.Runner 没有显式 Close, 这里只做日志占位,
// 保持接口稳定, 方便后续接入连接池/缓存清理等清理动作。
func (a *Agent) Stop(_ context.Context) error {
	logx.Infof("[agent] %s stopped", a.name)
	return nil
}

// =============================================================================
// 内部构造
// =============================================================================

func buildChatModelAgent(ctx context.Context, cfg *options) (*adk.ChatModelAgent, error) {
	description := cfg.description
	if description == "" {
		description = cfg.name + " - AI Assistant"
	}

	chatModel, ok := cfg.model.(model.BaseChatModel)
	if !ok {
		return nil, fmt.Errorf("agent: model must implement model.BaseChatModel")
	}

	agentCfg := &adk.ChatModelAgentConfig{
		Name:        cfg.name,
		Description: description,
		Instruction: cfg.instruction,
		Model:       chatModel,
	}

	if len(cfg.tools) > 0 {
		agentCfg.ToolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{Tools: cfg.tools},
		}
	}

	skillHandlers := buildSkillHandlers(ctx, cfg)
	cfg.handlers = append(skillHandlers, cfg.handlers...)

	if len(cfg.handlers) > 0 {
		agentCfg.Handlers = cfg.handlers
	}
	if len(cfg.middlewares) > 0 {
		agentCfg.Middlewares = cfg.middlewares
	}

	return adk.NewChatModelAgent(ctx, agentCfg)
}

// buildSkillHandlers 根据 skillsDir 构造 skill 中间件 (参考 chatwitheino)。
func buildSkillHandlers(ctx context.Context, cfg *options) []adk.ChatModelAgentMiddleware {
	dir := cfg.skillsDir
	if dir == "" {
		dir = os.Getenv("EINO_EXT_SKILLS_DIR")
	}
	if dir == "" {
		return nil
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		if err != nil {
			logx.Errorf("[agent] skills dir unavailable (skill middleware skipped): dir=%s err=%v", dir, err)
		} else {
			logx.Errorf("[agent] skills path is not a directory (skill middleware skipped): dir=%s", dir)
		}
		return nil
	}

	backend, err := localbk.NewBackend(ctx, &localbk.Config{})
	if err != nil {
		logx.Errorf("[agent] create local backend: %v", err)
		return nil
	}
	skillBackend, err := skill.NewBackendFromFilesystem(ctx, &skill.BackendFromFilesystemConfig{
		Backend: backend,
		BaseDir: dir,
	})
	if err != nil {
		logx.Errorf("[agent] create skill backend: %v", err)
		return nil
	}
	skillMw, err := skill.NewMiddleware(ctx, &skill.Config{Backend: skillBackend})
	if err != nil {
		logx.Errorf("[agent] create skill middleware: %v", err)
		return nil
	}
	logx.Infof("[agent] skill middleware loaded: dir=%s", dir)
	return []adk.ChatModelAgentMiddleware{skillMw}
}
