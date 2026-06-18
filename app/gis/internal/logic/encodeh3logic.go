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

// EncodeH3 将经纬度编码为指定分辨率的 H3 六边形索引。
func (l *EncodeH3Logic) EncodeH3(in *gis.EncodeH3Req) (*gis.EncodeH3Res, error) {
	if err := ValidatePoints(in.Point); err != nil {
		return nil, err
	}
	resolution := int(in.Resolution)
	if resolution <= 0 {
		resolution = 9
	} else if resolution > 15 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "h3分辨率必须为0-15")
	}

	latLng := h3.NewLatLng(in.Point.Lat, in.Point.Lon)
	cell, err := h3.LatLngToCell(latLng, resolution)
	if err != nil {
		return nil, err
	}
	return &gis.EncodeH3Res{
		H3Index: cell.String(),
	}, nil
}
