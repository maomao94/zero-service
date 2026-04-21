package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"zero-service/common/tool"

	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aichat/internal/provider"
	"zero-service/aiapp/aichat/internal/svc"
	"zero-service/common/antsx"
	"zero-service/common/mcpx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToolEvent 工具状态事件
type ToolEvent struct {
	Type          string `json:"type"`      // tool_start/progress/success/error
	ToolId        string `json:"tool_id"`   // 工具调用 ID
	ToolName      string `json:"tool_name"` // 工具名称
	Index         int    `json:"index"`     // 工具序号（1-based）
	Progress      int    `json:"progress,omitempty"`
	Total         int    `json:"total,omitempty"`
	Message       string `json:"message,omitempty"`
	ResultSummary string `json:"result_summary,omitempty"`
	Error         string `json:"error,omitempty"`
	DurationMs    int64  `json:"duration_ms,omitempty"`
}

// sendToolEvent 发送工具状态事件（通过 role=tool 的 content 字段）
func sendToolEvent(stream aichat.AiChat_ChatCompletionStreamServer, modelId string, event ToolEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return stream.Send(&aichat.ChatCompletionStreamChunk{
		Model: modelId,
		Choices: []*aichat.ChunkChoicePb{
			{
				Delta: &aichat.ChatDeltaPb{
					Role:    "tool",
					Content: string(data),
				},
			},
		},
	})
}

type ChatCompletionStreamLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewChatCompletionStreamLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatCompletionStreamLogic {
	return &ChatCompletionStreamLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ChatCompletionStreamLogic) ChatCompletionStream(in *aichat.ChatCompletionReq, stream aichat.AiChat_ChatCompletionStreamServer) error {
	p, backendModel, providerName, err := l.svcCtx.Registry.GetProvider(in.Model)
	if err != nil {
		return status.Errorf(codes.NotFound, "model %s not found", in.Model)
	}

	req := toProviderRequest(in, backendModel, providerName)

	// 注入 MCP 工具
	var hasTools bool
	if l.svcCtx.McpClient != nil && l.svcCtx.McpClient.HasTools() {
		req.Tools = provider.McpToolsToOpenAI(l.svcCtx.McpClient.Tools())
		hasTools = len(req.Tools) > 0
	}

	l.Logger.Infof("chat completion stream, model: %s -> %s (%s), tools: %v", in.Model, backendModel, providerName, hasTools)

	// 上下文大小检查（软限制）
	if maxTokens := l.svcCtx.Config.MaxContextTokens; maxTokens > 0 {
		var totalTokens int
		for _, msg := range req.Messages {
			totalTokens += 4 + tool.EstimateTokens(msg.Content)
		}
		if totalTokens > maxTokens {
			l.Logger.Infof("context large: %d tokens > %d limit, may cause timeout", totalTokens, maxTokens)
		} else {
			l.Logger.Debugf("context tokens: %d / %d", totalTokens, maxTokens)
		}
	}

	// 总超时
	streamCtx, streamCancel := context.WithTimeout(l.ctx, l.svcCtx.Config.StreamTimeout)
	defer streamCancel()

	// 混合流主循环
	return l.mixedStreamLoop(streamCtx, req, p, stream, in.Model)
}

