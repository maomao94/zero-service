package logic

import (
	"context"
	"encoding/json"

	"zero-service/socketapp/socketgtw/internal/svc"
	"zero-service/socketapp/socketgtw/socketgtw"

	"github.com/zeromicro/go-zero/core/jsonx"
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
	l.Infof("BroadcastRoom, event:%s, room: %s, reqId: %s", in.Event, in.Room, in.ReqId)
	var payload any
	raw := []byte(in.Payload)
	var js json.RawMessage
	if jsonx.Unmarshal(raw, &js) == nil {
		payload = json.RawMessage(raw)
	} else {
		payload = in.Payload
	}
	err := l.svcCtx.SocketServer.BroadcastRoom(in.Room, in.Event, payload, in.ReqId)
	if err != nil {
		return nil, err
	}
	return &socketgtw.BroadcastRoomRes{
		ReqId: in.ReqId,
	}, nil
}
