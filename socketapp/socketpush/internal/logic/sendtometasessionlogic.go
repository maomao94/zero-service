package logic

import (
	"context"
	"zero-service/socketapp/socketgtw/socketgtw"

	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendToMetaSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendToMetaSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendToMetaSessionLogic {
	return &SendToMetaSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 向指定元数据session 发送消息
func (l *SendToMetaSessionLogic) SendToMetaSession(in *socketpush.SendToMetaSessionReq) (*socketpush.SendToMetaSessionRes, error) {
	err := fanoutPush(l.ctx, l.Logger, l.svcCtx.SocketContainer.GetClients(), l.svcCtx.Config.PushTimeout,
		func(ctx context.Context, cli socketgtw.SocketGtwClient) error {
			_, err := cli.SendToMetaSession(ctx, &socketgtw.SendToMetaSessionReq{
				ReqId:   in.ReqId,
				Key:     in.Key,
				Value:   in.Value,
				Event:   in.Event,
				Payload: in.Payload,
			})
			return err
		})
	if err != nil {
		return nil, err
	}
	return &socketpush.SendToMetaSessionRes{
		ReqId: in.ReqId,
	}, nil
}
