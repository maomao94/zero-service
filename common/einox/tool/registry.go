package tool

import (
	"fmt"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/zeromicro/go-zero/core/logx"
)

// =============================================================================
// ToolRegistry 工具注册中心
// =============================================================================

// ToolInfo 工具信息
type ToolInfo struct {
	Name        string                 `json:"name"`        // 工具名称（唯一标识）
	Description string                 `json:"description"` // 工具描述
	Category    string                 `json:"category"`    // 工具分类
	Tags        []string               `json:"tags"`        // 标签
	Enabled     bool                   `json:"enabled"`     // 是否启用
	Metadata    map[string]interface{} `json:"metadata"`    // 元数据
}

// ToolRegistry 全局工具注册中心
type ToolRegistry struct {
	tools sync.Map // key: tool name, value: *toolEntry
}

type toolEntry struct {
	tool   tool.BaseTool
	info   ToolInfo
	config *ToolConfig
}

// ToolConfig 工具配置
type ToolConfig struct {
	Enabled        bool     `json:"enabled"`         // 是否启用
	Timeout        int      `json:"timeout"`         // 超时时间（秒）
	RateLimit      int      `json:"rate_limit"`      // 限流（次/分钟）
	MaxConcurrency int      `json:"max_concurrency"` // 最大并发数
	RequiredScopes []string `json:"required_scopes"` // 需要的权限范围
	Whitelist      []string `json:"whitelist"`       // 调用白名单（用户ID列表）
	Blacklist      []string `json:"blacklist"`       // 调用黑名单（用户ID列表）
}

var (
	defaultRegistry *ToolRegistry
	once            sync.Once
)

// DefaultToolRegistry 获取默认工具注册中心
func DefaultToolRegistry() *ToolRegistry {
	once.Do(func() {
		defaultRegistry = &ToolRegistry{}
	})
	return defaultRegistry
}

// NewToolRegistry 创建新的工具注册中心
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{}
}

// Register 注册工具
func (r *ToolRegistry) Register(name string, t tool.BaseTool, info ToolInfo, config *ToolConfig) error {
	if name == "" {
		return fmt.Errorf("tool name is required")
	}
	if t == nil {
		return fmt.Errorf("tool instance is required")
	}

	if config == nil {
		config = &ToolConfig{
			Enabled:        true,
			Timeout:        30,
			RateLimit:      60,
			MaxConcurrency: 10,
		}
	}

	info.Name = name
	info.Enabled = config.Enabled

	entry := &toolEntry{
		tool:   t,
		info:   info,
		config: config,
	}

	r.tools.Store(name, entry)
	logx.Infof("[ToolRegistry] Registered tool: %s", name)
	return nil
}

// Unregister 注销工具
func (r *ToolRegistry) Unregister(name string) {
	if _, ok := r.tools.Load(name); ok {
		r.tools.Delete(name)
		logx.Infof("[ToolRegistry] Unregistered tool: %s", name)
	}
}

// Get 获取工具
func (r *ToolRegistry) Get(name string) (tool.BaseTool, *ToolConfig, error) {
	entryAny, ok := r.tools.Load(name)
	if !ok {
		return nil, nil, fmt.Errorf("tool %s not found", name)
	}

	entry := entryAny.(*toolEntry)
	if !entry.config.Enabled {
		return nil, nil, fmt.Errorf("tool %s is disabled", name)
	}

	return entry.tool, entry.config, nil
}

