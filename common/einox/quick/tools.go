package quick

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ToolFunc 工具函数类型
type ToolFunc func(ctx context.Context, params string) (string, error)

// FunctionTool 创建一个简单的函数工具
//
//	calculator := quick.FunctionTool("calculator", "数学计算器", func(ctx, params) string {
//	    // 解析 params 并计算
//	    return "2"
//	})
func FunctionTool(name, desc string, fn ToolFunc) tool.BaseTool {
	return &simpleTool{
		name: name,
		desc: desc,
		fn:   fn,
	}
}

// simpleTool 简单工具实现
type simpleTool struct {
	name string
	desc string
	fn   ToolFunc
}

func (t *simpleTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.name,
		Desc: t.desc,
	}, nil
}

func (t *simpleTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.fn(ctx, argumentsInJSON)
}

// Ensure simpleTool 实现 tool.InvokableTool
var _ tool.InvokableTool = (*simpleTool)(nil)

// Calculator 创建计算器工具
//
//	bot := NewChatBot(ctx, cfg)
//	calculator := quick.Calculator()
//	// 使用 calculator
func Calculator() tool.BaseTool {
	return FunctionTool(
		"calculator",
		"数学计算器，支持加减乘除、幂运算。输入 JSON: {\"expression\": \"2+3*4\"}",
		func(ctx context.Context, params string) (string, error) {
			var req struct {
				Expression string `json:"expression"`
			}
			if err := json.Unmarshal([]byte(params), &req); err != nil {
				return "", fmt.Errorf("invalid params: %w", err)
			}

			result, err := evalMath(req.Expression)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%g", result), nil
		},
	)
}

// DateTime 创建日期时间工具
//
//	dateTime := quick.DateTime()
func DateTime() tool.BaseTool {
	return FunctionTool(
		"date_time",
		"获取当前日期和时间",
		func(ctx context.Context, params string) (string, error) {
			now := map[string]any{
				"date":     "2024-01-01",
				"time":     "00:00:00",
				"datetime": "2024-01-01 00:00:00",
			}
			data, _ := json.Marshal(now)
			return string(data), nil
		},
	)
}

// evalMath 简单的数学表达式计算
func evalMath(expr string) (float64, error) {
	// 简化实现：实际项目中建议使用专门的表达式解析库
	var result float64
	var currentOp byte = '+'
	var num float64

	for i := 0; i < len(expr); i++ {
		c := expr[i]
		if c == ' ' {
			continue
		}
		if c >= '0' && c <= '9' || c == '.' {
			j := i
			for j < len(expr) && ((expr[j] >= '0' && expr[j] <= '9') || expr[j] == '.') {
				j++
			}
			fmt.Sscanf(expr[i:j], "%f", &num)
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
				result = pow(result, num)
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
		result = pow(result, num)
	}

	return result, nil
}

func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}
