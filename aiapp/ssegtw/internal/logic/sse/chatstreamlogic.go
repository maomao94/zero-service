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
	msgChan, cancel := l.svcCtx.Emitter.Subscribe(channel)
	defer cancel()

	// 3. 发送连接成功事件
	sw.WriteEvent("connected", fmt.Sprintf(`{"channel":"%s"}`, channel))

	// 4. 启动模拟 worker：逐字符输出 token，最后 Resolve 完成信号
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

	// 5. 用独立 goroutine 等待完成信号，触发 cancel 关闭 msgChan
	go func() {
		donePromise.Await(l.r.Context())
		cancel()
	}()

	// 心跳定时器
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 6. 主循环：转发事件到客户端
	for {
		select {
		case <-l.r.Context().Done():
			l.Infof("chat stream disconnected, channel: %s", channel)
			return nil
		case msg, ok := <-msgChan:
			if !ok {
				l.Infof("chat stream completed, channel: %s", channel)
				return nil
			}
			if len(msg.Event) > 0 {
				sw.WriteEvent(msg.Event, msg.Data)
			} else {
				sw.WriteData(msg.Data)
			}
		case <-ticker.C:
			sw.WriteKeepAlive()
		}
	}
}
