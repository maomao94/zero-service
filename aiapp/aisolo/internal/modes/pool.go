package modes

import (
	"context"
	"fmt"
	"sync"

	"zero-service/aiapp/aisolo/aisolo"
	einoxagent "zero-service/common/einox/agent"
)

// Pool 按 Mode 缓存 Agent 实例。Agent 本身是无状态的 (会话历史 / checkpoint 外置),
// 所以一个 mode 对应一个单例即可复用。
type Pool struct {
	reg  *Registry
	deps Dependencies

	mu    sync.Mutex
	cache map[aisolo.AgentMode]*einoxagent.Agent
}

// NewPool 创建 Agent 缓存池。
func NewPool(reg *Registry, deps Dependencies) *Pool {
	return &Pool{
		reg:   reg,
		deps:  deps,
		cache: make(map[aisolo.AgentMode]*einoxagent.Agent),
	}
}

// Get 返回 (或惰性构造) 给定 mode 的 Agent 实例。
func (p *Pool) Get(ctx context.Context, mode aisolo.AgentMode) (*einoxagent.Agent, error) {
	effMode := mode
	if effMode == aisolo.AgentMode_AGENT_MODE_UNSPECIFIED {
		effMode = p.reg.Default()
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if a, ok := p.cache[effMode]; ok {
		return a, nil
	}

	bp, ok := p.reg.Get(effMode)
	if !ok {
		return nil, fmt.Errorf("modes: no blueprint for mode %v", effMode)
	}

	a, err := bp.Build(ctx, p.deps)
	if err != nil {
		return nil, fmt.Errorf("modes: build mode %v: %w", effMode, err)
	}
	p.cache[effMode] = a
	return a, nil
}

// Invalidate 清空 mode 对应的缓存 (例如 ChatModel 重新加载后调用)。
func (p *Pool) Invalidate(mode aisolo.AgentMode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.cache, mode)
}

// Cleanup 全量清空缓存。
func (p *Pool) Cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, a := range p.cache {
		_ = a.Stop(context.Background())
	}
	p.cache = make(map[aisolo.AgentMode]*einoxagent.Agent)
}