// mixedStreamLoop 混合流主循环：LLM token + tool 进度全部通过同一个 stream 输出
func (l *ChatCompletionStreamLogic) mixedStreamLoop(ctx context.Context, req *provider.ChatRequest, p provider.Provider, stream aichat.AiChat_ChatCompletionStreamServer, modelId string) error {
	idleTimeout := l.svcCtx.Config.StreamIdleTimeout
	toolBuf := provider.NewToolCallBuffer()

	for round := 0; round < l.svcCtx.Config.MaxToolRounds; round++ {
		// 开始新一轮 LLM 流式
		reader, err := p.ChatCompletionStream(ctx, req)
		if err != nil {
			return provider.ToGrpcError(err)
		}

		for {
			recv := antsx.NewPromise[*provider.StreamChunk]()
			threading.GoSafe(func() {
				chunk, recvErr := reader.Recv()
				if recvErr != nil {
					recv.Reject(recvErr)
				} else {
					recv.Resolve(chunk)
				}
			})
			idleCtx, idleCancel := context.WithTimeout(ctx, idleTimeout)
			chunk, awaitErr := recv.Await(idleCtx)
			idleCancel()

			if awaitErr != nil {
				reader.Close()
				if errors.Is(awaitErr, io.EOF) {
					goto checkToolCalls
				}
				return l.handleStreamError(awaitErr, ctx)
			}

			// 客户端断开检查
			if l.ctx.Err() != nil {
				reader.Close()
				return nil
			}

			// 空 chunk 检查
			if len(chunk.Choices) == 0 {
				continue
			}
			choice := chunk.Choices[0]

			// 收集 tool_calls 增量（需要按 id 拼接）
			if len(choice.Delta.ToolCalls) > 0 {
				l.Logger.Debugf("LLM choose tool_calls: %v", choice.Delta.ToolCalls)
				for i := range choice.Delta.ToolCalls {
					toolBuf.Accumulate(&choice.Delta.ToolCalls[i])
				}
				continue
			}

			// 正常 token → 直接发给前端
			protoChunk := toProtoStreamChunk(chunk, modelId)
			if err := stream.Send(protoChunk); err != nil {
				reader.Close()
				return err
			}
		}

	checkToolCalls:
		reader.Close()

		// 检查是否有累积的 tool_calls
		if !toolBuf.HasPendingTools() {
			return nil
		}

		toolCalls := toolBuf.Collect()
		toolBuf = provider.NewToolCallBuffer()

		l.Logger.Infof("round %d: detected %d tool calls", round+1, len(toolCalls))

		// 把 assistant tool call 写入上下文（需要转换为值类型）
		tcValues := make([]provider.ToolCall, len(toolCalls))
		for i, tc := range toolCalls {
			tcValues[i] = *tc
		}
		req.Messages = append(req.Messages, provider.ChatMessage{
			Role:      "assistant",
			Content:   "",
			ToolCalls: tcValues,
		})

		// 串行执行工具（每个工具都实时推送进度到 stream）
		toolIndex := 1
		for _, tc := range toolCalls {
			toolName := tc.Function.Name
			toolId := tc.Id

			// === 1. 发送工具开始事件 ===
			if err := sendToolEvent(stream, modelId, ToolEvent{
				Type:     "tool_start",
				ToolId:   toolId,
				ToolName: toolName,
				Index:    toolIndex,
			}); err != nil {
				return err
			}

			// === 2. 进度回调 ===
			progressCallback := func(info *mcpx.ProgressInfo) {
				l.Logger.Debugf("tool progress: tool=%s, percent=%d%%, msg=%s", toolName, info.Percent(), info.Message)
				sendToolEvent(stream, modelId, ToolEvent{
					Type:     "tool_progress",
					ToolId:   toolId,
					ToolName: toolName,
					Progress: info.Percent(),
					Total:    100,
					Message:  info.Message,
				})
			}

			// === 3. 提交到 Reactor 池执行，Promise 等待结果 ===
			taskID, promise, submitErr := l.svcCtx.McpClient.CallToolAsyncAwait(ctx, &mcpx.CallToolAsyncRequest{
				Name:         toolName,
				Args:         mcpx.ParseArgs(tc.Function.Arguments),
				TaskObserver: mcpx.NewDefaultTaskObserver(progressCallback),
			})
			if submitErr != nil {
				l.Logger.Errorf("submit tool error: tool=%s, err=%v", toolName, submitErr)
				sendToolEvent(stream, modelId, ToolEvent{
					Type:     "tool_error",
					ToolId:   toolId,
					ToolName: toolName,
					Error:    submitErr.Error(),
				})
				toolIndex++
				continue
			}

			// 等待异步任务完成
			result, awaitErr := promise.Await(ctx)
			l.Logger.Infof("tool executed: tool=%s, taskId=%s, result_len=%d, err=%v", toolName, taskID, len(result), awaitErr)

			// === 4. 发送完成事件 ===
			var toolResult string
			if awaitErr != nil {
				toolResult = fmt.Sprintf("tool error: %v", awaitErr)
				sendToolEvent(stream, modelId, ToolEvent{
					Type:     "tool_error",
					ToolId:   toolId,
					ToolName: toolName,
					Error:    awaitErr.Error(),
				})
			} else {
				toolResult = result
				summary := result
				if len(summary) > 100 {
					summary = summary[:100] + "..."
				}
				sendToolEvent(stream, modelId, ToolEvent{
					Type:          "tool_success",
					ToolId:        toolId,
					ToolName:      toolName,
					ResultSummary: summary,
				})
			}

			// 回填 tool 结果给 LLM
			req.Messages = append(req.Messages, provider.ChatMessage{
				Role:       "tool",
				Content:    toolResult,
				ToolCallId: tc.Id,
			})

			toolIndex++
		}

		// 第一轮后清除工具定义，后续轮次不再传输
		if round == 0 {
			req.Tools = nil
			l.Logger.Debugf("cleared tools after first round")
		}
	}

	return nil
}

