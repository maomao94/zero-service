package logic

import (
	"context"

	"zero-service/app/gis/gis"
	"zero-service/app/gis/internal/svc"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
)

type PointsWithinRadiusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPointsWithinRadiusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PointsWithinRadiusLogic {
	return &PointsWithinRadiusLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取半径内的点
func (l *PointsWithinRadiusLogic) PointsWithinRadius(in *gis.PointsWithinRadiusReq) (*gis.PointsWithinRadiusRes, error) {
	// 校验中心点和点列表
	if err := ValidatePoints(append([]*gis.Point{in.Center}, in.Points...)...); err != nil {
		return nil, err
	}
	center := orb.Point{in.Center.Lon, in.Center.Lat}
	//hitIndexs := make([]int32, 0)
	//for i, p := range in.Points {
	//	orbP := orb.Point{p.Lon, p.Lat}
	//	distance := geo.Distance(center, orbP)
	//	if distance <= in.RadiusMeters {
	//		hitIndexs = append(hitIndexs, int32(i))
	//	}
	//}
	type indexedPoint struct {
		idx int32
		pt  orb.Point
	}
	hits, err := mr.MapReduce[indexedPoint, int32, []int32](
		func(source chan<- indexedPoint) { // generate
			for i, p := range in.Points {
				source <- indexedPoint{
					idx: int32(i),
					pt:  orb.Point{p.Lon, p.Lat},
				}
			}
		},
		func(p indexedPoint, writer mr.Writer[int32], cancel func(error)) { // mapper
			if geo.Distance(center, p.pt) <= in.RadiusMeters {
				writer.Write(p.idx)
			}
		}, func(pipe <-chan int32, writer mr.Writer[[]int32], cancel func(error)) {
			var result []int32
			for idx := range pipe {
				result = append(result, idx)
			}
			writer.Write(result)
		},
		mr.WithWorkers(64),
	)
	if err != nil {
		return nil, err
	}
	return &gis.PointsWithinRadiusRes{
		HitIndexes: hits,
	}, nil
}
