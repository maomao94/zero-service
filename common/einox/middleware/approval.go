package middleware

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// =============================================================================
// ApprovalMiddleware - 通用审批中间件
// =============================================================================

// ApprovalMiddleware 审批中间件，用于拦截特定工具的调用并等待用户审批
// 支持两种模式：
// 1. 工具级别审批：只审批特定的工具
// 2. 参数级别审批：根据工具参数内容决定是否审批
type ApprovalMiddleware struct {
	// approvalTools 需要审批的工具名称集合
	approvalTools map[string]bool
	// approvalConfig 审批配置（可选的审批问题）
	approvalConfig map[string]*ApprovalConfig
}

// ApprovalConfig 审批配置
type ApprovalConfig struct {
	Question string // 审批问题文本
}

// NewApprovalMiddleware 创建审批中间件
// toolNames: 需要审批的工具名称列表
func NewApprovalMiddleware(toolNames []string) *ApprovalMiddleware {
	approvalMap := make(map[string]bool)
	for _, name := range toolNames {
		approvalMap[name] = true
	}
	return &ApprovalMiddleware{
		approvalTools:  approvalMap,
		approvalConfig: make(map[string]*ApprovalConfig),
	}
}

// WithApprovalConfig 设置审批配置
func (m *ApprovalMiddleware) WithApprovalConfig(toolName string, cfg *ApprovalConfig) *ApprovalMiddleware {
	m.approvalConfig[toolName] = cfg
	return m
}

// WrapInvokableToolCall 包装工具调用（实现 adk.ChatModelAgentMiddleware 接口）
func (m *ApprovalMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	if !m.approvalTools[tCtx.Name] {
		return endpoint, nil
	}

	cfg := m.approvalConfig[tCtx.Name]
	question := "请确认是否执行该操作"
	if cfg != nil && cfg.Question != "" {
		question = cfg.Question
	}

	return m.wrapApproval(tCtx.Name, question, endpoint), nil
}

// WrapStreamableToolCall 包装流式工具调用（实现 adk.ChatModelAgentMiddleware 接口）
func (m *ApprovalMiddleware) WrapStreamableToolCall(
	_ context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	if !m.approvalTools[tCtx.Name] {
		return endpoint, nil
	}

	cfg := m.approvalConfig[tCtx.Name]
	question := "请确认是否执行该操作"
	if cfg != nil && cfg.Question != "" {
		question = cfg.Question
	}

	return m.wrapStreamableApproval(tCtx.Name, question, endpoint), nil
}

// wrapApproval 包装审批类型的工具调用
func (m *ApprovalMiddleware) wrapApproval(
	toolName string,
	question string,
	endpoint adk.InvokableToolCallEndpoint,
) adk.InvokableToolCallEndpoint {
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)

		if !wasInterrupted {
			return "", tool.StatefulInterrupt(ctx, &ApprovalInfo{
				ToolName:        toolName,
				ArgumentsInJSON: args,
				Question:        question,
				Required:        true,
			}, args)
		}

		isTarget, hasData, data := tool.GetResumeContext[*ApprovalResult](ctx)
		if isTarget && hasData {
			if data.Approved {
				return endpoint(ctx, storedArgs, opts...)
			}
			if data.DisapproveReason != nil {
				return fmt.Sprintf("tool '%s' disapproved: %s", toolName, *data.DisapproveReason), nil
			}
			return fmt.Sprintf("tool '%s' disapproved", toolName), nil
		}

		return "", tool.StatefulInterrupt(ctx, &ApprovalInfo{
			ToolName:        toolName,
			ArgumentsInJSON: storedArgs,
			Question:        question,
			Required:        true,
		}, storedArgs)
	}
}

// wrapStreamableApproval 包装流式审批类型的工具调用
func (m *ApprovalMiddleware) wrapStreamableApproval(
	toolName string,
	question string,
	endpoint adk.StreamableToolCallEndpoint,
) adk.StreamableToolCallEndpoint {
	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
		if !wasInterrupted {
			return nil, tool.StatefulInterrupt(ctx, &ApprovalInfo{
				ToolName:        toolName,
				ArgumentsInJSON: args,
				Question:        question,
				Required:        true,
			}, args)
		}

		isTarget, hasData, data := tool.GetResumeContext[*ApprovalResult](ctx)
		if isTarget && hasData {
			if data.Approved {
				return endpoint(ctx, storedArgs, opts...)
			}
			var reason string
			if data.DisapproveReason != nil {
				reason = fmt.Sprintf("tool '%s' disapproved: %s", toolName, *data.DisapproveReason)
			} else {
				reason = fmt.Sprintf("tool '%s' disapproved", toolName)
			}
			return singleChunkReader(reason), nil
		}

		isTarget, _, _ = tool.GetResumeContext[any](ctx)
		if !isTarget {
			return nil, tool.StatefulInterrupt(ctx, &ApprovalInfo{
				ToolName:        toolName,
				ArgumentsInJSON: storedArgs,
				Question:        question,
				Required:        true,
			}, storedArgs)
		}

		return endpoint(ctx, storedArgs, opts...)
	}
}

// =============================================================================
// ApprovableTool - 可审批的工具包装器
// =============================================================================

// ApprovableTool 包装任意工具，添加审批流程
// 适用于单个工具需要审批的场景
type ApprovableTool struct {
	tool.InvokableTool
	Question string
}

// InvokableRun 实现 InvokableTool 接口
func (t *ApprovableTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	toolInfo, err := t.Info(ctx)
	if err != nil {
		return "", err
	}

	wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
	if !wasInterrupted {
		question := t.Question
		if question == "" {
			question = "请确认是否执行该操作"
		}
		return "", tool.StatefulInterrupt(ctx, &ApprovalInfo{
			ToolName:        toolInfo.Name,
			ArgumentsInJSON: args,
			Question:        question,
			Required:        true,
		}, args)
	}

	isTarget, hasData, data := tool.GetResumeContext[*ApprovalResult](ctx)
	if isTarget && hasData {
		if data.Approved {
			return t.InvokableTool.InvokableRun(ctx, storedArgs, opts...)
		}
		if data.DisapproveReason != nil {
			return fmt.Sprintf("tool '%s' disapproved: %s", toolInfo.Name, *data.DisapproveReason), nil
		}
		return fmt.Sprintf("tool '%s' disapproved", toolInfo.Name), nil
	}

	isTarget, _, _ = tool.GetResumeContext[any](ctx)
	if !isTarget {
		question := t.Question
		if question == "" {
			question = "请确认是否执行该操作"
		}
		return "", tool.StatefulInterrupt(ctx, &ApprovalInfo{
			ToolName:        toolInfo.Name,
			ArgumentsInJSON: storedArgs,
			Question:        question,
			Required:        true,
		}, storedArgs)
	}

	return t.InvokableTool.InvokableRun(ctx, storedArgs, opts...)
}

// =============================================================================
// 辅助函数
// =============================================================================

// singleChunkReader 创建单块流读取器
func singleChunkReader(content string) *schema.StreamReader[string] {
	return schema.StreamReaderFromArray([]string{content})
}
