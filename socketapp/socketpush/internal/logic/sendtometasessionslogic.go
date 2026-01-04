package logic

import (
	"context"
	"zero-service/socketapp/socketgtw/socketgtw"

	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
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
		baseCtx := context.WithoutCancel(l.ctx)
		metaSessions := make([]*socketgtw.PbMetaSession, len(in.MetaSessions))
		copier.Copy(&metaSessions, in.MetaSessions)
		for _, cli := range l.svcCtx.SocketContainer.GetClients() {
			threading.GoSafe(func() {
				cli.SendToMetaSessions(baseCtx, &socketgtw.SendToMetaSessionsReq{
					ReqId:        in.ReqId,
					MetaSessions: metaSessions,
					Event:        in.Event,
					Payload:      in.Payload,
				})
			})
		}
	}
	return &socketpush.SendToMetaSessionsRes{
		ReqId: in.ReqId,
	}, nil
}
