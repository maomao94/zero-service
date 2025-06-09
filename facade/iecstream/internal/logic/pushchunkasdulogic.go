package logic

import (
	"context"

	"zero-service/facade/iecstream/iecstream"
	"zero-service/facade/iecstream/internal/svc"

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

func (l *PushChunkAsduLogic) PushChunkAsdu(in *iecstream.PushChunkAsduReq) (*iecstream.PushChunkAsduRes, error) {
	logx.Infof("PushChunkAsduReq: %v", in)
	return &iecstream.PushChunkAsduRes{}, nil
}
