package logic

import (
	"context"
	"math"

	"zero-service/app/gis/gis"
	"zero-service/app/gis/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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

// RoutePoints 计算从起点出发访问所有点的近似最短路径（开放式 TSP）。
// 算法：
//  1. 最近邻贪心（Nearest Neighbor）生成初始路径，时间复杂度 O(n²)
//  2. 2-opt 局部搜索优化，反复尝试翻转子路径直到无法改进
//
// 适用于中小规模点集（≤500），不保证全局最优但实际效果较好。
func (l *RoutePointsLogic) RoutePoints(in *gis.RoutePointsReq) (*gis.RoutePointsRes, error) {
	if len(in.Points) > 500 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "点数不能超过500")
	}
	var err error
	err = ValidatePoints(append([]*gis.Point{in.Start}, in.Points...)...)
	if err != nil {
		return nil, err
	}
	n := len(in.Points)
	points := make([]orb.Point, n)
	for i, p := range in.Points {
		points[i] = orb.Point{p.Lon, p.Lat}
	}
	start := orb.Point{in.Start.Lon, in.Start.Lat}

	// 阶段一：最近邻贪心 — 每次选择离当前点最近的未访问点
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

	// 阶段二：2-opt 局部优化
	// 对于路径中的每对边 (a→b) 和 (c→d)，若翻转 b..c 段能缩短总距离则执行翻转。
	// 仅在 j+1 < n 时比较，避免开放路径末端的无效边计算。
	improved := true
	for improved {
		improved = false
		for i := 0; i < n-1; i++ {
			for j := i + 1; j < n; j++ {
				if j+1 >= n {
					continue
				}
				a := start
				if i > 0 {
					a = points[order[i-1]]
				}
				b := points[order[i]]
				c := points[order[j]]
				d := points[order[j+1]]
				before := geo.Distance(a, b) + geo.Distance(c, d)
				after := geo.Distance(a, c) + geo.Distance(b, d)
				if after < before {
					for l, r := i, j; l < r; l, r = l+1, r-1 {
						order[l], order[r] = order[r], order[l]
					}
					improved = true
				}
			}
		}
	}

	// 阶段三：沿优化后的顺序累加路径总距离
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
