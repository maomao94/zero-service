package logic

import (
	"context"
	"errors"
	"io"

	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aichat/internal/provider"
	"zero-service/aiapp/aichat/internal/svc"
	"zero-service/common/antsx"

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
	p, backendModel, providerName, err := l.svcCtx.Registry.GetProvider(in.Model)
	if err != nil {
		return status.Errorf(codes.NotFound, "model %s not found", in.Model)
	}

	req := toProviderRequest(in, backendModel, providerName)

	l.Logger.Infof("chat completion stream, model: %s -> %s (%s)", in.Model, backendModel, providerName)

	// 总超时：包裹整个 stream 生命周期，超时后 HTTP body 自动关闭
	streamCtx, streamCancel := context.WithTimeout(l.ctx, l.svcCtx.Config.StreamTimeout)
	defer streamCancel()

	reader, err := p.ChatCompletionStream(streamCtx, req)
	if err != nil {
		l.Logger.Errorf("chat completion stream error: %v", err)
		return toGrpcError(err)
	}
	defer reader.Close()

	idleTimeout := l.svcCtx.Config.StreamIdleTimeout

	for {
		// 将阻塞的 Recv 包装到 Promise，使其可被超时中断
		recv := antsx.NewPromise[*provider.StreamChunk]("stream-recv")
		go func() {
			chunk, err := reader.Recv()
			if err != nil {
				recv.Reject(err)
			} else {
				recv.Resolve(chunk)
			}
		}()

		// 空闲超时 ctx：继承 streamCtx（总超时 + 客户端断开），叠加 chunk 间空闲超时
		idleCtx, idleCancel := context.WithTimeout(streamCtx, idleTimeout)
		chunk, err := recv.Await(idleCtx)
		idleCancel()

		if err != nil {
			if errors.Is(err, io.EOF) {
				l.Logger.Infof("stream completed, model: %s", in.Model)
				return nil
			}
			// 优先检查客户端是否断开（l.ctx 来自 stream.Context()，gRPC 客户端断开时被取消）
			// 不依赖 error 类型判断，避免竞态：gRPC 取消传播有网络延迟，
			// 且 reader.Recv() 返回的错误不一定包装了 context 错误
			if l.ctx.Err() != nil {
				l.Logger.Infof("stream client disconnected, model: %s", in.Model)
				return nil
			}
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				l.Logger.Errorf("stream timeout (idle=%v, total=%v), model: %s",
					idleTimeout, l.svcCtx.Config.StreamTimeout, in.Model)
				return status.Errorf(codes.DeadlineExceeded, "stream timeout")
			}
			l.Logger.Errorf("stream recv error: %v", err)
			return toGrpcError(err)
		}

		protoChunk := toProtoStreamChunk(chunk, in.Model)
		if err := stream.Send(protoChunk); err != nil {
			l.Logger.Errorf("stream send error: %v", err)
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
				Role:             c.Delta.Role,
				Content:          c.Delta.Content,
				ReasoningContent: c.Delta.ReasoningContent,
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