// GetInfo 获取工具信息
func (r *ToolRegistry) GetInfo(name string) (*ToolInfo, error) {
	entryAny, ok := r.tools.Load(name)
	if !ok {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	entry := entryAny.(*toolEntry)
	info := entry.info
	return &info, nil
}

// ListTools 列出所有工具
func (r *ToolRegistry) ListTools() []ToolInfo {
	var tools []ToolInfo

	r.tools.Range(func(key, value interface{}) bool {
		entry := value.(*toolEntry)
		tools = append(tools, entry.info)
		return true
	})

	return tools
}

// ListEnabledTools 列出所有启用的工具
func (r *ToolRegistry) ListEnabledTools() []tool.BaseTool {
	var enabledTools []tool.BaseTool

	r.tools.Range(func(key, value interface{}) bool {
		entry := value.(*toolEntry)
		if entry.config.Enabled {
			enabledTools = append(enabledTools, entry.tool)
		}
		return true
	})

	return enabledTools
}

// GetByCategory 根据分类获取工具
func (r *ToolRegistry) GetByCategory(category string) []ToolInfo {
	var tools []ToolInfo

	r.tools.Range(func(key, value interface{}) bool {
		entry := value.(*toolEntry)
		if entry.info.Category == category && entry.config.Enabled {
			tools = append(tools, entry.info)
		}
		return true
	})

	return tools
}

// UpdateToolConfig 更新工具配置
func (r *ToolRegistry) UpdateToolConfig(name string, config *ToolConfig) error {
	entryAny, ok := r.tools.Load(name)
	if !ok {
		return fmt.Errorf("tool %s not found", name)
	}

	entry := entryAny.(*toolEntry)
	entry.config = config
	entry.info.Enabled = config.Enabled

	r.tools.Store(name, entry)
	logx.Infof("[ToolRegistry] Updated config for tool: %s", name)
	return nil
}

// EnableTool 启用工具
func (r *ToolRegistry) EnableTool(name string) error {
	return r.updateToolEnabled(name, true)
}

// DisableTool 禁用工具
func (r *ToolRegistry) DisableTool(name string) error {
	return r.updateToolEnabled(name, false)
}

func (r *ToolRegistry) updateToolEnabled(name string, enabled bool) error {
	entryAny, ok := r.tools.Load(name)
	if !ok {
		return fmt.Errorf("tool %s not found", name)
	}

	entry := entryAny.(*toolEntry)
	entry.config.Enabled = enabled
	entry.info.Enabled = enabled

	r.tools.Store(name, entry)

	action := "enabled"
	if !enabled {
		action = "disabled"
	}
	logx.Infof("[ToolRegistry] Tool %s %s", name, action)
	return nil
}

// CheckPermission 检查用户是否有权限调用工具
func (r *ToolRegistry) CheckPermission(name string, userID string, scopes []string) (bool, error) {
	entryAny, ok := r.tools.Load(name)
	if !ok {
		return false, fmt.Errorf("tool %s not found", name)
	}

	entry := entryAny.(*toolEntry)
	config := entry.config

	// 检查黑名单
	for _, blackUser := range config.Blacklist {
		if blackUser == userID {
			return false, fmt.Errorf("user %s is blacklisted for tool %s", userID, name)
		}
	}

	// 检查白名单（如果有配置）
	if len(config.Whitelist) > 0 {
		found := false
		for _, whiteUser := range config.Whitelist {
			if whiteUser == userID {
				found = true
				break
			}
		}
		if !found {
			return false, fmt.Errorf("user %s is not in whitelist for tool %s", userID, name)
		}
	}

	// 检查权限范围
	if len(config.RequiredScopes) > 0 {
		scopeMap := make(map[string]bool)
		for _, s := range scopes {
			scopeMap[s] = true
		}

		for _, required := range config.RequiredScopes {
			if !scopeMap[required] {
				return false, fmt.Errorf("missing required scope %s for tool %s", required, name)
			}
		}
	}

	return true, nil
}

// =============================================================================
// 全局快捷方法
// =============================================================================

// RegisterTool 注册工具到默认注册中心
func RegisterTool(name string, t tool.BaseTool, info ToolInfo, config *ToolConfig) error {
	return DefaultToolRegistry().Register(name, t, info, config)
}

// GetTool 从默认注册中心获取工具
func GetTool(name string) (tool.BaseTool, *ToolConfig, error) {
	return DefaultToolRegistry().Get(name)
}

// ListAllTools 列出默认注册中心所有工具
func ListAllTools() []ToolInfo {
	return DefaultToolRegistry().ListTools()
}

// ListEnabledTools 列出默认注册中心所有启用的工具
func ListEnabledTools() []tool.BaseTool {
	return DefaultToolRegistry().ListEnabledTools()
}
