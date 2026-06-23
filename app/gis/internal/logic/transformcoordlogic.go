package logic

import (
	"context"

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
	if err := ValidatePoints(in.Point); err != nil {
		return err
	}
	if err := validateCoordType(in.SourceType); err != nil {
		return err
	}
	return validateCoordType(in.TargetType)
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
