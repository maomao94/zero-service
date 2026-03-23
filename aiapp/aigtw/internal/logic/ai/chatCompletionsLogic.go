package ai

import (
	"context"
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

	l.Infof("chat completions stream started, model: %s", req.Model)

	for {
		chunk, err := grpcStream.Recv()
		if err == io.EOF {
			sw.WriteDone()
			l.Infof("stream completed, model: %s", req.Model)
			return nil
		}
		if err != nil {
			l.Errorf("stream recv error: %v", err)
			return err
		}

		// 检查 HTTP 客户端是否断开
		select {
		case <-l.r.Context().Done():
			l.Infof("stream client disconnected")
			return nil
		default:
		}

		httpChunk := toHTTPChunk(chunk)
		sw.WriteJSON(httpChunk)
	}
}

// toProtoRequest 将 HTTP 请求转为 proto 请求
func toProtoRequest(req *types.ChatCompletionRequest) *aichat.ChatCompletionReq {
	messages := make([]*aichat.ChatMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = &aichat.ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	return &aichat.ChatCompletionReq{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   int32(req.MaxTokens),
		Stop:        req.Stop,
		User:        req.User,
	}
}

// toHTTPResponse 将 proto 响应转为 HTTP JSON 响应
func toHTTPResponse(resp *aichat.ChatCompletionRes) *types.ChatCompletionResponse {
	choices := make([]types.Choice, len(resp.Choices))
	for i, c := range resp.Choices {
		choices[i] = types.Choice{
			Index: int(c.Index),
			Message: types.ChatMessage{
				Role:    c.Message.Role,
				Content: c.Message.Content,
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

// toHTTPChunk 将 proto 流式 chunk 转为 HTTP SSE chunk
func toHTTPChunk(chunk *aichat.ChatCompletionStreamChunk) types.ChatCompletionChunk {
	choices := make([]types.ChunkChoice, len(chunk.Choices))
	for i, c := range chunk.Choices {
		cc := types.ChunkChoice{
			Index: int(c.Index),
			Delta: types.ChatDelta{
				Role:    c.Delta.Role,
				Content: c.Delta.Content,
			},
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
