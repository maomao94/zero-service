package middleware

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// =============================================================================
// HumanConfirmMiddleware - 人工确认中间件
// =============================================================================

// HumanConfirmMiddleware 人工确认中间件，用于需要用户额外确认的操作
// 与 ApprovalMiddleware 的区别：
// 1. 只询问是否确认，不提供拒绝理由
// 2. 可以提供额外的评论/备注
// 3. 适合需要用户补充信息的场景
type HumanConfirmMiddleware struct {
	// tools 需要确认的工具名称集合
	tools map[string]*ConfirmConfig
}

// ConfirmConfig 确认配置
type ConfirmConfig struct {
	Message string // 确认消息
}

// NewHumanConfirmMiddleware 创建人工确认中间件
func NewHumanConfirmMiddleware() *HumanConfirmMiddleware {
	return &HumanConfirmMiddleware{
		tools: make(map[string]*ConfirmConfig),
	}
}

// WithConfirm 添加确认配置
func (m *HumanConfirmMiddleware) WithConfirm(toolName string, cfg *ConfirmConfig) *HumanConfirmMiddleware {
	m.tools[toolName] = cfg
	return m
}

// WrapInvokableToolCall 包装工具调用（实现 adk.ChatModelAgentMiddleware 接口）
func (m *HumanConfirmMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	cfg, ok := m.tools[tCtx.Name]
	if !ok {
		return endpoint, nil
	}

	return m.wrapConfirm(tCtx.Name, cfg.Message, endpoint), nil
}

// WrapStreamableToolCall 包装流式工具调用（实现 adk.ChatModelAgentMiddleware 接口）
func (m *HumanConfirmMiddleware) WrapStreamableToolCall(
	_ context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	cfg, ok := m.tools[tCtx.Name]
	if !ok {
		return endpoint, nil
	}

	return m.wrapStreamableConfirm(tCtx.Name, cfg.Message, endpoint), nil
}

// wrapConfirm 包装确认类型的工具调用
func (m *HumanConfirmMiddleware) wrapConfirm(
	toolName string,
	message string,
	endpoint adk.InvokableToolCallEndpoint,
) adk.InvokableToolCallEndpoint {
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)

		if !wasInterrupted {
			return "", tool.StatefulInterrupt(ctx, &HumanConfirmInfo{
				ToolName:        toolName,
				ArgumentsInJSON: args,
				Message:         message,
			}, args)
		}

		isTarget, hasData, data := tool.GetResumeContext[*HumanConfirmResult](ctx)
		if isTarget && hasData {
			if data.Confirmed {
				return endpoint(ctx, storedArgs, opts...)
			}
			if data.Comment != nil {
				return fmt.Sprintf("tool '%s' not confirmed: %s", toolName, *data.Comment), nil
			}
			return fmt.Sprintf("tool '%s' not confirmed", toolName), nil
		}

		return "", tool.StatefulInterrupt(ctx, &HumanConfirmInfo{
			ToolName:        toolName,
			ArgumentsInJSON: storedArgs,
			Message:         message,
		}, storedArgs)
	}
}

// wrapStreamableConfirm 包装流式确认类型的工具调用
func (m *HumanConfirmMiddleware) wrapStreamableConfirm(
	toolName string,
	message string,
	endpoint adk.StreamableToolCallEndpoint,
) adk.StreamableToolCallEndpoint {
	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
		if !wasInterrupted {
			return nil, tool.StatefulInterrupt(ctx, &HumanConfirmInfo{
				ToolName:        toolName,
				ArgumentsInJSON: args,
				Message:         message,
			}, args)
		}

		isTarget, hasData, data := tool.GetResumeContext[*HumanConfirmResult](ctx)
		if isTarget && hasData {
			if data.Confirmed {
				return endpoint(ctx, storedArgs, opts...)
			}
			var reason string
			if data.Comment != nil {
				reason = fmt.Sprintf("tool '%s' not confirmed: %s", toolName, *data.Comment)
			} else {
				reason = fmt.Sprintf("tool '%s' not confirmed", toolName)
			}
			return singleChunkReader(reason), nil
		}

		isTarget, _, _ = tool.GetResumeContext[any](ctx)
		if !isTarget {
			return nil, tool.StatefulInterrupt(ctx, &HumanConfirmInfo{
				ToolName:        toolName,
				ArgumentsInJSON: storedArgs,
				Message:         message,
			}, storedArgs)
		}

		return endpoint(ctx, storedArgs, opts...)
	}
}
