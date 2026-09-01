package webhook

import (
	"context"

	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/svc"

	"github.com/livekit/protocol/livekit"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/proto"
)

// WebhookNotifyLogic 负责把验签通过的 LiveKit webhook 事件透传给 app/live。
// 验签本身在 handler 层（依赖 HTTP 请求体/接口头），本层只做序列化+转发。
type WebhookNotifyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewWebhookNotifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WebhookNotifyLogic {
	return &WebhookNotifyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// WebhookNotify 把 *livekit.WebhookEvent 序列化为原始 proto 字节，经 WebhookNotify RPC
// 透传给 app/live 处理（幂等/事件全 case 由业务侧负责）。
func (l *WebhookNotifyLogic) WebhookNotify(event *livekit.WebhookEvent) error {
	data, err := proto.Marshal(event)
	if err != nil {
		return err
	}
	_, err = l.svcCtx.LiveRpcCli.WebhookNotify(l.ctx, &live.WebhookNotifyReq{Data: data})
	return err
}
