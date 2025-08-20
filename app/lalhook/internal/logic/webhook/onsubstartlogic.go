package webhook

import (
	"context"

	"zero-service/app/lalhook/internal/svc"
	"zero-service/app/lalhook/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type OnSubStartLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 别人从当前节点拉流
func NewOnSubStartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OnSubStartLogic {
	return &OnSubStartLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *OnSubStartLogic) OnSubStart(req *types.OnSubStartRequest) (resp *types.EmptyReply, err error) {
	// todo: add your logic here and delete this line

	return
}
