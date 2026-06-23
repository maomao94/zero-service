package logic

import (
	"context"

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

// BatchTransformCoord 批量坐标系转换，逐点调用 doTransformCoord。
// 若 source == target 直接拷贝返回，避免无意义计算。
func (l *BatchTransformCoordLogic) BatchTransformCoord(in *gis.BatchTransformCoordReq) (*gis.BatchTransformCoordRes, error) {
	if len(in.Points) == 0 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "points")
	}
	if err := ValidatePoints(in.Points...); err != nil {
		return nil, err
	}
	if err := validateCoordType(in.SourceType); err != nil {
		return nil, err
	}
	if err := validateCoordType(in.TargetType); err != nil {
		return nil, err
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
		lon, lat := doTransformCoord(p.Lon, p.Lat, in.SourceType, in.TargetType)
		resPoints[i] = &gis.Point{Lat: lat, Lon: lon}
	}

	return &gis.BatchTransformCoordRes{
		TransformedPoints: resPoints,
	}, nil
}
