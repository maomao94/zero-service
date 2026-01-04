package logic

import (
	"context"
	"zero-service/socketapp/socketgtw/socketgtw"

	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

type LeaveRoomLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLeaveRoomLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LeaveRoomLogic {
	return &LeaveRoomLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 离开房间
func (l *LeaveRoomLogic) LeaveRoom(in *socketpush.LeaveRoomReq) (*socketpush.LeaveRoomRes, error) {
	baseCtx := context.WithoutCancel(l.ctx)
	for _, cli := range l.svcCtx.SocketContainer.GetClients() {
		threading.GoSafe(func() {
			cli.LeaveRoom(baseCtx, &socketgtw.LeaveRoomReq{
				ReqId: in.ReqId,
				SId:   in.SId,
				Room:  in.Room,
			})
		})
	}
	return &socketpush.LeaveRoomRes{
		ReqId: in.ReqId,
	}, nil
}
