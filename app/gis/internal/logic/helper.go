package logic

import (
	"fmt"
	"math"

	"zero-service/app/gis/gis"
	"zero-service/common/gisx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/paulmach/orb"
	"github.com/uber/h3-go/v4"
)

// ValidateH3Resolution 校验 H3 分辨率，返回 int 类型或 nil。
func ValidateH3Resolution(resolution uint32) (int, error) {
	r := int(resolution)
	if r > 15 {
		return 0, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "H3分辨率必须为0-15")
	}
	return r, nil
}

// ValidateGeoHashPrecision 校验 geohash 精度，返回 int 类型或 nil。
func ValidateGeoHashPrecision(precision uint32) (int, error) {
	p := int(precision)
	if p < 1 || p > 12 {
		return 0, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "geohash精度必须为1-12")
	}
	return p, nil
}

// EncodeH3Cell 将经纬度编码为 H3 cell。
func EncodeH3Cell(point *gis.Point, resolution int) (h3.Cell, error) {
	latLng := h3.NewLatLng(point.Lat, point.Lon)
	return h3.LatLngToCell(latLng, resolution)
}

// ValidatePoints 批量校验 pb Point 列表：非空、非 nil、经纬度范围合法。
func ValidatePoints(points ...*gis.Point) error {
	if len(points) == 0 {
		return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "points")
	}
	for i, p := range points {
		if p == nil {
			return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, fmt.Sprintf("第 %d 个 point", i))
		}
		if err := gisx.ValidateCoordinate(p.Lat, p.Lon, i); err != nil {
			return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, err.Error())
		}
	}
	return nil
}

// pbPointToOrbPolygon 将 pb Point 切片转换为 orb.Polygon（单外环，无洞）。
// 步骤：校验点数 → 坐标范围检查 → 构建 ring → 自动闭合。
func pbPointToOrbPolygon(points []*gis.Point) (orb.Polygon, error) {
	if len(points) < 3 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "多边形至少需要3个点")
	}

	var ring orb.Ring
	for i, p := range points {
		if p.Lon < -180 || p.Lon > 180 || p.Lat < -90 || p.Lat > 90 {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID,
				fmt.Sprintf("第 %d 个点经纬度超出有效范围（经度-180~180，纬度-90~90）", i))
		}
		ring = append(ring, orb.Point{p.Lon, p.Lat})
	}

	// 自动闭合：首尾点不一致时追加首点
	first, last := ring[0], ring[len(ring)-1]
	const epsilon = 1e-8
	if math.Abs(first[0]-last[0]) > epsilon || math.Abs(first[1]-last[1]) > epsilon {
		ring = append(ring, first)
	}

	return orb.Polygon{ring}, nil
}
