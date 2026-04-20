// Package builtin 提供 einox 内置工具集。
//
// 文件分工：
//   - compute.go : 纯计算, 无副作用 (echo, calculator, math_*)
//   - io.go      : 带外部副作用 (时间、随机数、http_get 等)
//   - human.go   : 人机交互, 通过 Interrupt / Resume 让用户介入
package builtin

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// =============================================================================
// echo —— 原样回显，用于链路自检
// =============================================================================

type echoParam struct {
	Text string `json:"text" jsonschema:"required,description=要回显的文本"`
}

type echoResult struct {
	Result string `json:"result"`
}

// NewEcho 返回一个示例工具：原样返回输入。
func NewEcho() tool.InvokableTool {
	t, err := utils.InferTool("echo", "Echo: 原样返回输入的文本，常用于自检工具链路。",
		func(_ context.Context, in *echoParam) (*echoResult, error) {
			return &echoResult{Result: in.Text}, nil
		})
	if err != nil {
		panic(err)
	}
	return t
}

// =============================================================================
// calculator —— 只支持四则运算 + 括号
// =============================================================================

type calculatorParam struct {
	Expr string `json:"expr" jsonschema:"required,description=纯数字四则表达式，例如 '1+2*3' 或 '(4+5)/3'。仅允许数字、小数点、+ - * / 与括号；不得含字母或其它符号。若表达式不合法，工具仍会成功返回，请在返回 JSON 中读取 error 字段（若有）并向用户说明，不要假定一定有 result"`
}

type calculatorResult struct {
	Result string `json:"result,omitempty" jsonschema:"description=求值成功时的数字字符串"`
	Error  string `json:"error,omitempty" jsonschema:"description=表达式非法或无法求值时的说明; 若存在此字段应忽略 result 并向用户解释原因"`
}

// NewCalculator 返回一个简单计算器工具。
func NewCalculator() tool.InvokableTool {
	t, err := utils.InferTool("calculator", "Calculator: 对纯数字四则表达式求值（+ - * / 与括号）。重要：非法或无法解析的表达式不会使本工具报错中断；返回体中会出现 error 字段说明原因，此时无有效 result。每次调用后必须先检查返回 JSON：若 error 非空，向用户解释失败原因；仅当 error 为空时使用 result。",
		func(_ context.Context, in *calculatorParam) (*calculatorResult, error) {
			v, err := evalArith(in.Expr)
			if err != nil {
				// 返回 (result, nil) 而非 (_, err)：否则 ADK 会把工具失败当成 NodeRunError，整轮流中断。
				return &calculatorResult{Error: err.Error()}, nil
			}
			return &calculatorResult{Result: strconv.FormatFloat(v, 'f', -1, 64)}, nil
		})
	if err != nil {
		panic(err)
	}
	return t
}

// evalArith 递归下降求值: expression -> term {('+'|'-') term}
func evalArith(s string) (float64, error) {
	p := &arithParser{src: strings.ReplaceAll(s, " ", "")}
	v, err := p.expression()
	if err != nil {
		return 0, err
	}
	if p.pos != len(p.src) {
		return 0, fmt.Errorf("calculator: unexpected char %q at %d", p.src[p.pos], p.pos)
	}
	return v, nil
}

type arithParser struct {
	src string
	pos int
}

func (p *arithParser) peek() byte {
	if p.pos >= len(p.src) {
		return 0
	}
	return p.src[p.pos]
}

func (p *arithParser) expression() (float64, error) {
	v, err := p.term()
	if err != nil {
		return 0, err
	}
	for {
		c := p.peek()
		if c != '+' && c != '-' {
			return v, nil
		}
		p.pos++
		rhs, err := p.term()
		if err != nil {
			return 0, err
		}
		if c == '+' {
			v += rhs
		} else {
			v -= rhs
		}
	}
}

func (p *arithParser) term() (float64, error) {
	v, err := p.factor()
	if err != nil {
		return 0, err
	}
	for {
		c := p.peek()
		if c != '*' && c != '/' {
			return v, nil
		}
		p.pos++
		rhs, err := p.factor()
		if err != nil {
			return 0, err
		}
		if c == '*' {
			v *= rhs
		} else {
			if rhs == 0 {
				return 0, fmt.Errorf("calculator: div by zero")
			}
			v /= rhs
		}
	}
}

func (p *arithParser) factor() (float64, error) {
	if p.peek() == '(' {
		p.pos++
		v, err := p.expression()
		if err != nil {
			return 0, err
		}
		if p.peek() != ')' {
			return 0, fmt.Errorf("calculator: missing ')'")
		}
		p.pos++
		return v, nil
	}
	if p.peek() == '-' {
		p.pos++
		v, err := p.factor()
		if err != nil {
			return 0, err
		}
		return -v, nil
	}

	start := p.pos
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if (c >= '0' && c <= '9') || c == '.' {
			p.pos++
			continue
		}
		break
	}
	if start == p.pos {
		return 0, fmt.Errorf("calculator: expected number at %d", p.pos)
	}
	return strconv.ParseFloat(p.src[start:p.pos], 64)
}
