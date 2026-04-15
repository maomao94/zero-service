package middleware

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"zero-service/common/einox"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// ChoiceType 选项类型
type ChoiceType string

const (
	ChoiceTypeSingle ChoiceType = "single" // 单选
	ChoiceTypeMulti  ChoiceType = "multi"  // 多选
)

// ChoiceOption 用户选项
type ChoiceOption struct {
	Value string `json:"value"` // 选项值
	Label string `json:"label"` // 选项显示文本
	Desc  string `json:"desc"`  // 选项描述
}

// ChoiceConfig 选项配置
type ChoiceConfig struct {
	ID          string         `json:"id"`           // 选项ID
	Type        ChoiceType     `json:"type"`         // 类型：单选/多选
	Title       string         `json:"title"`        // 选择标题
	Description string         `json:"description"`  // 选择描述
	Options     []ChoiceOption `json:"options"`      // 选项列表
	Required    bool           `json:"required"`     // 是否必填
	MultiMax    int            `json:"multi_max"`    // 多选最大数量，默认等于选项总数
	CustomInput bool           `json:"custom_input"` // 是否允许自定义输入
	CustomHint  string         `json:"custom_hint"`  // 自定义输入提示
}

// ChoiceResult 用户选择结果
type ChoiceResult struct {
	ChoiceID    string   `json:"choice_id"`    // 选项ID
	Selected    []string `json:"selected"`     // 选中的选项值列表
	CustomInput string   `json:"custom_input"` // 自定义输入内容
	Confirmed   bool     `json:"confirmed"`    // 用户是否确认
}

// ChoiceMiddleware 用户选项选择中间件
type ChoiceMiddleware struct {
	adk.BaseChatModelAgentMiddleware
	pendingChoices map[string]*ChoiceConfig      // 待处理的用户选择：sessionID -> config
	resultChan     map[string]chan *ChoiceResult // 结果通道：sessionID -> chan
}

// NewChoiceMiddleware 创建用户选择中间件
func NewChoiceMiddleware() *ChoiceMiddleware {
	return &ChoiceMiddleware{
		pendingChoices: make(map[string]*ChoiceConfig),
		resultChan:     make(map[string]chan *ChoiceResult),
	}
}

// PromptChoice 提示用户进行选择
func (m *ChoiceMiddleware) PromptChoice(ctx context.Context, sessionID string, config ChoiceConfig) (*ChoiceResult, error) {
	if config.ID == "" {
		config.ID = fmt.Sprintf("choice_%d", time.Now().UnixNano())
	}
	if config.Type == "" {
		config.Type = ChoiceTypeSingle
	}
	if config.MultiMax <= 0 && config.Type == ChoiceTypeMulti {
		config.MultiMax = len(config.Options)
	}

	// 保存待处理选择
	m.pendingChoices[sessionID] = &config

	// 创建结果通道
	resultChan := make(chan *ChoiceResult, 1)
	m.resultChan[sessionID] = resultChan

	// 生成选择提示文本
	prompt := m.generateChoicePrompt(&config)

	// TODO: 发送提示给用户（这里需要根据实际交互方式实现发送逻辑）
	fmt.Printf("提示用户选择: %s\n", prompt)

	// 等待用户输入结果
	select {
	case result := <-resultChan:
		// 清理临时数据
		delete(m.pendingChoices, sessionID)
		delete(m.resultChan, sessionID)
		return result, nil
	case <-ctx.Done():
		// 上下文取消，清理数据
		delete(m.pendingChoices, sessionID)
		delete(m.resultChan, sessionID)
		return nil, ctx.Err()
	}
}

// SubmitChoiceResult 提交用户选择结果
func (m *ChoiceMiddleware) SubmitChoiceResult(sessionID string, result *ChoiceResult) error {
	// 检查是否存在待处理选择
	config, ok := m.pendingChoices[sessionID]
	if !ok {
		return fmt.Errorf("no pending choice for session %s", sessionID)
	}

	// 验证选择结果合法性
	if err := m.validateChoiceResult(config, result); err != nil {
		return err
	}

	// 发送结果到等待通道
	if ch, ok := m.resultChan[sessionID]; ok {
		ch <- result
	}

	return nil
}

