package logic

import (
	"context"
	"errors"
	"fmt"

	"zero-service/app/geo/geo"
	"zero-service/app/geo/internal/svc"

	"github.com/uber/h3-go/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type EncodeH3Logic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewEncodeH3Logic(ctx context.Context, svcCtx *svc.ServiceContext) *EncodeH3Logic {
	return &EncodeH3Logic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 编码 h3
func (l *EncodeH3Logic) EncodeH3(in *geo.EncodeH3Req) (*geo.EncodeH3Res, error) {
	if in.Point == nil {
		return nil, errors.New("参数错误")
	}
	if in.Resolution > 15 {
		return nil, fmt.Errorf("h3 resolution must be 0-15")
	}

	latLng := h3.NewLatLng(in.Point.Lat, in.Point.Lon)
	cell, err := h3.LatLngToCell(latLng, int(in.Resolution))
	if err != nil {
		return nil, err
	}
	return &geo.EncodeH3Res{
		H3Index: cell.String(),
	}, nil
}
