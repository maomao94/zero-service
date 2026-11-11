package logic

import (
	"context"
	"zero-service/socketapp/socketgtw/socketgtw"

	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

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
func (l *BroadcastRoomLogic) BroadcastRoom(in *socketpush.BroadcastRoomReq) (*socketpush.BroadcastRoomRes, error) {
	// 广播类接口保持原语义：单个网关失败只记录日志，不返回错误，
	// 避免调用方重试导致重复广播；失败节点已由 fanoutPush 逐个记录。
	err := fanoutPush(l.ctx, l.Logger, l.svcCtx.SocketContainer.GetClients(), l.svcCtx.Config.PushTimeout,
		func(ctx context.Context, cli socketgtw.SocketGtwClient) error {
			_, err := cli.BroadcastRoom(ctx, &socketgtw.BroadcastRoomReq{
				ReqId:   in.ReqId,
				Room:    in.Room,
				Event:   in.Event,
				Payload: in.Payload,
			})
			return err
		})
	if err != nil {
		l.Errorf("broadcast room partially failed: room=%s reqId=%s err=%v", in.Room, in.ReqId, err)
	}
	return &socketpush.BroadcastRoomRes{
		ReqId: in.ReqId,
	}, nil
}
