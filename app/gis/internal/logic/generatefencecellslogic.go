package logic

import (
	"context"
	"math"

	"zero-service/app/gis/gis"

	"zero-service/app/gis/internal/svc"
	"zero-service/common/gisx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/mmcloughlin/geohash"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
	"github.com/zeromicro/go-zero/core/logx"
)

type GenerateFenceCellsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGenerateFenceCellsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GenerateFenceCellsLogic {
	return &GenerateFenceCellsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GenerateFenceCells 生成覆盖围栏多边形的 geohash cells。
// 算法流程：
//  1. 获取多边形（请求传入或从 store 加载）
//  2. 计算多边形 bbox，以半步长遍历 bbox 内所有候选 geohash
//  3. 精过滤：geohash 格子中心在多边形内 或 格子与多边形边界相交
//  4. 可选：扩展命中格子的 8 邻居（用于模糊匹配场景）
func (l *GenerateFenceCellsLogic) GenerateFenceCells(in *gis.GenFenceCellsReq) (*gis.GenFenceCellsRes, error) {
	// 参数校验与默认值
	precision := int(in.Precision)
	if precision <= 0 {
		precision = 7
	} else if precision > 12 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "geohash精度最大为12")
	}

	// 构建多边形：本接口为纯计算，仅支持请求中直接传入顶点
	if len(in.Points) < 3 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "围栏至少需要3个顶点")
	}
	polygon, err := pbPointToOrbPolygon(in.Points)
	if err != nil {
		l.Logger.Error("构建多边形失败: ", err)
		return nil, err
	}

	// 计算多边形 bbox 作为扫描范围
	bbox := polygon.Bound()
	latMin, latMax := bbox.Min.Y(), bbox.Max.Y()
	lonMin, lonMax := bbox.Min.X(), bbox.Max.X()
	l.Logger.Debugf("围栏边界框: 纬度[%v,%v], 经度[%v,%v]", latMin, latMax, lonMin, lonMax)

	// 计算扫描步长：取格子尺寸的一半，确保不遗漏边界格子
	geohashSet := make(map[string]struct{}, 1024)
	centerLat := (latMin + latMax) / 2
	lonStep, latStep := geohashCellSize(precision, centerLat)
	latStep /= 2
	lonStep /= 2
	epsilon := 1e-8

	// 以半步长遍历 bbox，对每个采样点生成 geohash 并判定是否与围栏相交
	for lat := latMin; lat <= latMax+epsilon; lat += latStep {
		for lon := lonMin; lon <= lonMax+epsilon; lon += lonStep {
			hash := geohash.EncodeWithPrecision(lat, lon, uint(precision))
			if len(hash) != precision {
				l.Logger.Errorf("无效geohash生成: %s（精度不匹配）", hash)
				continue
			}

			// 构造 geohash 格子的矩形多边形，用于相交判定
			box := geohash.BoundingBox(hash)
			cellOrb := orb.Polygon{
				orb.Ring{
					{box.MinLng, box.MinLat},
					{box.MinLng, box.MaxLat},
					{box.MaxLng, box.MaxLat},
					{box.MaxLng, box.MinLat},
					{box.MinLng, box.MinLat},
				},
			}

			// 精过滤：格子中心点在多边形内 或 格子边界与多边形相交
			cLat, cLon := geohash.DecodeCenter(hash)
			isInside := planar.PolygonContains(polygon, orb.Point{cLon, cLat})
			isIntersect := gisx.PolygonIntersect(polygon, cellOrb)

			if isInside || isIntersect {
				geohashSet[hash] = struct{}{}
				l.Logger.Debugf("命中有效格子: %s（中心在内部: %v, 相交: %v）", hash, isInside, isIntersect)

				// 若开启邻居扩展，将命中格子的 8 邻域一并纳入
				if in.IncludeNeighbors {
					for _, neighbor := range geohash.Neighbors(hash) {
						if len(neighbor) == precision {
							geohashSet[neighbor] = struct{}{}
						}
					}
				}
			}
		}
	}

	result := make([]string, 0, len(geohashSet))
	for h := range geohashSet {
		result = append(result, h)
	}

	l.Logger.Infof("生成围栏geohash完成，共%d个格子", len(result))
	return &gis.GenFenceCellsRes{
		Geohashes: result,
	}, nil
}

// computeGeohashCells 计算覆盖多边形的 geohash cells（不含邻居扩展）。
// 与 GenerateFenceCells 逻辑一致，用于 CreateFence/UpdateFence 内部调用。
func computeGeohashCells(polygon orb.Polygon, precision int) []string {
	bbox := polygon.Bound()
	latMin, latMax := bbox.Min.Y(), bbox.Max.Y()
	lonMin, lonMax := bbox.Min.X(), bbox.Max.X()

	geohashSet := make(map[string]struct{}, 1024)
	centerLat := (latMin + latMax) / 2
	lonStep, latStep := geohashCellSize(precision, centerLat)
	latStep /= 2
	lonStep /= 2
	epsilon := 1e-8

	for lat := latMin; lat <= latMax+epsilon; lat += latStep {
		for lon := lonMin; lon <= lonMax+epsilon; lon += lonStep {
			hash := geohash.EncodeWithPrecision(lat, lon, uint(precision))
			if len(hash) != precision {
				continue
			}

			box := geohash.BoundingBox(hash)
			cellOrb := orb.Polygon{
				orb.Ring{
					{box.MinLng, box.MinLat},
					{box.MinLng, box.MaxLat},
					{box.MaxLng, box.MaxLat},
					{box.MaxLng, box.MinLat},
					{box.MinLng, box.MinLat},
				},
			}

			cLat, cLon := geohash.DecodeCenter(hash)
			isInside := planar.PolygonContains(polygon, orb.Point{cLon, cLat})
			isIntersect := gisx.PolygonIntersect(polygon, cellOrb)

			if isInside || isIntersect {
				geohashSet[hash] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(geohashSet))
	for h := range geohashSet {
		result = append(result, h)
	}
	return result
}

// geohashCellSize 根据 geohash 位划分精确计算单个格子的经纬度跨度（单位：度）。
// 返回值：widthDeg 为经度方向跨度，heightDeg 为纬度方向跨度。
func geohashCellSize(precision int, _ float64) (widthDeg, heightDeg float64) {
	totalBits := precision * 5
	lonBits := (totalBits + 1) / 2
	latBits := totalBits / 2
	return 360 / math.Pow(2, float64(lonBits)), 180 / math.Pow(2, float64(latBits))
}
