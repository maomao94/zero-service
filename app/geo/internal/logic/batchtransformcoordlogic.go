package logic

import (
	"context"

	"zero-service/app/geo/geo"
	"zero-service/app/geo/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchTransformCoordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchTransformCoordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchTransformCoordLogic {
	return &BatchTransformCoordLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 批量坐标转换
func (l *BatchTransformCoordLogic) BatchTransformCoord(in *geo.BatchTransformCoordReq) (*geo.BatchTransformCoordRes, error) {
	// todo: add your logic here and delete this line

	return &geo.BatchTransformCoordRes{}, nil
}
