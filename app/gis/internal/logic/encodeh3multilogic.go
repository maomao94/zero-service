package logic

import (
	"context"
	"zero-service/app/gis/gis"

	"zero-service/app/gis/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type EncodeH3MultiLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewEncodeH3MultiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EncodeH3MultiLogic {
	return &EncodeH3MultiLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *EncodeH3MultiLogic) EncodeH3Multi(in *gis.EncodeH3MultiReq) (*gis.EncodeH3MultiRes, error) {
	if err := ValidatePoints(in.Point); err != nil {
		return nil, err
	}
	if len(in.Resolutions) == 0 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "resolutions")
	}

	items := make([]*gis.H3Index, 0, len(in.Resolutions))
	for _, resolution := range in.Resolutions {
		r, err := ValidateH3Resolution(resolution)
		if err != nil {
			return nil, err
		}

		cell, err := EncodeH3Cell(in.Point, r)
		if err != nil {
			return nil, err
		}

		items = append(items, &gis.H3Index{
			Resolution: resolution,
			H3Index:    cell.String(),
		})
	}

	return &gis.EncodeH3MultiRes{
		H3Indexes: items,
	}, nil
}
