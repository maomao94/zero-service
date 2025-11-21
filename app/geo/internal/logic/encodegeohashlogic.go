package logic

import (
	"context"
	"errors"

	"zero-service/app/geo/geo"
	"zero-service/app/geo/internal/svc"

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
func (l *EncodeGeoHashLogic) EncodeGeoHash(in *geo.EncodeGeoHashReq) (*geo.EncodeGeoHashRes, error) {
	if in.Point == nil {
		return nil, errors.New("参数错误")
	}
	// 默认精度 7
	precision := int(in.Precision)
	if precision <= 0 {
		precision = 7
	}

	hash := geohash.EncodeWithPrecision(in.Point.Lat, in.Point.Lon, uint(precision))

	return &geo.EncodeGeoHashRes{
		Geohash: hash,
	}, nil
}
