package logic

import (
	"context"
	"errors"

	"zero-service/app/geo/geo"
	"zero-service/app/geo/internal/svc"

	"github.com/mmcloughlin/geohash"
	"github.com/zeromicro/go-zero/core/logx"
)

type DecodeGeoHashLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDecodeGeoHashLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DecodeGeoHashLogic {
	return &DecodeGeoHashLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 解码 geohash -> 经纬度
func (l *DecodeGeoHashLogic) DecodeGeoHash(in *geo.DecodeGeoHashReq) (*geo.DecodeGeoHashRes, error) {
	if in == nil || in.Geohash == "" {
		return nil, errors.New("参数错误")
	}
	// 中心点
	lat, lon := geohash.DecodeCenter(in.Geohash)
	// bbox
	box := geohash.BoundingBox(in.Geohash)
	return &geo.DecodeGeoHashRes{
		Point: &geo.Point{
			Lat: lat,
			Lon: lon,
		},
		LatMin: box.MinLat,
		LatMax: box.MaxLat,
		LonMin: box.MinLng,
		LonMax: box.MaxLng,
	}, nil
}
