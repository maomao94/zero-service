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

func (l *EncodeGeoHashLogic) EncodeGeoHash(in *gis.EncodeGeoHashReq) (*gis.EncodeGeoHashRes, error) {
	if in.Point == nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "point")
	}
	precision := int(in.Precision)
	if precision <= 0 {
		precision = 7
	}
	hash := geohash.EncodeWithPrecision(in.Point.Lat, in.Point.Lon, uint(precision))
	return &gis.EncodeGeoHashRes{
		Geohash: hash,
	}, nil
}
