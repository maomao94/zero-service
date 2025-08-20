package webhook

import (
	"context"

	"zero-service/app/lalhook/internal/svc"
	"zero-service/app/lalhook/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type OnServerStartLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 服务启动时
func NewOnServerStartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OnServerStartLogic {
	return &OnServerStartLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *OnServerStartLogic) OnServerStart(req *types.OnServerStartRequest) (resp *types.EmptyReply, err error) {
	// todo: add your logic here and delete this line

	return
}
