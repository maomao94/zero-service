package logic

import (
	"context"
	"io"

	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aichat/internal/provider"
	"zero-service/aiapp/aichat/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
	p, backendModel, err := l.svcCtx.Registry.GetProvider(in.Model)
	if err != nil {
		return status.Errorf(codes.NotFound, "model %s not found", in.Model)
	}

	req := toProviderRequest(in, backendModel)

	l.Infof("chat completion stream, model: %s -> %s", in.Model, backendModel)

	reader, err := p.ChatCompletionStream(l.ctx, req)
	if err != nil {
		l.Errorf("chat completion stream error: %v", err)
		return toGrpcError(err)
	}
	defer reader.Close()

	for {
		chunk, err := reader.Recv()
		if err == io.EOF {
			l.Infof("stream completed, model: %s", in.Model)
			return nil
		}
		if err != nil {
			l.Errorf("stream recv error: %v", err)
			return toGrpcError(err)
		}

		// 检查客户端是否断开
		select {
		case <-stream.Context().Done():
			l.Infof("stream client disconnected, model: %s", in.Model)
			return nil
		default:
		}

		// 转换并发送 chunk
		protoChunk := toProtoStreamChunk(chunk, in.Model)
		if err := stream.Send(protoChunk); err != nil {
			l.Errorf("stream send error: %v", err)
			return err
		}
	}
}

// toProtoStreamChunk 将 provider 流式 chunk 转为 proto chunk
func toProtoStreamChunk(chunk *provider.StreamChunk, modelId string) *aichat.ChatCompletionStreamChunk {
	choices := make([]*aichat.ChunkChoice, len(chunk.Choices))
	for i, c := range chunk.Choices {
		choices[i] = &aichat.ChunkChoice{
			Index: int32(c.Index),
			Delta: &aichat.ChatDelta{
				Role:    c.Delta.Role,
				Content: c.Delta.Content,
			},
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
