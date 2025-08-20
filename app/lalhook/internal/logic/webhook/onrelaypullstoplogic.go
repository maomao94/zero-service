package webhook

import (
	"context"

	"zero-service/app/lalhook/internal/svc"
	"zero-service/app/lalhook/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type OnRelayPullStopLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 回源拉流停止
func NewOnRelayPullStopLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OnRelayPullStopLogic {
	return &OnRelayPullStopLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *OnRelayPullStopLogic) OnRelayPullStop(req *types.OnRelayPullStopRequest) (resp *types.EmptyReply, err error) {
	// todo: add your logic here and delete this line

	return
}
