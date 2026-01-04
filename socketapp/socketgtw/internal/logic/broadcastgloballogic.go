package logic

import (
	"context"

	"zero-service/socketapp/socketgtw/internal/svc"
	"zero-service/socketapp/socketgtw/socketgtw"

	"github.com/zeromicro/go-zero/core/logx"
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
func (l *BroadcastGlobalLogic) BroadcastGlobal(in *socketgtw.BroadcastGlobalReq) (*socketgtw.BroadcastGlobalRes, error) {
	l.Infof("BroadcastGlobal, event: %s, reqId: %s", in.Event, in.ReqId)
	err := l.svcCtx.SocketServer.BroadcastGlobal(in.Event, string(in.Payload), in.ReqId)
	if err != nil {
		return nil, err
	}
	return &socketgtw.BroadcastGlobalRes{
		ReqId: in.ReqId,
	}, nil
}
