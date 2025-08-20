package webhook

import (
	"context"

	"zero-service/app/lalhook/internal/svc"
	"zero-service/app/lalhook/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type OnUpdateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 定时汇报所有group、session的信息
func NewOnUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OnUpdateLogic {
	return &OnUpdateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *OnUpdateLogic) OnUpdate(req *types.OnUpdateRequest) (resp *types.EmptyReply, err error) {
	// todo: add your logic here and delete this line

	return
}
