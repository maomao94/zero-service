package logic

import (
	"context"
	"zero-service/app/gis/gis"

	"zero-service/app/gis/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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

func (l *DecodeH3Logic) DecodeH3(in *gis.DecodeH3Req) (*gis.DecodeH3Res, error) {
	if in.H3Index == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "h3_index")
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
	boundary := make([]*gis.Point, len(b))
	for i, v := range b {
		boundary[i] = &gis.Point{
			Lat: v.Lat,
			Lon: v.Lng,
		}
	}
	return &gis.DecodeH3Res{
		Center: &gis.Point{
			Lat: latLng.Lat,
			Lon: latLng.Lng,
		},
		Boundary: boundary,
	}, nil
}
