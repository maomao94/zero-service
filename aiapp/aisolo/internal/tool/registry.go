package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
)

// =============================================================================
// 内置工具信息
// =============================================================================

// ToolInfo 工具信息
type ToolInfo struct {
	Name        string
	Description string
}

// ListBuiltinTools 返回所有内置工具信息
func ListBuiltinTools() []ToolInfo {
	return []ToolInfo{
		{Name: "calculator", Description: "数学计算器，支持加减乘除、幂运算等"},
		{Name: "date_time", Description: "获取当前日期和时间"},
	}
}

// =============================================================================
// Calculator 工具
// =============================================================================

// CalculatorTool 计算器工具
type CalculatorTool struct{}

// NewCalculatorTool 创建计算器工具
func NewCalculatorTool() *CalculatorTool {
	return &CalculatorTool{}
}

// Info 返回工具信息
func (t *CalculatorTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "calculator",
		Desc: "数学计算器，支持加减乘除、幂运算等基本数学运算。输入数学表达式，返回计算结果。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"expression": {
				Type: "string",
				Desc: "数学表达式，如 2+3*4 或 (10+5)/3",
			},
		}),
	}, nil
}

// Invoke 执行计算
func (t *CalculatorTool) Invoke(ctx context.Context, params string) (string, error) {
	var req struct {
		Expression string `json:"expression"`
	}
	if err := json.Unmarshal([]byte(params), &req); err != nil {
		return "", fmt.Errorf("invalid params: %w", err)
	}

	result, err := evalExpression(req.Expression)
	if err != nil {
		return "", fmt.Errorf("calculation error: %w", err)
	}

	return fmt.Sprintf("%.6g", result), nil
}

// evalExpression 计算数学表达式
func evalExpression(expression string) (float64, error) {
	var result float64
	var currentOp byte = '+'
	var num float64

	for i := 0; i < len(expression); i++ {
		c := expression[i]
		if c == ' ' {
			continue
		}
		if c >= '0' && c <= '9' || c == '.' {
			j := i
			for j < len(expression) && ((expression[j] >= '0' && expression[j] <= '9') || expression[j] == '.') {
				j++
			}
			fmt.Sscanf(expression[i:j], "%f", &num)
			i = j - 1
		} else if c == '+' || c == '-' || c == '*' || c == '/' || c == '^' {
			switch currentOp {
			case '+':
				result += num
			case '-':
				result -= num
			case '*':
				result *= num
			case '/':
				if num == 0 {
					return 0, fmt.Errorf("division by zero")
				}
				result /= num
			case '^':
				result = math.Pow(result, num)
			}
			currentOp = c
		}
	}

	switch currentOp {
	case '+':
		result += num
	case '-':
		result -= num
	case '*':
		result *= num
	case '/':
		if num == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		result /= num
	case '^':
		result = math.Pow(result, num)
	}

	return result, nil
}

// =============================================================================
// DateTime 工具
// =============================================================================

// DateTimeTool 日期时间工具
type DateTimeTool struct{}

// NewDateTimeTool 创建日期时间工具
func NewDateTimeTool() *DateTimeTool {
	return &DateTimeTool{}
}

// Info 返回工具信息
func (t *DateTimeTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "date_time",
		Desc:        "获取当前的日期和时间信息，包括年月日时分秒和星期。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

// Invoke 执行获取日期时间
func (t *DateTimeTool) Invoke(ctx context.Context, params string) (string, error) {
	now := time.Now()
	result := map[string]any{
		"date":     now.Format("2006-01-02"),
		"time":     now.Format("15:04:05"),
		"datetime": now.Format("2006-01-02 15:04:05"),
		"weekday":  now.Weekday().String(),
		"unix":     now.Unix(),
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// =============================================================================
// 工具注册中心
// =============================================================================

// ToolFunc 工具函数接口（简化版）
type ToolFunc interface {
	Info(ctx context.Context) (*schema.ToolInfo, error)
	Invoke(ctx context.Context, params string) (string, error)
}

// Registry 工具注册中心
type Registry struct {
	mu    sync.RWMutex
	tools map[string]ToolFunc
}

// NewRegistry 创建工具注册中心
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]ToolFunc),
	}
}

// Register 注册工具
func (r *Registry) Register(name string, t ToolFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[name] = t
	logx.Infof("[ToolRegistry] registered tool: %s", name)
}

// Get 获取工具
func (r *Registry) Get(name string) (ToolFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List 返回所有工具
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// GetTool 实现tool.Registry接口
func (r *Registry) GetTool(ctx context.Context, name string) (tool.BaseTool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool %s not found", name)
	}
	// 适配成tool.BaseTool
	return &toolAdapter{t: t}, nil
}

// ListTools 实现tool.Registry接口
func (r *Registry) ListTools(ctx context.Context) ([]tool.BaseTool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var tools []tool.BaseTool
	for _, t := range r.tools {
		tools = append(tools, &toolAdapter{t: t})
	}
	return tools, nil
}

// Register 实现tool.Registry接口
func (r *Registry) RegisterTool(ctx context.Context, t tool.InvokableTool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	info, err := t.Info(ctx)
	if err != nil {
		return err
	}
	// 适配成ToolFunc
	r.tools[info.Name] = &baseToolAdapter{t: t}
	logx.Infof("[ToolRegistry] registered tool: %s", info.Name)
	return nil
}

// toolAdapter 把ToolFunc适配成tool.InvokableTool
type toolAdapter struct {
	t ToolFunc
}

func (a *toolAdapter) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return a.t.Info(ctx)
}

func (a *toolAdapter) Invoke(ctx context.Context, params string) (string, error) {
	return a.t.Invoke(ctx, params)
}

// baseToolAdapter 把tool.InvokableTool适配成ToolFunc
type baseToolAdapter struct {
	t tool.InvokableTool
}

func (a *baseToolAdapter) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return a.t.Info(ctx)
}

func (a *baseToolAdapter) Invoke(ctx context.Context, params string) (string, error) {
	return a.t.InvokableRun(ctx, params)
}

// RegisterBuiltinTools 注册内置工具
func (r *Registry) RegisterBuiltinTools() {
	r.Register("calculator", NewCalculatorTool())
	r.Register("date_time", NewDateTimeTool())
}

// 全局注册中心
var (
	globalRegistry     *Registry
	globalRegistryOnce sync.Once
)

// GetGlobalRegistry 获取全局工具注册中心
func GetGlobalRegistry() *Registry {
	globalRegistryOnce.Do(func() {
		globalRegistry = NewRegistry()
		globalRegistry.RegisterBuiltinTools()
	})
	return globalRegistry
}
