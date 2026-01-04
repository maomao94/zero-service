package logic

import (
	"context"
	"zero-service/socketapp/socketgtw/socketgtw"

	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

type BroadcastRoomLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBroadcastRoomLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BroadcastRoomLogic {
	return &BroadcastRoomLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 向指定房间广播消息
func (l *BroadcastRoomLogic) BroadcastRoom(in *socketpush.BroadcastRoomReq) (*socketpush.BroadcastRoomRes, error) {
	baseCtx := context.WithoutCancel(l.ctx)
	for _, cli := range l.svcCtx.SocketContainer.GetClients() {
		threading.GoSafe(func() {
			cli.BroadcastRoom(baseCtx, &socketgtw.BroadcastRoomReq{
				ReqId:   in.ReqId,
				Room:    in.Room,
				Event:   in.Event,
				Payload: in.Payload,
			})
		})
	}
	return &socketpush.BroadcastRoomRes{
		ReqId: in.ReqId,
	}, nil
}
