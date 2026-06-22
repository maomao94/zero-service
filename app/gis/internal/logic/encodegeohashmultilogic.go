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

type EncodeGeoHashMultiLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewEncodeGeoHashMultiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EncodeGeoHashMultiLogic {
	return &EncodeGeoHashMultiLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *EncodeGeoHashMultiLogic) EncodeGeoHashMulti(in *gis.EncodeGeoHashMultiReq) (*gis.EncodeGeoHashMultiRes, error) {
	if err := ValidatePoints(in.Point); err != nil {
		return nil, err
	}
	if len(in.Precisions) == 0 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "precisions")
	}

	items := make([]*gis.GeoHashIndex, 0, len(in.Precisions))
	for _, precision := range in.Precisions {
		p, err := ValidateGeoHashPrecision(precision)
		if err != nil {
			return nil, err
		}
		hash := geohash.EncodeWithPrecision(in.Point.Lat, in.Point.Lon, uint(p))
		items = append(items, &gis.GeoHashIndex{
			Precision: precision,
			Geohash:   hash,
		})
	}

	return &gis.EncodeGeoHashMultiRes{
		Geohashes: items,
	}, nil
}
