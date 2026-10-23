package gateway

import (
	"context"

	"zero-service/openapp/opengtw/internal/svc"
	"zero-service/openapp/opengtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type HealthLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 存活探针（不鉴权）
func NewHealthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HealthLogic {
	return &HealthLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *HealthLogic) Health() (*types.HealthReply, error) {
	return &types.HealthReply{Status: "ok"}, nil
}
