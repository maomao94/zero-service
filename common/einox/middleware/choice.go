package middleware

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// =============================================================================
// ChoiceMiddleware - 通用选择中间件（单选/多选）
// =============================================================================

// ChoiceMiddleware 选择中间件，用于拦截工具调用并让用户选择选项
// 支持：
// 1. 单选：用户从多个选项中选择一个
// 2. 多选：用户从多个选项中选择多个
type ChoiceMiddleware struct {
	// singleChoiceTools 单选配置
	singleChoiceTools map[string]*SingleChoiceConfig
	// multipleChoiceTools 多选配置
	multipleChoiceTools map[string]*MultipleChoiceConfig
}

// SingleChoiceConfig 单选配置
type SingleChoiceConfig struct {
	Question string         // 问题文本
	Options  []ChoiceOption // 选项列表
	Required bool           // 是否必须选择
}

// MultipleChoiceConfig 多选配置
type MultipleChoiceConfig struct {
	Question  string         // 问题文本
	Options   []ChoiceOption // 选项列表
	MinSelect int            // 最少选择数量
	MaxSelect int            // 最多选择数量
	Required  bool           // 是否必须选择
}

// NewChoiceMiddleware 创建选择中间件
func NewChoiceMiddleware() *ChoiceMiddleware {
	return &ChoiceMiddleware{
		singleChoiceTools:   make(map[string]*SingleChoiceConfig),
		multipleChoiceTools: make(map[string]*MultipleChoiceConfig),
	}
}

// WithSingleChoice 添加单选配置
func (m *ChoiceMiddleware) WithSingleChoice(toolName string, cfg *SingleChoiceConfig) *ChoiceMiddleware {
	m.singleChoiceTools[toolName] = cfg
	return m
}

// WithMultipleChoice 添加多选配置
func (m *ChoiceMiddleware) WithMultipleChoice(toolName string, cfg *MultipleChoiceConfig) *ChoiceMiddleware {
	m.multipleChoiceTools[toolName] = cfg
	return m
}

// WrapInvokableToolCall 包装工具调用（实现 adk.ChatModelAgentMiddleware 接口）
func (m *ChoiceMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	switch {
	case m.singleChoiceTools[tCtx.Name] != nil:
		return m.wrapSingleChoice(tCtx.Name, m.singleChoiceTools[tCtx.Name], endpoint), nil
	case m.multipleChoiceTools[tCtx.Name] != nil:
		return m.wrapMultipleChoice(tCtx.Name, m.multipleChoiceTools[tCtx.Name], endpoint), nil
	default:
		return endpoint, nil
	}
}

// WrapStreamableToolCall 包装流式工具调用（实现 adk.ChatModelAgentMiddleware 接口）
func (m *ChoiceMiddleware) WrapStreamableToolCall(
	_ context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	switch {
	case m.singleChoiceTools[tCtx.Name] != nil:
		return m.wrapStreamableSingleChoice(tCtx.Name, m.singleChoiceTools[tCtx.Name], endpoint), nil
	case m.multipleChoiceTools[tCtx.Name] != nil:
		return m.wrapStreamableMultipleChoice(tCtx.Name, m.multipleChoiceTools[tCtx.Name], endpoint), nil
	default:
		return endpoint, nil
	}
}

// wrapSingleChoice 包装单选类型的工具调用
func (m *ChoiceMiddleware) wrapSingleChoice(
	toolName string,
	cfg *SingleChoiceConfig,
	endpoint adk.InvokableToolCallEndpoint,
) adk.InvokableToolCallEndpoint {
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)

		if !wasInterrupted {
			return "", tool.StatefulInterrupt(ctx, &ChoiceInfo{
				ToolName:        toolName,
				ArgumentsInJSON: args,
				Question:        cfg.Question,
				Options:         cfg.Options,
				Required:        cfg.Required,
				MaxSelect:       1,
			}, args)
		}

		isTarget, hasData, data := tool.GetResumeContext[*ChoiceResult](ctx)
		if isTarget && hasData {
			if len(data.SelectedIDs) > 0 {
				modifiedArgs := m.applyChoiceToArgs(storedArgs, data.SelectedIDs)
				return endpoint(ctx, modifiedArgs, opts...)
			}
			if cfg.Required {
				return "user did not select any option (required)", nil
			}
			return "user did not select any option", nil
		}

		return "", tool.StatefulInterrupt(ctx, &ChoiceInfo{
			ToolName:        toolName,
			ArgumentsInJSON: storedArgs,
			Question:        cfg.Question,
			Options:         cfg.Options,
			Required:        cfg.Required,
			MaxSelect:       1,
		}, storedArgs)
	}
}