// generateChoicePrompt 生成选择提示文本
func (m *ChoiceMiddleware) generateChoicePrompt(config *ChoiceConfig) string {
	var sb strings.Builder

	// 标题和描述
	sb.WriteString(fmt.Sprintf("### %s\n\n", config.Title))
	if config.Description != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", config.Description))
	}

	// 选项列表
	sb.WriteString("请选择：\n")
	for i, opt := range config.Options {
		sb.WriteString(fmt.Sprintf("%d. %s", i+1, opt.Label))
		if opt.Desc != "" {
			sb.WriteString(fmt.Sprintf("：%s", opt.Desc))
		}
		sb.WriteString("\n")
	}

	// 选择提示
	if config.Type == ChoiceTypeSingle {
		sb.WriteString("\n请输入选项序号：")
	} else {
		sb.WriteString(fmt.Sprintf("\n请输入选项序号，多选用逗号分隔（最多选%d个）：", config.MultiMax))
	}

	// 自定义输入提示
	if config.CustomInput {
		hint := "自定义内容"
		if config.CustomHint != "" {
			hint = config.CustomHint
		}
		sb.WriteString(fmt.Sprintf("\n或者直接输入%s：", hint))
	}

	return sb.String()
}

// validateChoiceResult 验证选择结果合法性
func (m *ChoiceMiddleware) validateChoiceResult(config *ChoiceConfig, result *ChoiceResult) error {
	// 检查是否确认
	if !result.Confirmed {
		return fmt.Errorf("choice not confirmed")
	}

	// 检查必填
	if config.Required && len(result.Selected) == 0 && result.CustomInput == "" {
		return fmt.Errorf("choice is required")
	}

	// 如果没有选择，也没有自定义输入，直接返回（非必填情况）
	if len(result.Selected) == 0 && result.CustomInput == "" {
		return nil
	}

	// 验证选择值是否合法
	validValues := make(map[string]bool)
	for _, opt := range config.Options {
		validValues[opt.Value] = true
	}

	for _, val := range result.Selected {
		if !validValues[val] {
			return fmt.Errorf("invalid choice value: %s", val)
		}
	}

	// 验证多选数量
	if config.Type == ChoiceTypeMulti && len(result.Selected) > config.MultiMax {
		return fmt.Errorf("too many choices selected, max %d", config.MultiMax)
	}

	// 验证自定义输入是否允许
	if result.CustomInput != "" && !config.CustomInput {
		return fmt.Errorf("custom input is not allowed")
	}

	return nil
}

// HandleMessage 处理用户消息，用于自动匹配选择结果
func (m *ChoiceMiddleware) HandleMessage(ctx context.Context, sessionID string, message string) (bool, error) {
	// 检查是否有待处理选择
	config, ok := m.pendingChoices[sessionID]
	if !ok {
		return false, nil
	}

	// 解析用户输入
	result := &ChoiceResult{
		ChoiceID:  config.ID,
		Confirmed: true,
	}

	// 检查是否是自定义输入
	// 如果用户输入不是数字序号，视为自定义输入
	fields := strings.FieldsFunc(message, func(r rune) bool {
		return r == ',' || r == '，' || r == ' ' || r == '、'
	})

	var selectedIndices []int
	hasInvalid := false

	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		idx, err := strconv.Atoi(f)
		if err != nil || idx < 1 || idx > len(config.Options) {
			hasInvalid = true
			break
		}
		selectedIndices = append(selectedIndices, idx-1)
	}

	if hasInvalid || len(selectedIndices) == 0 {
		// 视为自定义输入
		if config.CustomInput {
			result.CustomInput = message
		} else {
			// 不允许自定义输入，返回错误，要求重新输入
			return true, fmt.Errorf("invalid input, please enter valid option number(s)")
		}
	} else {
		// 解析选择值
		for _, idx := range selectedIndices {
			result.Selected = append(result.Selected, config.Options[idx].Value)
		}
	}

	// 提交结果
	if err := m.SubmitChoiceResult(sessionID, result); err != nil {
		return true, err
	}

	return true, nil
}

// BeforeModelRewriteState 实现ChatModelAgentMiddleware接口，在模型调用前处理用户选择
func (m *ChoiceMiddleware) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	// 检查是否有未处理选择
	sessionID := einox.GetSessionID(ctx)
	if sessionID != "" {
		if _, ok := m.pendingChoices[sessionID]; ok {
			// 有未处理选择，取最新的用户消息
			if len(state.Messages) == 0 {
				return ctx, state, nil
			}
			lastMsg := state.Messages[len(state.Messages)-1]
			if lastMsg.Role != schema.User {
				return ctx, state, nil
			}

			// 处理用户输入
			handled, err := m.HandleMessage(ctx, sessionID, lastMsg.Content)
			if handled {
				if err != nil {
					// 处理错误，返回错误提示给用户，中断模型调用
					state.Messages = append(state.Messages, schema.AssistantMessage(err.Error(), nil))
					return ctx, state, fmt.Errorf("choice handling error: %w", err)
				}
				// 选择已处理，继续执行后续逻辑
				return ctx, state, nil
			}
		}
	}

	// 没有未处理选择，继续执行
	return ctx, state, nil
}
