package logic

import (
	"context"
	"zero-service/socketapp/socketgtw/socketgtw"

	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

type BroadcastGlobalLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBroadcastGlobalLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BroadcastGlobalLogic {
	return &BroadcastGlobalLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 向所有在线前端广播消息
func (l *BroadcastGlobalLogic) BroadcastGlobal(in *socketpush.BroadcastGlobalReq) (*socketpush.BroadcastGlobalRes, error) {
	baseCtx := context.WithoutCancel(l.ctx)
	for _, cli := range l.svcCtx.SocketContainer.GetClients() {
		threading.GoSafe(func() {
			cli.BroadcastGlobal(baseCtx, &socketgtw.BroadcastGlobalReq{
				ReqId:   in.ReqId,
				Event:   in.Event,
				Payload: in.Payload,
			})
		})
	}
	return &socketpush.BroadcastGlobalRes{}, nil
}
