// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package ai

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ssex"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/logx"
)

// ChatCompletionChunk OpenAI SSE 流式 chunk 结构
type ChatCompletionChunk struct {
	Id      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []ChunkChoice `json:"choices"`
}

type ChunkChoice struct {
	Index        int        `json:"index"`
	Delta        ChatDelta  `json:"delta"`
	FinishReason *string    `json:"finish_reason"`
}

type ChatDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type ChatCompletionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	w      http.ResponseWriter
	r      *http.Request
}

// 对话补全
func NewChatCompletionsLogic(ctx context.Context, svcCtx *svc.ServiceContext, w http.ResponseWriter, r *http.Request) *ChatCompletionsLogic {
	return &ChatCompletionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		w:      w,
		r:      r,
	}
}

// ChatCompletions 非流式对话补全
func (l *ChatCompletionsLogic) ChatCompletions(req *types.ChatCompletionRequest) (resp *types.ChatCompletionResponse, err error) {
	if err := l.validateModel(req.Model); err != nil {
		return nil, err
	}

	id, _ := tool.SimpleUUID()
	completionId := "chatcmpl-" + id

	// 获取最后一条 user 消息作为 mock 输出
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

// ChatCompletionsStream 流式对话补全（SSE）
func (l *ChatCompletionsLogic) ChatCompletionsStream(req *types.ChatCompletionRequest) error {
	if err := l.validateModel(req.Model); err != nil {
		return err
	}

	sw, err := ssex.NewWriter(l.w)
	if err != nil {
		return err
	}

	id, _ := tool.SimpleUUID()
	completionId := "chatcmpl-" + id
	now := time.Now().Unix()
	model := req.Model

	// 获取最后一条 user 消息
	prompt := l.getLastUserMessage(req.Messages)
	content := fmt.Sprintf("Echo [%s]: %s", model, prompt)

	l.Infof("chat completions stream started, id: %s, model: %s", completionId, model)

	// 注册完成信号
	channel := completionId
	donePromise, err := l.svcCtx.PendingReg.Register(channel, 60*time.Second)
	if err != nil {
		return fmt.Errorf("register pending failed: %w", err)
	}

	// 订阅事件流
	msgChan, cancel := l.svcCtx.Emitter.Subscribe(channel)
	defer cancel()

	// 发送首个 chunk：role = "assistant"
	if err := sw.WriteJSON(ChatCompletionChunk{
		Id:      completionId,
		Object:  "chat.completion.chunk",
		Created: now,
		Model:   model,
		Choices: []ChunkChoice{
			{
				Index: 0,
				Delta: ChatDelta{Role: "assistant"},
			},
		},
	}); err != nil {
		return err
	}

	// 启动 mock worker：逐字输出 token
	go func() {
		tokens := []rune(content)
		for _, token := range tokens {
			time.Sleep(50 * time.Millisecond)
			l.svcCtx.Emitter.Emit(channel, svc.ChunkEvent{
				Data: ChatCompletionChunk{
					Id:      completionId,
					Object:  "chat.completion.chunk",
					Created: now,
					Model:   model,
					Choices: []ChunkChoice{
						{
							Index: 0,
							Delta: ChatDelta{Content: string(token)},
						},
					},
				},
			})
		}

		// 发送 finish chunk
		finishReason := "stop"
		l.svcCtx.Emitter.Emit(channel, svc.ChunkEvent{
			Data: ChatCompletionChunk{
				Id:      completionId,
				Object:  "chat.completion.chunk",
				Created: now,
				Model:   model,
				Choices: []ChunkChoice{
					{
						Index:        0,
						Delta:        ChatDelta{},
						FinishReason: &finishReason,
					},
				},
			},
			Done: true,
		})

		l.svcCtx.PendingReg.Resolve(channel, "completed")
	}()

	// 等待完成信号后关闭 msgChan
	go func() {
		donePromise.Await(l.r.Context())
		cancel()
	}()

	// 心跳定时器
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 主循环：转发事件到客户端
	for {
		select {
		case <-l.r.Context().Done():
			l.Infof("chat completions stream disconnected, id: %s", completionId)
			return nil
		case msg, ok := <-msgChan:
			if !ok {
				// 流结束，发送 [DONE]
				sw.WriteDone()
				l.Infof("chat completions stream completed, id: %s", completionId)
				return nil
			}
			if msg.Error != nil {
				return msg.Error
			}
			if err := sw.WriteJSON(msg.Data); err != nil {
				return err
			}
			if msg.Done {
				sw.WriteDone()
				l.Infof("chat completions stream completed, id: %s", completionId)
				return nil
			}
		case <-ticker.C:
			sw.WriteKeepAlive()
		}
	}
}

// validateModel 校验 model 是否在配置的能力列表中
func (l *ChatCompletionsLogic) validateModel(model string) error {
	for _, ab := range l.svcCtx.Config.Abilities {
		if ab.Id == model {
			return nil
		}
	}

	var validModels []string
	for _, ab := range l.svcCtx.Config.Abilities {
		validModels = append(validModels, ab.Id)
	}
	return fmt.Errorf("model '%s' not found, available models: [%s]",
		model, strings.Join(validModels, ", "))
}

// getLastUserMessage 获取最后一条 user 角色的消息内容
func (l *ChatCompletionsLogic) getLastUserMessage(messages []types.ChatMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}
	return "Hello World"
}
