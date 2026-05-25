package logic

import (
	"context"
	"fmt"

	"zero-service/app/gis/gis"

	"zero-service/app/gis/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "points")
	}

	if in.SourceType == in.TargetType {
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

	resPoints := make([]*gis.Point, len(in.Points))
	for i, p := range in.Points {
		res, err := NewTransformCoordLogic(l.ctx, l.svcCtx).TransformCoord(&gis.TransformCoordReq{
			Point:      p,
			SourceType: in.SourceType,
			TargetType: in.TargetType,
		})
		if err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM, err, fmt.Sprintf("坐标转换第 %d 个点失败", i))
		}
		resPoints[i] = res.TransformedPoint
	}

	return &gis.BatchTransformCoordRes{
		TransformedPoints: resPoints,
	}, nil
}
