package logic

import (
	"context"
	"errors"
	"fmt"

	"zero-service/app/geo/geo"
	"zero-service/app/geo/internal/svc"

	"github.com/qichengzx/coordtransform"
	"github.com/zeromicro/go-zero/core/logx"
)

type TransformCoordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTransformCoordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TransformCoordLogic {
	return &TransformCoordLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 单个坐标转换
func (l *TransformCoordLogic) TransformCoord(in *geo.TransformCoordReq) (*geo.TransformCoordRes, error) {
	if err := l.validateReq(in); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	if in.SourceType == in.TargetType {
		return &geo.TransformCoordRes{
			TransformedPoint: in.Point,
		}, nil
	}

	lon := in.Point.Lon
	lat := in.Point.Lat

	transformedLon, transformedLat := doTransformCoord(lon, lat, in.SourceType, in.TargetType)

	return &geo.TransformCoordRes{
		TransformedPoint: &geo.Point{
			Lat: transformedLat,
			Lon: transformedLon,
		},
	}, nil
}

func (l *TransformCoordLogic) validateReq(in *geo.TransformCoordReq) error {
	if in.Point == nil {
		return errors.New("point cannot be nil")
	}

	// 校验经纬度范围（避免明显非法值）
	if in.Point.Lat < -90 || in.Point.Lat > 90 {
		return fmt.Errorf("invalid lat: %v (must be between -90 and 90)", in.Point.Lat)
	}
	if in.Point.Lon < -180 || in.Point.Lon > 180 {
		return fmt.Errorf("invalid lon: %v (must be between -180 and 180)", in.Point.Lon)
	}

	sourceVal := uint32(in.SourceType)
	if sourceVal < 1 || sourceVal > 3 {
		return fmt.Errorf("invalid source_type: %v (only support 1=WGS84, 2=GCJ02, 3=BD09)", sourceVal)
	}

	targetVal := uint32(in.TargetType)
	if targetVal < 1 || targetVal > 3 {
		return fmt.Errorf("invalid target_type: %v (only support 1=WGS84, 2=GCJ02, 3=BD09)", targetVal)
	}

	return nil
}

func doTransformCoord(lon, lat float64, source, target geo.CoordType) (float64, float64) {
	switch {
	// WGS84(1) ↔ GCJ02(2)
	case source == geo.CoordType_COORD_TYPE_WGS84 && target == geo.CoordType_COORD_TYPE_GCJ02:
		return coordtransform.WGS84toGCJ02(lon, lat)
	case source == geo.CoordType_COORD_TYPE_GCJ02 && target == geo.CoordType_COORD_TYPE_WGS84:
		return coordtransform.GCJ02toWGS84(lon, lat)

	// GCJ02(2) ↔ BD09(3)
	case source == geo.CoordType_COORD_TYPE_GCJ02 && target == geo.CoordType_COORD_TYPE_BD09:
		return coordtransform.GCJ02toBD09(lon, lat)
	case source == geo.CoordType_COORD_TYPE_BD09 && target == geo.CoordType_COORD_TYPE_GCJ02:
		return coordtransform.BD09toGCJ02(lon, lat)

	// WGS84(1) ↔ BD09(3)（中转GCJ02）
	case source == geo.CoordType_COORD_TYPE_WGS84 && target == geo.CoordType_COORD_TYPE_BD09:
		gcjLon, gcjLat := coordtransform.WGS84toGCJ02(lon, lat)
		return coordtransform.GCJ02toBD09(gcjLon, gcjLat)
	case source == geo.CoordType_COORD_TYPE_BD09 && target == geo.CoordType_COORD_TYPE_WGS84:
		gcjLon, gcjLat := coordtransform.BD09toGCJ02(lon, lat)
		return coordtransform.GCJ02toWGS84(gcjLon, gcjLat)
	}
	return lon, lat
}
