package logic

import (
	"context"

	"zero-service/app/lalproxy/internal/svc"
	"zero-service/app/lalproxy/lalproxy"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddIpBlacklistLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAddIpBlacklistLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddIpBlacklistLogic {
	return &AddIpBlacklistLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 增加IP黑名单，加入名单的IP将无法连接本服务
func (l *AddIpBlacklistLogic) AddIpBlacklist(in *lalproxy.AddIpBlacklistReq) (*lalproxy.AddIpBlacklistRes, error) {
	// todo: add your logic here and delete this line

	return &lalproxy.AddIpBlacklistRes{}, nil
}
