package logic

import (
	"testing"

	"zero-service/app/gis/gis"
)

// TestBatchTransformCoord_Precision BD09→GCJ02 返回 6 位小数，且与全精度结果误差不超过 0.5e-6 度。
func TestBatchTransformCoord_Precision(t *testing.T) {
	l := NewBatchTransformCoordLogic(t.Context(), nil)
	in := &gis.BatchTransformCoordReq{
		Points:     []*gis.Point{{Lon: 121.4737, Lat: 31.2304}},
		SourceType: gis.CoordType_COORD_TYPE_BD09,
		TargetType: gis.CoordType_COORD_TYPE_GCJ02,
	}

	res, err := l.BatchTransformCoord(in)
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}
	got := res.TransformedPoints[0]

	fullLon, fullLat := doTransformCoord(in.Points[0].Lon, in.Points[0].Lat, in.SourceType, in.TargetType)
	if d := abs(got.Lon - fullLon); d > 5e-7 {
		t.Errorf("lon=%v 与全精度 %v 偏差 %v 超过 6 位小数容差", got.Lon, fullLon, d)
	}
	if d := abs(got.Lat - fullLat); d > 5e-7 {
		t.Errorf("lat=%v 与全精度 %v 偏差 %v 超过 6 位小数容差", got.Lat, fullLat, d)
	}
}

// TestTransformCoord_SameType 同坐标系直通时同样按 6 位小数返回。
func TestTransformCoord_SameType(t *testing.T) {
	l := NewTransformCoordLogic(t.Context(), nil)
	res, err := l.TransformCoord(&gis.TransformCoordReq{
		Point:      &gis.Point{Lon: 120.0919826720555, Lat: 30.420827468961456},
		SourceType: gis.CoordType_COORD_TYPE_WGS84,
		TargetType: gis.CoordType_COORD_TYPE_WGS84,
	})
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}
	assertSixDecimals(t, "transformedPoint.lon", res.TransformedPoint.Lon)
	assertSixDecimals(t, "transformedPoint.lat", res.TransformedPoint.Lat)
}
