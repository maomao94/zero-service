package pass

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ssex"

	"github.com/zeromicro/go-zero/core/logx"
)

type ChatCompletionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	w      http.ResponseWriter
	r      *http.Request
}

func NewChatCompletionsLogic(ctx context.Context, svcCtx *svc.ServiceContext, w http.ResponseWriter, r *http.Request) *ChatCompletionsLogic {
	return &ChatCompletionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		w:      w,
		r:      r,
	}
}

// ChatCompletions 统一入口，根据 req.Stream 判断流式/非流式
func (l *ChatCompletionsLogic) ChatCompletions(req *types.ChatCompletionRequest) (*types.ChatCompletionResponse, error) {
	if req.Stream {
		return nil, l.handleStream(req)
	}
	return l.handleSync(req)
}

// handleSync 非流式处理：调用 aichat RPC
func (l *ChatCompletionsLogic) handleSync(req *types.ChatCompletionRequest) (*types.ChatCompletionResponse, error) {
	protoReq := toProtoRequest(req)

	resp, err := l.svcCtx.AiChatCli.ChatCompletion(l.ctx, protoReq)
	if err != nil {
		return nil, err
	}

	return toHTTPResponse(resp), nil
}

// handleStream 流式处理：gRPC server-side stream → SSE 桥接
func (l *ChatCompletionsLogic) handleStream(req *types.ChatCompletionRequest) error {
	sw, err := ssex.NewWriter(l.w)
	if err != nil {
		return err
	}

	protoReq := toProtoRequest(req)

	grpcStream, err := l.svcCtx.AiChatCli.ChatCompletionStream(l.ctx, protoReq)
	if err != nil {
		return err
	}

	l.Logger.Infof("chat completions stream started, model: %s", req.Model)

	for {
		chunk, err := grpcStream.Recv()
		if errors.Is(err, io.EOF) {
			sw.WriteDone()
			l.Logger.Infof("stream completed, model: %s", req.Model)
			return nil
		}
		if err != nil {
			l.Logger.Errorf("stream recv error: %v", err)
			// SSE 已开始写入（HTTP 200 已提交），不能再返回 error 给 handler 写 JSON 错误，
			// 否则客户端 SSE parser 无法解析混入的 JSON 错误体。
			// 直接返回 nil，让连接正常关闭，客户端通过未收到 [DONE] 检测异常。
			return nil
		}

		// 检查 HTTP 客户端是否断开
		select {
		case <-l.r.Context().Done():
			l.Logger.Infof("stream client disconnected")
			return nil
		default:
		}

		// 检查是否为工具事件（role=tool，content 是 JSON 格式）
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil && chunk.Choices[0].Delta.Role == "tool" {
			content := chunk.Choices[0].Delta.Content
			eventType := inferToolEventType(content)
			if eventType != "" {
				// 写入 SSE 事件：event: {type}\ndata: {content}\n\n
				sw.WriteEvent(eventType, content)
				continue
			}
		}

		httpChunk := toHTTPChunk(chunk)
		if err := sw.WriteJSON(httpChunk); err != nil {
			l.Logger.Errorf("write sse chunk error: %v", err)
			return nil
		}
	}
}

// inferToolEventType 根据 JSON 内容推断工具事件类型
func inferToolEventType(content string) string {
	var event map[string]interface{}
	if err := json.Unmarshal([]byte(content), &event); err != nil {
		return ""
	}
	if t, ok := event["type"].(string); ok {
		return t
	}
	return ""
}

// toProtoRequest 将 HTTP JSON 请求转为 gRPC proto 请求。
// 负责将 HTTP 层的 OpenAI 标准 thinking 对象转换为 gRPC 层的 bool enable_thinking，
// 同时透传 ReasoningContent 使 aichat 服务能正确处理 thinking 模式相关参数。
func toProtoRequest(req *types.ChatCompletionRequest) *aichat.ChatCompletionReq {
	messages := make([]*aichat.ChatMessagePb, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = &aichat.ChatMessagePb{
			Role:             m.Role,
			Content:          m.Content,
			ReasoningContent: m.ReasoningContent,
		}
	}

	// 将 OpenAI 标准的 thinking 对象（{"thinking":{"type":"enabled"}}）
	// 映射为内部 proto 的 bool enable_thinking
	enableThinking := req.Thinking.Type == "enabled"

	return &aichat.ChatCompletionReq{
		Model:          req.Model,
		Messages:       messages,
		Temperature:    req.Temperature,
		TopP:           req.TopP,
		MaxTokens:      int32(req.MaxTokens),
		Stop:           req.Stop,
		User:           req.User,
		EnableThinking: enableThinking,
	}
}

// toHTTPResponse 将 gRPC proto 响应转为 HTTP JSON 响应。
// 包含 ReasoningContent 字段的透传，确保 thinking 模式下的推理内容能返回给前端。
func toHTTPResponse(resp *aichat.ChatCompletionRes) *types.ChatCompletionResponse {
	choices := make([]types.Choice, len(resp.Choices))
	for i, c := range resp.Choices {
		choices[i] = types.Choice{
			Index: int(c.Index),
			Message: types.ChatMessage{
				Role:             c.Message.Role,
				Content:          c.Message.Content,
				ReasoningContent: c.Message.ReasoningContent,
			},
			FinishReason: c.FinishReason,
		}
	}

	result := &types.ChatCompletionResponse{
		Id:      resp.Id,
		Object:  resp.Object,
		Created: resp.Created,
		Model:   resp.Model,
		Choices: choices,
	}

	if resp.Usage != nil {
		result.Usage = types.Usage{
			PromptTokens:     int(resp.Usage.PromptTokens),
			CompletionTokens: int(resp.Usage.CompletionTokens),
			TotalTokens:      int(resp.Usage.TotalTokens),
		}
	}

	return result
}

// toHTTPChunk 将 gRPC proto 流式 chunk 转为 HTTP SSE chunk。
// 透传 Delta.ReasoningContent，前端通过此字段实时展示模型的推理思考过程。
// 当有 tool_calls 时，透传工具调用增量。
func toHTTPChunk(chunk *aichat.ChatCompletionStreamChunk) types.ChatCompletionChunk {
	choices := make([]types.ChunkChoice, len(chunk.Choices))
	for i, c := range chunk.Choices {
		delta := types.ChatDelta{
			Role:             c.Delta.Role,
			Content:          c.Delta.Content,
			ReasoningContent: c.Delta.ReasoningContent,
		}
		// 转换 tool_calls 增量
		if len(c.Delta.ToolCalls) > 0 {
			delta.ToolCalls = make([]types.ToolCallDelta, len(c.Delta.ToolCalls))
			for j, tc := range c.Delta.ToolCalls {
				delta.ToolCalls[j] = types.ToolCallDelta{
					Index: int(tc.Index),
					Id:    tc.Id,
					Type:  tc.Type,
					Function: types.ToolCallFunctionDelta{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
		cc := types.ChunkChoice{
			Index: int(c.Index),
			Delta: delta,
		}
		if c.FinishReason != "" {
			reason := c.FinishReason
			cc.FinishReason = &reason
		}
		choices[i] = cc
	}

	return types.ChatCompletionChunk{
		Id:      chunk.Id,
		Object:  chunk.Object,
		Created: chunk.Created,
		Model:   chunk.Model,
		Choices: choices,
	}
}
