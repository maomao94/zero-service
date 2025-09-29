package logic

import (
	"context"

	"zero-service/facade/streamevent/internal/svc"
	"zero-service/facade/streamevent/streamevent"

	"github.com/zeromicro/go-zero/core/logx"
)

type PushChunkAsduLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPushChunkAsduLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PushChunkAsduLogic {
	return &PushChunkAsduLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 推送 chunk asdu 104协议消息
func (l *PushChunkAsduLogic) PushChunkAsdu(in *streamevent.PushChunkAsduReq) (*streamevent.PushChunkAsduRes, error) {
	// todo: add your logic here and delete this line

	return &streamevent.PushChunkAsduRes{}, nil
}
