package logic

import (
	"context"
	"errors"
	"fmt"
	"io"

	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aichat/internal/mcpclient"
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

	// 注入 MCP 工具
	if l.svcCtx.McpClient != nil {
		req.Tools = l.svcCtx.McpClient.ToOpenAITools()
	}

	l.Logger.Infof("chat completion stream, model: %s -> %s (%s), tools: %d", in.Model, backendModel, providerName, len(req.Tools))

	// 总超时：包裹整个 stream 生命周期，超时后 HTTP body 自动关闭
	streamCtx, streamCancel := context.WithTimeout(l.ctx, l.svcCtx.Config.StreamTimeout)
	defer streamCancel()

	// 如果有工具，先用非流式完成 tool-calling 循环
	if len(req.Tools) > 0 {
		for round := 0; round < l.svcCtx.Config.MaxToolRounds; round++ {
			resp, err := p.ChatCompletion(streamCtx, req)
			if err != nil {
				l.Logger.Errorf("tool-calling round %d error: %v", round+1, err)
				return toGrpcError(err)
			}

			if len(resp.Choices) == 0 || resp.Choices[0].FinishReason != "tool_calls" {
				break // 没有更多工具调用
			}

			if l.svcCtx.McpClient == nil {
				break
			}

			// 执行工具调用，追加消息
			assistantMsg := resp.Choices[0].Message
			req.Messages = append(req.Messages, assistantMsg)

			for _, tc := range assistantMsg.ToolCalls {
				l.Infof("stream tool call round %d: %s(%s)", round+1, tc.Function.Name, tc.Function.Arguments)
				result, callErr := l.svcCtx.McpClient.CallTool(streamCtx, tc.Function.Name, mcpclient.ParseArgs(tc.Function.Arguments))
				if callErr != nil {
					l.Logger.Errorf("tool call %s error: %v", tc.Function.Name, callErr)
					result = fmt.Sprintf("tool error: %v", callErr)
				}
				req.Messages = append(req.Messages, provider.ChatMessage{
					Role:       "tool",
					Content:    result,
					ToolCallId: tc.Id,
				})
			}
		}
		// 清除 tools，最终流式调用不再触发工具
		req.Tools = nil
	}

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
			chunk, recvErr := reader.Recv()
			if recvErr != nil {
				recv.Reject(recvErr)
			} else {
				recv.Resolve(chunk)
			}
		}()

		// 空闲超时 ctx：继承 streamCtx（总超时 + 客户端断开），叠加 chunk 间空闲超时
		idleCtx, idleCancel := context.WithTimeout(streamCtx, idleTimeout)
		chunk, awaitErr := recv.Await(idleCtx)
		idleCancel()

		if awaitErr != nil {
			if errors.Is(awaitErr, io.EOF) {
				l.Logger.Infof("stream completed, model: %s", in.Model)
				return nil
			}
			// 按优先级判断：客户端断开 > 总超时 > 空闲超时 > 上游错误
			switch {
			case l.ctx.Err() != nil:
				// gRPC 客户端断开（浏览器关闭 SSE → aigtw 取消 gRPC 调用 → l.ctx 取消）
				l.Logger.Infof("stream client disconnected, model: %s, awaitErr: %v",
					in.Model, awaitErr)
				return nil
			case streamCtx.Err() != nil:
				// streamCtx 超时但 l.ctx 未取消 → 总超时到期
				l.Logger.Errorf("stream total timeout (%v), model: %s, awaitErr: %v",
					l.svcCtx.Config.StreamTimeout, in.Model, awaitErr)
				return status.Errorf(codes.DeadlineExceeded, "stream total timeout")
			case errors.Is(awaitErr, context.DeadlineExceeded):
				// streamCtx 正常但 awaitErr 是 DeadlineExceeded → idleCtx 空闲超时
				l.Logger.Errorf("stream idle timeout (%v), model: %s, awaitErr: %v",
					idleTimeout, in.Model, awaitErr)
				return status.Errorf(codes.DeadlineExceeded, "stream idle timeout")
			default:
				// 上游 provider 返回的业务错误
				l.Logger.Errorf("stream recv error: %v, model: %s", awaitErr, in.Model)
				return toGrpcError(awaitErr)
			}
		}

		// select 竞态：Resolve 和 ctx.Done 同时 ready 时可能选中 Resolve，
		// 此时 awaitErr == nil 但 l.ctx 已取消，提前退出避免无意义的 Send
		if l.ctx.Err() != nil {
			l.Logger.Infof("stream client disconnected, model: %s", in.Model)
			return nil
		}

		protoChunk := toProtoStreamChunk(chunk, in.Model)
		if sendErr := stream.Send(protoChunk); sendErr != nil {
			l.Logger.Errorf("stream send error: %v", sendErr)
			return sendErr
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
