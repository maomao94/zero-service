package logic

import (
	"context"
	"zero-service/socketapp/socketgtw/socketgtw"

	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/logx"
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
	err := fanoutPush(l.ctx, l.Logger, l.svcCtx.SocketContainer.GetClients(), l.svcCtx.Config.PushTimeout,
		func(ctx context.Context, cli socketgtw.SocketGtwClient) error {
			_, err := cli.SendToSessions(ctx, &socketgtw.SendToSessionsReq{
				ReqId:     in.ReqId,
				SocketIds: in.SocketIds,
				Event:     in.Event,
				Payload:   in.Payload,
			})
			return err
		})
	if err != nil {
		return nil, err
	}
	return &socketpush.SendToSessionsRes{
		ReqId: in.ReqId,
	}, nil
}
