package logic

import (
	"context"
	"time"
	"zero-service/socketapp/socketgtw/socketgtw"

	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
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
	baseCtx := context.WithoutCancel(l.ctx)
	for _, cli := range l.svcCtx.SocketContainer.GetClients() {
		threading.GoSafe(func() {
			socktCTx, cancel := context.WithTimeout(baseCtx, 10*time.Second)
			defer cancel()
			cli.SendToMetaSession(socktCTx, &socketgtw.SendToMetaSessionReq{
				ReqId:   in.ReqId,
				Key:     in.Key,
				Value:   in.Value,
				Event:   in.Event,
				Payload: in.Payload,
			})
		})
	}
	return &socketpush.SendToMetaSessionRes{}, nil
}
