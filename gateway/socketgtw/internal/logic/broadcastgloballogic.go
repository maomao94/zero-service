package logic

import (
	"context"

	"zero-service/gateway/socketgtw/internal/svc"
	"zero-service/gateway/socketgtw/socketgtw"

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
	err := l.svcCtx.SocketServer.BroadcastGlobal(in.Event, string(in.Payload), in.ReqId)
	if err != nil {
		return nil, err
	}
	return &socketgtw.BroadcastGlobalRes{}, nil
}
