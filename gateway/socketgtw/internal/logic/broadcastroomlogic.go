package logic

import (
	"context"

	"zero-service/gateway/socketgtw/internal/svc"
	"zero-service/gateway/socketgtw/socketgtw"

	"github.com/zeromicro/go-zero/core/logx"
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
func (l *BroadcastRoomLogic) BroadcastRoom(in *socketgtw.BroadcastRoomReq) (*socketgtw.BroadcastRoomRes, error) {
	err := l.svcCtx.SocketServer.BroadcastRoom(in.Room, in.Event, string(in.Payload), in.ReqId)
	if err != nil {
		return nil, err
	}
	return &socketgtw.BroadcastRoomRes{}, nil
}
