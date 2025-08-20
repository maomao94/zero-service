package webhook

import (
	"context"

	"zero-service/app/lalhook/internal/svc"
	"zero-service/app/lalhook/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type OnRtmpConnectLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 收到 rtmp connect message 信令
func NewOnRtmpConnectLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OnRtmpConnectLogic {
	return &OnRtmpConnectLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *OnRtmpConnectLogic) OnRtmpConnect(req *types.OnRtmpConnectRequest) (resp *types.EmptyReply, err error) {
	// todo: add your logic here and delete this line

	return
}
