package logic

import (
	"context"
	"errors"
	"fmt"
	"zero-service/app/gis/gis"

	"zero-service/app/gis/internal/svc"

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

// 计算两个点之间的距离（米）
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

// ValidatePoints 验证 []*Point 参数合法性
func ValidatePoints(points ...*gis.Point) error {
	if len(points) == 0 {
		return errors.New("points 不能为空")
	}
	for i, p := range points {
		if p == nil {
			return fmt.Errorf("第 %d 个 point 为空", i)
		}
		//if math.IsNaN(p.Lat) || math.IsInf(p.Lat, 0) {
		//	return fmt.Errorf("第 %d 个 point 的 Lat 非法", i)
		//}
		//if math.IsNaN(p.Lon) || math.IsInf(p.Lon, 0) {
		//	return fmt.Errorf("第 %d 个 point 的 Lon 非法", i)
		//}
		if p.Lat < -90 || p.Lat > 90 {
			return fmt.Errorf("第 %d 个 point 的纬度超出范围：lat=%.8f（有效范围 -90~90）", i, p.Lat)
		}
		if p.Lon < -180 || p.Lon > 180 {
			return fmt.Errorf("第 %d 个 point 的经度超出范围：lon=%.8f（有效范围 -180~180）", i, p.Lon)
		}
	}
	return nil
}
