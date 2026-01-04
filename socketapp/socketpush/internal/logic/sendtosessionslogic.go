package logic

import (
	"context"
	"zero-service/socketapp/socketgtw/socketgtw"

	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

type SendToSessionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendToSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendToSessionsLogic {
	return &SendToSessionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 向指定 session 批量发送消息
func (l *SendToSessionsLogic) SendToSessions(in *socketpush.SendToSessionsReq) (*socketpush.SendToSessionsRes, error) {
	baseCtx := context.WithoutCancel(l.ctx)
	for _, cli := range l.svcCtx.SocketContainer.GetClients() {
		threading.GoSafe(func() {
			cli.SendToSessions(baseCtx, &socketgtw.SendToSessionsReq{
				ReqId:   in.ReqId,
				SIds:    in.SIds,
				Event:   in.Event,
				Payload: in.Payload,
			})
		})
	}
	return &socketpush.SendToSessionsRes{
		ReqId: in.ReqId,
	}, nil
}
