package webhook

import (
	"context"

	"zero-service/app/lalhook/internal/svc"
	"zero-service/app/lalhook/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type OnPubStopLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 推流停止
func NewOnPubStopLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OnPubStopLogic {
	return &OnPubStopLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *OnPubStopLogic) OnPubStop(req *types.OnPubStopRequest) (resp *types.EmptyReply, err error) {
	// todo: add your logic here and delete this line

	return
}
