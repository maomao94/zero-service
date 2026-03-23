package provider

import (
	"fmt"

	"zero-service/aiapp/aichat/internal/config"
)

// ModelMapping 模型 ID 到 provider + 后端模型的映射
type ModelMapping struct {
	ProviderName string
	BackendModel string
}

// Registry provider 注册表
type Registry struct {
	providers map[string]Provider     // key = provider name
	models    map[string]ModelMapping // key = model id
}

// NewRegistry 根据配置构建注册表
func NewRegistry(providers []config.ProviderConfig, models []config.ModelConfig) (*Registry, error) {
	r := &Registry{
		providers: make(map[string]Provider),
		models:    make(map[string]ModelMapping),
	}

	// 初始化 providers
	for _, pc := range providers {
		p, err := newProvider(pc)
		if err != nil {
			return nil, fmt.Errorf("init provider %s: %w", pc.Name, err)
		}
		r.providers[pc.Name] = p
	}

	// 建立模型映射
	for _, mc := range models {
		if _, ok := r.providers[mc.Provider]; !ok {
			return nil, fmt.Errorf("model %s references unknown provider %s", mc.Id, mc.Provider)
		}
		r.models[mc.Id] = ModelMapping{
			ProviderName: mc.Provider,
			BackendModel: mc.BackendModel,
		}
	}

	return r, nil
}

// GetProvider 根据模型 ID 获取 provider 和后端模型名
func (r *Registry) GetProvider(modelId string) (Provider, string, error) {
	mapping, ok := r.models[modelId]
	if !ok {
		return nil, "", fmt.Errorf("model %s not found", modelId)
	}
	p, ok := r.providers[mapping.ProviderName]
	if !ok {
		return nil, "", fmt.Errorf("provider %s not found", mapping.ProviderName)
	}
	return p, mapping.BackendModel, nil
}

// ModelIds 返回所有已注册的模型 ID
func (r *Registry) ModelIds() []string {
	ids := make([]string, 0, len(r.models))
	for id := range r.models {
		ids = append(ids, id)
	}
	return ids
}

// newProvider 根据配置创建 provider 实例
func newProvider(pc config.ProviderConfig) (Provider, error) {
	switch pc.Type {
	case "openai_compatible":
		return NewOpenAICompatible(pc.Endpoint, pc.ApiKey), nil
	default:
		return nil, fmt.Errorf("unknown provider type: %s", pc.Type)
	}
}
