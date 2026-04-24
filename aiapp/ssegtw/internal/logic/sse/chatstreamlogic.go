// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package sse

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"zero-service/aiapp/ssegtw/internal/svc"
	"zero-service/aiapp/ssegtw/internal/types"
	"zero-service/common/ssex"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/logx"
)

type ChatStreamLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	w      http.ResponseWriter
	r      *http.Request
}

// AI对话流
func NewChatStreamLogic(ctx context.Context, svcCtx *svc.ServiceContext, w http.ResponseWriter, r *http.Request) *ChatStreamLogic {
	return &ChatStreamLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		w:      w,
		r:      r,
	}
}

func (l *ChatStreamLogic) ChatStream(req *types.ChatStreamRequest) error {
	sw, err := ssex.NewWriter(l.w)
	if err != nil {
		return err
	}

	// 确定 channel，未指定则生成唯一 ID
	channel := req.Channel
	if len(channel) == 0 {
		channel, _ = tool.SimpleUUID()
	}

	prompt := req.Prompt
	if len(prompt) == 0 {
		prompt = "Hello World"
	}

	l.Infof("chat stream connected, channel: %s, prompt: %s", channel, prompt)

	// 1. 注册完成信号（PendingRegistry）
	donePromise, err := l.svcCtx.PendingReg.Register(channel, 60*time.Second)
	if err != nil {
		return fmt.Errorf("register pending failed: %w", err)
	}

	// 2. 订阅事件流（EventEmitter）
	msgSR, cancel := l.svcCtx.Emitter.Subscribe(l.ctx, channel)
	defer cancel()

	sw.WriteEvent("connected", fmt.Sprintf(`{"channel":"%s"}`, channel))

	go func() {
		tokens := []rune(prompt)
		for _, token := range tokens {
			time.Sleep(500 * time.Millisecond)
			l.svcCtx.Emitter.Emit(channel, svc.SSEEvent{
				Event: "token",
				Data:  string(token),
			})
		}
		time.Sleep(300 * time.Millisecond)
		l.svcCtx.Emitter.Emit(channel, svc.SSEEvent{
			Event: "done",
			Data:  "生成完毕",
		})
		l.svcCtx.PendingReg.Resolve(channel, "completed")
	}()

	go func() {
		donePromise.Await(l.r.Context())
		cancel()
	}()

	type recvResult struct {
		msg svc.SSEEvent
		ok  bool
	}
	msgCh := make(chan recvResult, 1)
	go func() {
		defer close(msgCh)
		for {
			msg, err := msgSR.Recv()
			if err != nil {
				return
			}
			msgCh <- recvResult{msg: msg, ok: true}
		}
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-l.r.Context().Done():
			l.Infof("chat stream disconnected, channel: %s", channel)
			return nil
		case r, ok := <-msgCh:
			if !ok {
				l.Infof("chat stream completed, channel: %s", channel)
				return nil
			}
			if len(r.msg.Event) > 0 {
				sw.WriteEvent(r.msg.Event, r.msg.Data)
			} else {
				sw.WriteData(r.msg.Data)
			}
		case <-ticker.C:
			sw.WriteKeepAlive()
		}
	}
}
