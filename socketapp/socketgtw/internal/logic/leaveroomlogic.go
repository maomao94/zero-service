package logic

import (
	"context"
	"zero-service/socketapp/socketgtw/internal/svc"
	"zero-service/socketapp/socketgtw/socketgtw"

	"github.com/zeromicro/go-zero/core/logx"
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
func (l *LeaveRoomLogic) LeaveRoom(in *socketgtw.LeaveRoomReq) (*socketgtw.LeaveRoomRes, error) {
	session := l.svcCtx.SocketServer.GetSession(in.SId)
	if session != nil {
		err := session.LeaveRoom(in.Room)
		if err != nil {
			return nil, err
		}
	}
	return &socketgtw.LeaveRoomRes{
		ReqId: in.ReqId,
	}, nil
}