// handleStreamError 处理流式错误
func (l *ChatCompletionStreamLogic) handleStreamError(awaitErr error, streamCtx context.Context) error {
	if errors.Is(awaitErr, io.EOF) {
		return nil
	}

	switch {
	case l.ctx.Err() != nil:
		l.Logger.Infof("stream client disconnected, awaitErr: %v", awaitErr)
		return nil
	case streamCtx.Err() != nil:
		l.Logger.Errorf("stream total timeout (%v), awaitErr: %v",
			l.svcCtx.Config.StreamTimeout, awaitErr)
		return status.Errorf(codes.DeadlineExceeded, "stream total timeout")
	case errors.Is(awaitErr, context.DeadlineExceeded):
		l.Logger.Errorf("stream idle timeout (%v), awaitErr: %v",
			l.svcCtx.Config.StreamIdleTimeout, awaitErr)
		return status.Errorf(codes.DeadlineExceeded, "stream idle timeout")
	default:
		l.Logger.Errorf("stream recv error: %v", awaitErr)
		return provider.ToGrpcError(awaitErr)
	}
}

// toProtoStreamChunk 将 provider 流式 chunk 转为 proto chunk
func toProtoStreamChunk(chunk *provider.StreamChunk, modelId string) *aichat.ChatCompletionStreamChunk {
	choices := make([]*aichat.ChunkChoicePb, len(chunk.Choices))
	for i, c := range chunk.Choices {
		delta := &aichat.ChatDeltaPb{
			Role:             c.Delta.Role,
			Content:          c.Delta.Content,
			ReasoningContent: c.Delta.ReasoningContent,
		}
		// 转换 tool_calls 增量
		if len(c.Delta.ToolCalls) > 0 {
			delta.ToolCalls = make([]*aichat.ToolCallDeltaPb, len(c.Delta.ToolCalls))
			for j, tc := range c.Delta.ToolCalls {
				delta.ToolCalls[j] = &aichat.ToolCallDeltaPb{
					Index: int32(tc.Index),
					Id:    tc.Id,
					Type:  tc.Type,
					Function: &aichat.ToolCallFunctionDeltaPb{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
		choices[i] = &aichat.ChunkChoicePb{
			Index:        int32(c.Index),
			Delta:        delta,
			FinishReason: c.FinishReason,
		}
	}

	return &aichat.ChatCompletionStreamChunk{
		Id:      chunk.Id,
		Object:  chunk.Object,
		Created: chunk.Created,
		Model:   modelId,
		Choices: choices,
	}
}
