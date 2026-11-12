package logic

import (
	"context"
	"zero-service/socketapp/socketgtw/socketgtw"

	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type SendToMetaSessionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendToMetaSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendToMetaSessionsLogic {
	return &SendToMetaSessionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 向指定元数据 session 批量发送消息
func (l *SendToMetaSessionsLogic) SendToMetaSessions(in *socketpush.SendToMetaSessionsReq) (*socketpush.SendToMetaSessionsRes, error) {
	if len(in.MetaSessions) != 0 {
		metaSessions := make([]*socketgtw.PbMetaSession, len(in.MetaSessions))
		copier.Copy(&metaSessions, in.MetaSessions)
		err := fanoutPush(l.ctx, l.Logger, l.svcCtx.SocketContainer.GetClients(), l.svcCtx.Config.PushTimeout,
			func(ctx context.Context, cli socketgtw.SocketGtwClient) error {
				_, err := cli.SendToMetaSessions(ctx, &socketgtw.SendToMetaSessionsReq{
					ReqId:        in.ReqId,
					MetaSessions: metaSessions,
					Event:        in.Event,
					Payload:      in.Payload,
				})
				return err
			})
		if err != nil {
			return nil, err
		}
	}
	return &socketpush.SendToMetaSessionsRes{
		ReqId: in.ReqId,
	}, nil
}
