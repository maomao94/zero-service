package logic

import (
	"context"
	"errors"
	"zero-service/app/gis/gis"

	"zero-service/app/gis/internal/svc"

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

// 编码 geohash
func (l *EncodeGeoHashLogic) EncodeGeoHash(in *gis.EncodeGeoHashReq) (*gis.EncodeGeoHashRes, error) {
	if in.Point == nil {
		return nil, errors.New("参数错误")
	}
	// 默认精度 9
	precision := int(in.Precision)
	if precision <= 0 {
		precision = 7
	}

	hash := geohash.EncodeWithPrecision(in.Point.Lat, in.Point.Lon, uint(precision))

	return &gis.EncodeGeoHashRes{
		Geohash: hash,
	}, nil
}
