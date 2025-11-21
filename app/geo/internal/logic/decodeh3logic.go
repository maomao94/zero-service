package logic

import (
	"context"
	"errors"

	"zero-service/app/geo/geo"
	"zero-service/app/geo/internal/svc"

	"github.com/uber/h3-go/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type DecodeH3Logic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDecodeH3Logic(ctx context.Context, svcCtx *svc.ServiceContext) *DecodeH3Logic {
	return &DecodeH3Logic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 解码 h3
func (l *DecodeH3Logic) DecodeH3(in *geo.DecodeH3Req) (*geo.DecodeH3Res, error) {
	if in.H3Index == "" {
		return nil, errors.New("参数错误")
	}
	cell := h3.Cell(h3.IndexFromString(in.H3Index))
	latLng, err := h3.CellToLatLng(cell)
	if err != nil {
		return nil, err
	}
	b, err := cell.Boundary()
	if err != nil {
		return nil, err
	}
	boundary := make([]*geo.Point, len(b))
	for i, v := range b {
		boundary[i] = &geo.Point{
			Lat: v.Lat,
			Lon: v.Lng,
		}
	}
	return &geo.DecodeH3Res{
		Center: &geo.Point{
			Lat: latLng.Lat,
			Lon: latLng.Lng,
		},
		Boundary: boundary,
	}, nil
}
