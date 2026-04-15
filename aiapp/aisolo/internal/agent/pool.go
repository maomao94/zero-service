package agent

import (
	"context"
	"sync"
	"time"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/roles"
	"zero-service/common/einox"
	"zero-service/common/einox/agent"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/syncx"
)

// AgentPool Agent池
type AgentPool struct {
	pools   map[string]*syncx.Pool
	roleMgr *roles.RoleManager
	mu      sync.RWMutex
	maxIdle int
	maxLive time.Duration
}

// NewAgentPool 创建Agent池
func NewAgentPool(roleMgr *roles.RoleManager, maxIdle int, maxLive time.Duration) *AgentPool {
	p := &AgentPool{
		pools:   make(map[string]*syncx.Pool),
		roleMgr: roleMgr,
		maxIdle: maxIdle,
		maxLive: maxLive,
	}
	return p
}

// GetAgent 从池中获取Agent
func (p *AgentPool) GetAgent(ctx context.Context, roleID string, mode aisolo.AgentMode) (*agent.Agent, error) {
	key := p.getPoolKey(roleID, mode)

	p.mu.RLock()
	pool, ok := p.pools[key]
	p.mu.RUnlock()

	if !ok {
		p.mu.Lock()
		defer p.mu.Unlock()

		pool, ok = p.pools[key]
		if !ok {
			pool = syncx.NewPool(p.maxIdle, func() any {
				agent, err := p.roleMgr.CreateAgent(context.Background(), roleID)
				if err != nil {
					logx.Errorf("create agent failed: %v", err)
					return nil
				}
				return agent
			}, func(a any) {
				if agent, ok := a.(*agent.Agent); ok {
					_ = agent.Stop(context.Background())
				}
			}, syncx.WithMaxAge(p.maxLive))

			p.pools[key] = pool
		}
	}

	agent := pool.Get().(*agent.Agent)
	if agent == nil {
		return nil, einox.ErrAgentNotFound
	}

	return agent, nil
}

// PutAgent 归还Agent到池
func (p *AgentPool) PutAgent(roleID string, mode aisolo.AgentMode, agent *agent.Agent) {
	if agent == nil {
		return
	}

	key := p.getPoolKey(roleID, mode)

	p.mu.RLock()
	pool, ok := p.pools[key]
	p.mu.RUnlock()

	if ok {
		pool.Put(agent)
	}
}

// AddRole 动态添加角色
func (p *AgentPool) AddRole(role *roles.Role) error {
	return p.roleMgr.AddRole(role)
}

// UpdateRole 动态更新角色
func (p *AgentPool) UpdateRole(role *roles.Role) error {
	return p.roleMgr.UpdateRole(role)
}

// getPoolKey 获取池的key
func (p *AgentPool) getPoolKey(roleID string, mode aisolo.AgentMode) string {
	return roleID + ":" + mode.String()
}

// Cleanup 清理过期Agent
func (p *AgentPool) Cleanup() {
	// 后续实现过期清理逻辑
}
