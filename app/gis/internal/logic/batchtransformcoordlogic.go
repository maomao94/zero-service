package logic

import (
	"context"
	"errors"
	"fmt"

	"zero-service/app/gis/gis"
	"zero-service/app/gis/internal/svc"

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
func (l *BatchTransformCoordLogic) BatchTransformCoord(in *gis.BatchTransformCoordReq) (*gis.BatchTransformCoordRes, error) {
	if in.Points == nil || len(in.Points) == 0 {
		return nil, errors.New("points cannot be empty")
	}

	if in.SourceType == in.TargetType {
		// 类型相同直接返回原始点
		resPoints := make([]*gis.Point, len(in.Points))
		for i, p := range in.Points {
			resPoints[i] = &gis.Point{
				Lat: p.Lat,
				Lon: p.Lon,
			}
		}
		return &gis.BatchTransformCoordRes{
			TransformedPoints: resPoints,
		}, nil
	}

	// 批量转换
	resPoints := make([]*gis.Point, len(in.Points))
	for i, p := range in.Points {
		res, err := NewTransformCoordLogic(l.ctx, l.svcCtx).TransformCoord(&gis.TransformCoordReq{
			Point:      p,
			SourceType: in.SourceType,
			TargetType: in.TargetType,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to transform point %d: %w", i, err)
		}
		resPoints[i] = res.TransformedPoint
	}

	return &gis.BatchTransformCoordRes{
		TransformedPoints: resPoints,
	}, nil
}
