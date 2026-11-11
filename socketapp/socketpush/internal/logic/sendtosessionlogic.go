package logic

import (
	"context"
	"zero-service/socketapp/socketgtw/socketgtw"

	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendToSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendToSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendToSessionLogic {
	return &SendToSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 向指定 session 发送消息
func (l *SendToSessionLogic) SendToSession(in *socketpush.SendToSessionReq) (*socketpush.SendToSessionRes, error) {
	err := fanoutPush(l.ctx, l.Logger, l.svcCtx.SocketContainer.GetClients(), l.svcCtx.Config.PushTimeout,
		func(ctx context.Context, cli socketgtw.SocketGtwClient) error {
			_, err := cli.SendToSession(ctx, &socketgtw.SendToSessionReq{
				ReqId:    in.ReqId,
				SocketId: in.SocketId,
				Event:    in.Event,
				Payload:  in.Payload,
			})
			return err
		})
	if err != nil {
		return nil, err
	}
	return &socketpush.SendToSessionRes{
		ReqId: in.ReqId,
	}, nil
}
