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

type SendOneToSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendOneToSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendOneToSessionLogic {
	return &SendOneToSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 向指定 session 发送消息
func (l *SendOneToSessionLogic) SendOneToSession(in *socketpush.SendOneToSessionReq) (*socketpush.SendOneToSessionRes, error) {
	baseCtx := context.WithoutCancel(l.ctx)
	for _, cli := range l.svcCtx.SocketContainer.GetClients() {
		threading.GoSafe(func() {
			socktCTx, cancel := context.WithTimeout(baseCtx, 10*time.Second)
			defer cancel()
			cli.SendOneToSession(socktCTx, &socketgtw.SendOneToSessionReq{
				ReqId:   in.ReqId,
				SId:     in.SId,
				Event:   in.Event,
				Payload: in.Payload,
			})
		})
	}
	return &socketpush.SendOneToSessionRes{}, nil
}
