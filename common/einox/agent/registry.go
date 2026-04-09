package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/common/einox"
)

// =============================================================================
// AgentRegistry - Agent 注册与管理中心
// =============================================================================

// AgentRegistry Agent 注册表
type AgentRegistry struct {
	mu     sync.RWMutex
	agents map[string]einox.AgentInterface
}

// NewAgentRegistry 创建新的 Agent 注册表
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]einox.AgentInterface),
	}
}

// Register 注册 Agent
func (r *AgentRegistry) Register(name string, agent einox.AgentInterface) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if name == "" {
		return fmt.Errorf("agent name cannot be empty")
	}
	if agent == nil {
		return fmt.Errorf("agent cannot be nil")
	}
	if _, exists := r.agents[name]; exists {
		return fmt.Errorf("agent %s already registered", name)
	}

	r.agents[name] = agent
	logx.Infof("[AgentRegistry] Registered agent: %s", name)
	return nil
}

// Unregister 注销 Agent
func (r *AgentRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[name]; !exists {
		return fmt.Errorf("agent %s not found", name)
	}

	delete(r.agents, name)
	logx.Infof("[AgentRegistry] Unregistered agent: %s", name)
	return nil
}

// Get 获取 Agent
func (r *AgentRegistry) Get(name string) (einox.AgentInterface, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, exists := r.agents[name]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", name)
	}
	return agent, nil
}

// List 列出所有 Agent
func (r *AgentRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}
	return names
}

// Clear 清空所有 Agent
func (r *AgentRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for name := range r.agents {
		delete(r.agents, name)
	}
	logx.Infof("[AgentRegistry] Cleared all agents")
}

// ClearMemory 清除所有 Agent 的记忆
func (r *AgentRegistry) ClearMemory(ctx context.Context, userID, sessionID string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var lastErr error
	for name, agent := range r.agents {
		if err := agent.ClearMemory(ctx, userID, sessionID); err != nil {
			logx.Errorf("[AgentRegistry] ClearMemory failed for %s: %v", name, err)
			lastErr = err
		}
	}
	return lastErr
}

// Count 返回注册的 Agent 数量
func (r *AgentRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.agents)
}

// Clone 克隆注册表（用于只读访问）
func (r *AgentRegistry) Clone() map[string]einox.AgentInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()

	clone := make(map[string]einox.AgentInterface, len(r.agents))
	for k, v := range r.agents {
		clone[k] = v
	}
	return clone
}

// =============================================================================
// GlobalAgentRegistry - 全局单例 Agent 注册表
// =============================================================================

var (
	globalRegistry     *AgentRegistry
	globalRegistryOnce sync.Once
)

// GetGlobalRegistry 获取全局 Agent 注册表（单例）
func GetGlobalRegistry() *AgentRegistry {
	globalRegistryOnce.Do(func() {
		globalRegistry = NewAgentRegistry()
	})
	return globalRegistry
}

// =============================================================================
// AgentManager - Agent 生命周期管理器
// =============================================================================

// AgentManager Agent 生命周期管理器
type AgentManager struct {
	registry *AgentRegistry
}

// NewAgentManager 创建 Agent 生命周期管理器
func NewAgentManager() *AgentManager {
	return &AgentManager{
		registry: NewAgentRegistry(),
	}
}

// GetRegistry 获取注册表
func (m *AgentManager) GetRegistry() *AgentRegistry {
	return m.registry
}

// InitAgents 初始化一组 Agent
func (m *AgentManager) InitAgents(ctx context.Context, agents map[string]*Agent) error {
	for name, agent := range agents {
		if err := m.registry.Register(name, agent); err != nil {
			return fmt.Errorf("register agent %s: %w", name, err)
		}
	}
	return nil
}

// Shutdown 关闭所有 Agent
func (m *AgentManager) Shutdown(ctx context.Context) error {
	// 清除所有 Agent 的注册
	m.registry.Clear()
	logx.Info("[AgentManager] Shutdown complete")
	return nil
}

// HealthCheck 健康检查
func (m *AgentManager) HealthCheck(ctx context.Context) map[string]bool {
	names := m.registry.List()

	status := make(map[string]bool)
	for _, name := range names {
		status[name] = true // Agent 存在即认为健康
	}
	return status
}