// wrapMultipleChoice 包装多选类型的工具调用
func (m *ChoiceMiddleware) wrapMultipleChoice(
	toolName string,
	cfg *MultipleChoiceConfig,
	endpoint adk.InvokableToolCallEndpoint,
) adk.InvokableToolCallEndpoint {
	maxSelect := cfg.MaxSelect
	if maxSelect == 0 {
		maxSelect = len(cfg.Options)
	}

	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)

		if !wasInterrupted {
			return "", tool.StatefulInterrupt(ctx, &ChoiceInfo{
				ToolName:        toolName,
				ArgumentsInJSON: args,
				Question:        cfg.Question,
				Options:         cfg.Options,
				Required:        cfg.Required,
				MinSelect:       cfg.MinSelect,
				MaxSelect:       maxSelect,
			}, args)
		}

		isTarget, hasData, data := tool.GetResumeContext[*ChoiceResult](ctx)
		if isTarget && hasData {
			if len(data.SelectedIDs) >= cfg.MinSelect {
				modifiedArgs := m.applyChoiceToArgs(storedArgs, data.SelectedIDs)
				return endpoint(ctx, modifiedArgs, opts...)
			}
			if cfg.Required {
				return fmt.Sprintf("user selected %d options, minimum required: %d", len(data.SelectedIDs), cfg.MinSelect), nil
			}
			return "user did not select enough options", nil
		}

		return "", tool.StatefulInterrupt(ctx, &ChoiceInfo{
			ToolName:        toolName,
			ArgumentsInJSON: storedArgs,
			Question:        cfg.Question,
			Options:         cfg.Options,
			Required:        cfg.Required,
			MinSelect:       cfg.MinSelect,
			MaxSelect:       maxSelect,
		}, storedArgs)
	}
}

// wrapStreamableSingleChoice 包装流式单选类型的工具调用
func (m *ChoiceMiddleware) wrapStreamableSingleChoice(
	toolName string,
	cfg *SingleChoiceConfig,
	endpoint adk.StreamableToolCallEndpoint,
) adk.StreamableToolCallEndpoint {
	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
		if !wasInterrupted {
			return nil, tool.StatefulInterrupt(ctx, &ChoiceInfo{
				ToolName:        toolName,
				ArgumentsInJSON: args,
				Question:        cfg.Question,
				Options:         cfg.Options,
				Required:        cfg.Required,
				MaxSelect:       1,
			}, args)
		}

		isTarget, hasData, data := tool.GetResumeContext[*ChoiceResult](ctx)
		if isTarget && hasData {
			if len(data.SelectedIDs) > 0 {
				modifiedArgs := m.applyChoiceToArgs(storedArgs, data.SelectedIDs)
				return endpoint(ctx, modifiedArgs, opts...)
			}
			if cfg.Required {
				return singleChunkReader("user did not select any option (required)"), nil
			}
			return singleChunkReader("user did not select any option"), nil
		}

		isTarget, _, _ = tool.GetResumeContext[any](ctx)
		if !isTarget {
			return nil, tool.StatefulInterrupt(ctx, &ChoiceInfo{
				ToolName:        toolName,
				ArgumentsInJSON: storedArgs,
				Question:        cfg.Question,
				Options:         cfg.Options,
				Required:        cfg.Required,
				MaxSelect:       1,
			}, storedArgs)
		}

		return endpoint(ctx, storedArgs, opts...)
	}
}

// wrapStreamableMultipleChoice 包装流式多选类型的工具调用
func (m *ChoiceMiddleware) wrapStreamableMultipleChoice(
	toolName string,
	cfg *MultipleChoiceConfig,
	endpoint adk.StreamableToolCallEndpoint,
) adk.StreamableToolCallEndpoint {
	maxSelect := cfg.MaxSelect
	if maxSelect == 0 {
		maxSelect = len(cfg.Options)
	}

	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
		if !wasInterrupted {
			return nil, tool.StatefulInterrupt(ctx, &ChoiceInfo{
				ToolName:        toolName,
				ArgumentsInJSON: args,
				Question:        cfg.Question,
				Options:         cfg.Options,
				Required:        cfg.Required,
				MinSelect:       cfg.MinSelect,
				MaxSelect:       maxSelect,
			}, args)
		}

		isTarget, hasData, data := tool.GetResumeContext[*ChoiceResult](ctx)
		if isTarget && hasData {
			if len(data.SelectedIDs) >= cfg.MinSelect {
				modifiedArgs := m.applyChoiceToArgs(storedArgs, data.SelectedIDs)
				return endpoint(ctx, modifiedArgs, opts...)
			}
			if cfg.Required {
				return singleChunkReader(fmt.Sprintf("user selected %d options, minimum required: %d", len(data.SelectedIDs), cfg.MinSelect)), nil
			}
			return singleChunkReader("user did not select enough options"), nil
		}

		isTarget, _, _ = tool.GetResumeContext[any](ctx)
		if !isTarget {
			return nil, tool.StatefulInterrupt(ctx, &ChoiceInfo{
				ToolName:        toolName,
				ArgumentsInJSON: storedArgs,
				Question:        cfg.Question,
				Options:         cfg.Options,
				Required:        cfg.Required,
				MinSelect:       cfg.MinSelect,
				MaxSelect:       maxSelect,
			}, storedArgs)
		}

		return endpoint(ctx, storedArgs, opts...)
	}
}

// applyChoiceToArgs 将用户选择的选项应用到工具参数
func (m *ChoiceMiddleware) applyChoiceToArgs(args string, selectedIDs []string) string {
	var argsMap map[string]interface{}
	if err := json.Unmarshal([]byte(args), &argsMap); err != nil {
		return fmt.Sprintf(`{"selectedOptions": %s, "originalArgs": %s}`,
			marshalToJSON(selectedIDs), args)
	}

	argsMap["selectedOptions"] = selectedIDs

	result, err := json.Marshal(argsMap)
	if err != nil {
		return args
	}

	return string(result)
}

// marshalToJSON 将切片序列化为 JSON 字符串
func marshalToJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(data)
}
