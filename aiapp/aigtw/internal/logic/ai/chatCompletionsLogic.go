package ai

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ssex"
	"zero-service/common/tool"

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
	if err := l.validateModel(req.Model); err != nil {
		return nil, err
	}

	if req.Stream {
		return nil, l.handleStream(req)
	}
	return l.handleSync(req)
}

// handleSync 非流式处理
func (l *ChatCompletionsLogic) handleSync(req *types.ChatCompletionRequest) (*types.ChatCompletionResponse, error) {
	id, _ := tool.SimpleUUID()
	completionId := "chatcmpl-" + id
	prompt := l.getLastUserMessage(req.Messages)
	content := fmt.Sprintf("Echo [%s]: %s", req.Model, prompt)

	return &types.ChatCompletionResponse{
		Id:      completionId,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []types.Choice{
			{
				Index: 0,
				Message: types.ChatMessage{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		},
		Usage: types.Usage{
			PromptTokens:     len([]rune(prompt)),
			CompletionTokens: len([]rune(content)),
			TotalTokens:      len([]rune(prompt)) + len([]rune(content)),
		},
	}, nil
}

// handleStream 流式处理
func (l *ChatCompletionsLogic) handleStream(req *types.ChatCompletionRequest) error {
	sw, err := ssex.NewWriter(l.w)
	if err != nil {
		return err
	}

	id, _ := tool.SimpleUUID()
	completionId := "chatcmpl-" + id
	now := time.Now().Unix()
	prompt := l.getLastUserMessage(req.Messages)
	content := fmt.Sprintf("Echo [%s]: %s", req.Model, prompt)
	tokens := []rune(content)

	l.Infof("chat completions stream started, id: %s", completionId)

	// 发送首个 chunk：role
	sw.WriteJSON(types.ChatCompletionChunk{
		Id:      completionId,
		Object:  "chat.completion.chunk",
		Created: now,
		Model:   req.Model,
		Choices: []types.ChunkChoice{
			{Index: 0, Delta: types.ChatDelta{Role: "assistant"}},
		},
	})

	// 逐 token 发送
	for _, token := range tokens {
		select {
		case <-l.r.Context().Done():
			l.Infof("stream disconnected")
			return nil
		default:
			time.Sleep(50 * time.Millisecond)
			sw.WriteJSON(types.ChatCompletionChunk{
				Id:      completionId,
				Object:  "chat.completion.chunk",
				Created: now,
				Model:   req.Model,
				Choices: []types.ChunkChoice{
					{Index: 0, Delta: types.ChatDelta{Content: string(token)}},
				},
			})
		}
	}

	// 发送结束 chunk
	finishReason := "stop"
	sw.WriteJSON(types.ChatCompletionChunk{
		Id:      completionId,
		Object:  "chat.completion.chunk",
		Created: now,
		Model:   req.Model,
		Choices: []types.ChunkChoice{
			{Index: 0, Delta: types.ChatDelta{}, FinishReason: &finishReason},
		},
	})

	sw.WriteDone()
	l.Infof("stream completed, id: %s", completionId)
	return nil
}

// validateModel 校验模型
func (l *ChatCompletionsLogic) validateModel(model string) error {
	for _, ab := range l.svcCtx.Config.Abilities {
		if ab.Id == model {
			return nil
		}
	}
	var available []string
	for _, ab := range l.svcCtx.Config.Abilities {
		available = append(available, ab.Id)
	}
	return types.NewModelNotFoundError(model, available)
}

// getLastUserMessage 获取最后一条 user 消息
func (l *ChatCompletionsLogic) getLastUserMessage(messages []types.ChatMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}
	return "Hello World"
}
