package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/zeromicro/go-zero/core/breaker"
	"github.com/zeromicro/go-zero/core/logx"
)

// ToolManager 工具管理器
type ToolManager struct {
	registry   *Registry
	timeout    time.Duration
	maxRetries int
	breaker    breaker.Breaker
	semaphore  chan struct{}
}

// ToolConfig 工具配置
type ToolConfig struct {
	Timeout        time.Duration
	MaxRetries     int
	MaxConcurrency int
}

// NewToolManager 创建工具管理器
func NewToolManager(registry *Registry, config ToolConfig) *ToolManager {
	return &ToolManager{
		registry:   registry,
		timeout:    config.Timeout,
		maxRetries: config.MaxRetries,
		breaker:    breaker.NewBreaker(),
		semaphore:  make(chan struct{}, config.MaxConcurrency),
	}
}

// InvokeTool 调用工具，带超时、重试、熔断
func (m *ToolManager) InvokeTool(ctx context.Context, toolName string, parameters map[string]any) (any, error) {
	var result any
	var err error

	err = m.breaker.Do(func() error {
		m.semaphore <- struct{}{}
		defer func() { <-m.semaphore }()

		// 带超时的上下文
		ctx, cancel := context.WithTimeout(ctx, m.timeout)
		defer cancel()

		t, err := m.registry.GetTool(ctx, toolName)
		if err != nil {
			return err
		}

		// 检查是否实现InvokableTool接口
		invokable, ok := t.(tool.InvokableTool)
		if !ok {
			return fmt.Errorf("tool %s is not invokable", toolName)
		}

		// 把parameters序列化成JSON字符串
		paramsBytes, err := json.Marshal(parameters)
		if err != nil {
			return fmt.Errorf("marshal parameters failed: %w", err)
		}

		// 重试逻辑
		for i := 0; i < m.maxRetries; i++ {
			result, err = invokable.InvokableRun(ctx, string(paramsBytes))
			if err == nil {
				return nil
			}
			logx.Infof("tool %s invoke failed, retry %d/%d: %v", toolName, i+1, m.maxRetries, err)
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
		}
		return err
	})

	return result, err
}

// RegisterTool 注册工具
func (m *ToolManager) RegisterTool(t tool.InvokableTool) error {
	return m.registry.RegisterTool(context.Background(), t)
}

// GetTools 获取所有工具
func (m *ToolManager) GetTools() []tool.BaseTool {
	tools, _ := m.registry.ListTools(context.Background())
	return tools
}
