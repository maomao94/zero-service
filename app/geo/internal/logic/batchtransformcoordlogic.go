package logic

import (
	"context"
	"errors"
	"fmt"

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
	if in.Points == nil || len(in.Points) == 0 {
		return nil, errors.New("points cannot be empty")
	}

	if in.SourceType == in.TargetType {
		// 类型相同直接返回原始点
		resPoints := make([]*geo.Point, len(in.Points))
		for i, p := range in.Points {
			resPoints[i] = &geo.Point{
				Lat: p.Lat,
				Lon: p.Lon,
			}
		}
		return &geo.BatchTransformCoordRes{
			TransformedPoints: resPoints,
		}, nil
	}

	// 批量转换
	resPoints := make([]*geo.Point, len(in.Points))
	for i, p := range in.Points {
		res, err := NewTransformCoordLogic(l.ctx, l.svcCtx).TransformCoord(&geo.TransformCoordReq{
			Point:      p,
			SourceType: in.SourceType,
			TargetType: in.TargetType,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to transform point %d: %w", i, err)
		}
		resPoints[i] = res.TransformedPoint
	}

	return &geo.BatchTransformCoordRes{
		TransformedPoints: resPoints,
	}, nil
}
