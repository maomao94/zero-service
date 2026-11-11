package logic

import (
	"context"
	"zero-service/socketapp/socketgtw/socketgtw"

	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/logx"
)

type KickMetaSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewKickMetaSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *KickMetaSessionLogic {
	return &KickMetaSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 指定元数据剔除 session
func (l *KickMetaSessionLogic) KickMetaSession(in *socketpush.KickMetaSessionReq) (*socketpush.KickMetaSessionRes, error) {
	err := fanoutPush(l.ctx, l.Logger, l.svcCtx.SocketContainer.GetClients(), l.svcCtx.Config.PushTimeout,
		func(ctx context.Context, cli socketgtw.SocketGtwClient) error {
			_, err := cli.KickMetaSession(ctx, &socketgtw.KickMetaSessionReq{
				ReqId: in.ReqId,
				Key:   in.Key,
				Value: in.Value,
			})
			return err
		})
	if err != nil {
		return nil, err
	}
	return &socketpush.KickMetaSessionRes{
		ReqId: in.ReqId,
	}, nil
}
