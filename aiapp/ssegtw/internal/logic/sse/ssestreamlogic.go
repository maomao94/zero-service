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
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/logx"
)

type SseStreamLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	w      http.ResponseWriter
	r      *http.Request
}

// SSE事件流
func NewSseStreamLogic(ctx context.Context, svcCtx *svc.ServiceContext, w http.ResponseWriter, r *http.Request) *SseStreamLogic {
	return &SseStreamLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		w:      w,
		r:      r,
	}
}

func (l *SseStreamLogic) SseStream(req *types.SSEStreamRequest) error {
	flusher, ok := l.w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	// 确定 channel，未指定则生成唯一 ID
	channel := req.Channel
	if len(channel) == 0 {
		channel, _ = tool.SimpleUUID()
	}

	l.Infof("sse stream connected, channel: %s", channel)

	// 订阅事件
	msgChan, cancel := l.svcCtx.Emitter.Subscribe(channel)
	defer cancel()

	// 心跳定时器
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 发送连接成功事件
	fmt.Fprintf(l.w, "event: connected\ndata: {\"channel\":\"%s\"}\n\n", channel)
	flusher.Flush()

	for {
		select {
		case <-l.r.Context().Done():
			l.Infof("sse stream disconnected, channel: %s", channel)
			return nil
		case msg, ok := <-msgChan:
			if !ok {
				return nil
			}
			if len(msg.Event) > 0 {
				fmt.Fprintf(l.w, "event: %s\ndata: %s\n\n", msg.Event, msg.Data)
			} else {
				fmt.Fprintf(l.w, "data: %s\n\n", msg.Data)
			}
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(l.w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}
