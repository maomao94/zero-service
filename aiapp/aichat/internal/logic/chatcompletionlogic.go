package logic

import (
	"context"
	"errors"

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
	p, backendModel, providerName, err := l.svcCtx.Registry.GetProvider(in.Model)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "model %s not found", in.Model)
	}

	// 构建 provider 请求
	req := toProviderRequest(in, backendModel, providerName)

	l.Infof("chat completion, model: %s -> %s (%s)", in.Model, backendModel, providerName)

	resp, err := p.ChatCompletion(l.ctx, req)
	if err != nil {
		l.Errorf("chat completion error: %v", err)
		return nil, toGrpcError(err)
	}

	// 转换为 proto 响应，model 替换回原始 ID
	return toProtoResponse(resp, in.Model), nil
}

// toProviderRequest 将 gRPC proto 请求转为 provider 内部请求。
// 接收 providerName 参数，用于根据厂商名称构建特有的扩展参数（如 thinking 模式参数）。
func toProviderRequest(in *aichat.ChatCompletionReq, backendModel, providerName string) *provider.ChatRequest {
	messages := make([]provider.ChatMessage, len(in.Messages))
	for i, m := range in.Messages {
		messages[i] = provider.ChatMessage{
			Role:             m.Role,
			Content:          m.Content,
			ReasoningContent: m.ReasoningContent,
		}
	}

	req := &provider.ChatRequest{
		Model:       backendModel,
		Messages:    messages,
		Temperature: in.Temperature,
		TopP:        in.TopP,
	}
	// 若启用深度思考模式，根据厂商名称构建对应的扩展参数并注入 ExtraBody。
	// ExtraBody 会在 provider 层的 marshalWithExtraBody() 中合并到 JSON 请求体顶层。
	if in.EnableThinking {
		req.ExtraBody = buildThinkingParams(providerName, true)
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

// buildThinkingParams 根据 provider 名称构建厂商特有的深度思考（thinking）参数。
//
// 不同大模型厂商启用 thinking 模式的请求参数格式不同：
//
//   - dashscope（千问/通义）: 在请求体顶层添加 {"enable_thinking": true}
//     文档: https://help.aliyun.com/zh/model-studio/developer-reference/openai-sdk
//
//   - openai / zhipu（智谱）及其他兼容厂商（默认）:
//     在请求体顶层添加 {"thinking": {"type": "enabled", "clear_thinking": true}}
//     其中 clear_thinking 表示自动清除历史 messages 中的 reasoning_content，
//     避免思考内容占用 token 额度。
//     文档: https://docs.bigmodel.cn/api-reference
//
// 返回的 map 会赋值给 ChatRequest.ExtraBody，最终由 marshalWithExtraBody() 合并到 JSON 顶层。
func buildThinkingParams(providerName string, enable bool) map[string]any {
	switch providerName {
	case "dashscope":
		// 千问格式: {"enable_thinking": true}
		return map[string]any{
			"enable_thinking": enable,
		}
	default:
		// OpenAI 标准格式（智谱等）:
		// {"thinking": {"type": "enabled", "clear_thinking": true}}
		// clear_thinking: 自动清除历史消息中的 reasoning_content，避免额外占用 token
		t := "disabled"
		if enable {
			t = "enabled"
		}
		return map[string]any{
			"thinking": map[string]any{
				"type":           t,
				"clear_thinking": true,
			},
		}
	}
}

// toProtoResponse 将 provider 响应转为 proto 响应
func toProtoResponse(resp *provider.ChatResponse, modelId string) *aichat.ChatCompletionRes {
	choices := make([]*aichat.Choice, len(resp.Choices))
	for i, c := range resp.Choices {
		choices[i] = &aichat.Choice{
			Index: int32(c.Index),
			Message: &aichat.ChatMessage{
				Role:             c.Message.Role,
				Content:          c.Message.Content,
				ReasoningContent: c.Message.ReasoningContent,
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
	var apiErr *provider.APIError
	if errors.As(err, &apiErr) {
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
