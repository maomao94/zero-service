package webhook

import (
	"context"

	"zero-service/app/lalhook/internal/svc"
	"zero-service/app/lalhook/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type OnPubStartLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 别人推流到当前节点
func NewOnPubStartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OnPubStartLogic {
	return &OnPubStartLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *OnPubStartLogic) OnPubStart(req *types.OnPubStartRequest) (resp *types.EmptyReply, err error) {
	// todo: add your logic here and delete this line

	return
}
