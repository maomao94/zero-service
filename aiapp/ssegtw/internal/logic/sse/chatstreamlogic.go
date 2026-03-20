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
	flusher, ok := l.w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	// 确定 channel，未指定则生成唯一 ID
	channel := req.Channel
	if len(channel) == 0 {
		channel, _ = tool.SimpleUUID()
	}

	l.Infof("chat stream connected, channel: %s, prompt: %s", channel, req.Prompt)

	// 订阅事件
	msgChan, cancel := l.svcCtx.Emitter.Subscribe(channel)
	defer cancel()

	// 心跳定时器
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 发送连接成功事件
	fmt.Fprintf(l.w, "event: connected\ndata: {\"channel\":\"%s\"}\n\n", channel)
	flusher.Flush()

	// TODO: 这里未来会发起后端 RPC/MQ 调用，触发大模型推理
	// 后端服务通过 Emitter.Emit(channel, event) 推送流式结果
	// 例如:
	// go func() {
	//     res, err := l.svcCtx.ZeroRpcCli.ChatCompletion(l.ctx, &zerorpc.ChatReq{Prompt: req.Prompt})
	//     ...
	// }()

	for {
		select {
		case <-l.r.Context().Done():
			l.Infof("chat stream disconnected, channel: %s", channel)
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
