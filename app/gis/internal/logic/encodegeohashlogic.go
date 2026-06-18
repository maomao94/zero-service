package logic

import (
	"context"
	"zero-service/app/gis/gis"

	"zero-service/app/gis/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/mmcloughlin/geohash"
	"github.com/zeromicro/go-zero/core/logx"
)

type EncodeGeoHashLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewEncodeGeoHashLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EncodeGeoHashLogic {
	return &EncodeGeoHashLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// EncodeGeoHash 将经纬度编码为指定精度的 geohash 字符串。
func (l *EncodeGeoHashLogic) EncodeGeoHash(in *gis.EncodeGeoHashReq) (*gis.EncodeGeoHashRes, error) {
	if err := ValidatePoints(in.Point); err != nil {
		return nil, err
	}
	precision := int(in.Precision)
	if precision <= 0 {
		precision = 7
	} else if precision > 12 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "geohash精度最大为12")
	}
	hash := geohash.EncodeWithPrecision(in.Point.Lat, in.Point.Lon, uint(precision))
	return &gis.EncodeGeoHashRes{
		Geohash: hash,
	}, nil
}
