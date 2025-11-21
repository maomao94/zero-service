package logic

import (
	"context"
	"math"

	"zero-service/app/gis/gis"
	"zero-service/app/gis/internal/svc"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/zeromicro/go-zero/core/logx"
)

type RoutePointsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRoutePointsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RoutePointsLogic {
	return &RoutePointsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 计算点集合的最优路径
func (l *RoutePointsLogic) RoutePoints(in *gis.RoutePointsReq) (*gis.RoutePointsRes, error) {
	var err error
	err = ValidatePoints(append([]*gis.Point{in.Start}, in.Points...)...)
	if err != nil {
		return nil, err
	}
	n := len(in.Points)
	// 构造 orb.Point 列表
	points := make([]orb.Point, n)
	for i, p := range in.Points {
		points[i] = orb.Point{p.Lon, p.Lat}
	}
	start := orb.Point{in.Start.Lon, in.Start.Lat}

	// 贪心算法生成初始顺序
	visited := make([]bool, n)
	order := make([]int32, 0, n)
	current := start

	for i := 0; i < n; i++ {
		minDist := math.MaxFloat64
		nextIdx := -1
		for j, p := range points {
			if visited[j] {
				continue
			}
			d := geo.Distance(current, p)
			if d < minDist {
				minDist = d
				nextIdx = j
			}
		}
		if nextIdx == -1 {
			break
		}
		visited[nextIdx] = true
		order = append(order, int32(nextIdx))
		current = points[nextIdx]
	}

	// 2-opt 优化路径
	improved := true
	for improved {
		improved = false
		for i := 0; i < n-1; i++ {
			for j := i + 1; j < n; j++ {
				a := start
				if i > 0 {
					a = points[order[i-1]]
				}
				b := points[order[i]]
				c := points[order[j]]
				d := c
				if j+1 < n {
					d = points[order[j+1]]
				}
				before := geo.Distance(a, b) + geo.Distance(c, d)
				after := geo.Distance(a, c) + geo.Distance(b, d)
				if after < before {
					// 翻转 i..j
					for l, r := i, j; l < r; l, r = l+1, r-1 {
						order[l], order[r] = order[r], order[l]
					}
					improved = true
				}
			}
		}
	}

	// 计算总距离
	totalDist := 0.0
	current = start
	for _, idx := range order {
		next := points[idx]
		totalDist += geo.Distance(current, next)
		current = next
	}

	return &gis.RoutePointsRes{
		VisitOrder:          order,
		TotalDistanceMeters: totalDist,
	}, nil
}
