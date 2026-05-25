package logic

import (
	"context"
	"fmt"
	"zero-service/app/gis/gis"

	"zero-service/app/gis/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/zeromicro/go-zero/core/logx"
)

type DistanceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDistanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DistanceLogic {
	return &DistanceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DistanceLogic) Distance(in *gis.DistanceReq) (*gis.DistanceRes, error) {
	if err := ValidatePoints(in.A, in.B); err != nil {
		return nil, err
	}
	a := orb.Point{in.A.Lon, in.A.Lat}
	b := orb.Point{in.B.Lon, in.B.Lat}
	distance := geo.Distance(a, b)
	return &gis.DistanceRes{
		Meters: distance,
	}, nil
}

func ValidatePoints(points ...*gis.Point) error {
	if len(points) == 0 {
		return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "points")
	}
	for i, p := range points {
		if p == nil {
			return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, fmt.Sprintf("第 %d 个 point", i))
		}
		if p.Lat < -90 || p.Lat > 90 {
			return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, fmt.Sprintf("第 %d 个 point 的纬度超出范围：lat=%.8f（有效范围 -90~90）", i, p.Lat))
		}
		if p.Lon < -180 || p.Lon > 180 {
			return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, fmt.Sprintf("第 %d 个 point 的经度超出范围：lon=%.8f（有效范围 -180~180）", i, p.Lon))
		}
	}
	return nil
}
