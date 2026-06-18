package logic

import (
	"context"
	"fmt"
	"zero-service/app/gis/gis"

	"zero-service/app/gis/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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

// TransformCoord 单点坐标系转换（WGS84 / GCJ02 / BD09 互转）。
func (l *TransformCoordLogic) TransformCoord(in *gis.TransformCoordReq) (*gis.TransformCoordRes, error) {
	if err := l.validateReq(in); err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, err, "坐标转换请求参数无效")
	}
	if in.SourceType == in.TargetType {
		return &gis.TransformCoordRes{
			TransformedPoint: in.Point,
		}, nil
	}

	lon := in.Point.Lon
	lat := in.Point.Lat

	transformedLon, transformedLat := doTransformCoord(lon, lat, in.SourceType, in.TargetType)

	return &gis.TransformCoordRes{
		TransformedPoint: &gis.Point{
			Lat: transformedLat,
			Lon: transformedLon,
		},
	}, nil
}

func (l *TransformCoordLogic) validateReq(in *gis.TransformCoordReq) error {
	if in.Point == nil {
		return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "point")
	}
	if in.Point.Lat < -90 || in.Point.Lat > 90 {
		return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, fmt.Sprintf("invalid lat: %v (valid -90~90)", in.Point.Lat))
	}
	if in.Point.Lon < -180 || in.Point.Lon > 180 {
		return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, fmt.Sprintf("invalid lon: %v (valid -180~180)", in.Point.Lon))
	}
	sourceVal := uint32(in.SourceType)
	if sourceVal < 1 || sourceVal > 3 {
		return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, fmt.Sprintf("invalid source_type: %v (only support 1=WGS84, 2=GCJ02, 3=BD09)", sourceVal))
	}
	targetVal := uint32(in.TargetType)
	if targetVal < 1 || targetVal > 3 {
		return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, fmt.Sprintf("invalid target_type: %v (only support 1=WGS84, 2=GCJ02, 3=BD09)", targetVal))
	}
	return nil
}

// doTransformCoord 执行坐标系转换。
// 转换链：WGS84 ↔ GCJ02 ↔ BD09。WGS84 与 BD09 之间需中转 GCJ02。
func doTransformCoord(lon, lat float64, source, target gis.CoordType) (float64, float64) {
	switch {
	case source == gis.CoordType_COORD_TYPE_WGS84 && target == gis.CoordType_COORD_TYPE_GCJ02:
		return coordtransform.WGS84toGCJ02(lon, lat)
	case source == gis.CoordType_COORD_TYPE_GCJ02 && target == gis.CoordType_COORD_TYPE_WGS84:
		return coordtransform.GCJ02toWGS84(lon, lat)
	case source == gis.CoordType_COORD_TYPE_GCJ02 && target == gis.CoordType_COORD_TYPE_BD09:
		return coordtransform.GCJ02toBD09(lon, lat)
	case source == gis.CoordType_COORD_TYPE_BD09 && target == gis.CoordType_COORD_TYPE_GCJ02:
		return coordtransform.BD09toGCJ02(lon, lat)
	case source == gis.CoordType_COORD_TYPE_WGS84 && target == gis.CoordType_COORD_TYPE_BD09:
		gcjLon, gcjLat := coordtransform.WGS84toGCJ02(lon, lat)
		return coordtransform.GCJ02toBD09(gcjLon, gcjLat)
	case source == gis.CoordType_COORD_TYPE_BD09 && target == gis.CoordType_COORD_TYPE_WGS84:
		gcjLon, gcjLat := coordtransform.BD09toGCJ02(lon, lat)
		return coordtransform.GCJ02toWGS84(gcjLon, gcjLat)
	}
	return lon, lat
}
