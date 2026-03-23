package logic

import (
	"context"

	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aichat/internal/provider"
	"zero-service/aiapp/aichat/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ChatCompletionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewChatCompletionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatCompletionLogic {
	return &ChatCompletionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ChatCompletionLogic) ChatCompletion(in *aichat.ChatCompletionReq) (*aichat.ChatCompletionRes, error) {
	p, backendModel, err := l.svcCtx.Registry.GetProvider(in.Model)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "model %s not found", in.Model)
	}

	// 构建 provider 请求
	req := toProviderRequest(in, backendModel)

	l.Infof("chat completion, model: %s -> %s", in.Model, backendModel)

	resp, err := p.ChatCompletion(l.ctx, req)
	if err != nil {
		l.Errorf("chat completion error: %v", err)
		return nil, toGrpcError(err)
	}

	// 转换为 proto 响应，model 替换回原始 ID
	return toProtoResponse(resp, in.Model), nil
}

// toProviderRequest 将 proto 请求转为 provider 内部请求
func toProviderRequest(in *aichat.ChatCompletionReq, backendModel string) *provider.ChatRequest {
	messages := make([]provider.ChatMessage, len(in.Messages))
	for i, m := range in.Messages {
		messages[i] = provider.ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	req := &provider.ChatRequest{
		Model:    backendModel,
		Messages: messages,
	}
	if in.Temperature != 0 {
		req.Temperature = in.Temperature
	}
	if in.TopP != 0 {
		req.TopP = in.TopP
	}
	if in.MaxTokens != 0 {
		req.MaxTokens = int(in.MaxTokens)
	}
	if len(in.Stop) > 0 {
		req.Stop = in.Stop
	}
	if in.User != "" {
		req.User = in.User
	}
	return req
}

// toProtoResponse 将 provider 响应转为 proto 响应
func toProtoResponse(resp *provider.ChatResponse, modelId string) *aichat.ChatCompletionRes {
	choices := make([]*aichat.Choice, len(resp.Choices))
	for i, c := range resp.Choices {
		choices[i] = &aichat.Choice{
			Index: int32(c.Index),
			Message: &aichat.ChatMessage{
				Role:    c.Message.Role,
				Content: c.Message.Content,
			},
			FinishReason: c.FinishReason,
		}
	}

	return &aichat.ChatCompletionRes{
		Id:      resp.Id,
		Object:  resp.Object,
		Created: resp.Created,
		Model:   modelId,
		Choices: choices,
		Usage: &aichat.Usage{
			PromptTokens:     int32(resp.Usage.PromptTokens),
			CompletionTokens: int32(resp.Usage.CompletionTokens),
			TotalTokens:      int32(resp.Usage.TotalTokens),
		},
	}
}

// toGrpcError 将 provider 错误转为 gRPC status error
func toGrpcError(err error) error {
	if apiErr, ok := err.(*provider.APIError); ok {
		switch {
		case apiErr.StatusCode == 401 || apiErr.StatusCode == 403:
			return status.Errorf(codes.PermissionDenied, "upstream auth error: %s", apiErr.Body)
		case apiErr.StatusCode == 429:
			return status.Errorf(codes.ResourceExhausted, "upstream rate limit: %s", apiErr.Body)
		case apiErr.StatusCode == 400:
			return status.Errorf(codes.InvalidArgument, "upstream bad request: %s", apiErr.Body)
		default:
			return status.Errorf(codes.Unavailable, "upstream error (status %d): %s", apiErr.StatusCode, apiErr.Body)
		}
	}
	return status.Errorf(codes.Internal, "internal error: %v", err)
}
