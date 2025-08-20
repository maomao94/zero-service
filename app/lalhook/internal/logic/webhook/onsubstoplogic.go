package webhook

import (
	"context"

	"zero-service/app/lalhook/internal/svc"
	"zero-service/app/lalhook/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type OnSubStopLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 拉流停止
func NewOnSubStopLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OnSubStopLogic {
	return &OnSubStopLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *OnSubStopLogic) OnSubStop(req *types.OnSubStopRequest) (resp *types.EmptyReply, err error) {
	// todo: add your logic here and delete this line

	return
}
