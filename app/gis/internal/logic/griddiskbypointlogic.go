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

type GridDiskByPointLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGridDiskByPointLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GridDiskByPointLogic {
	return &GridDiskByPointLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GridDiskByPointLogic) GridDiskByPoint(in *gis.GridDiskByPointReq) (*gis.GridDiskRes, error) {
	if err := ValidatePoints(in.Point); err != nil {
		return nil, err
	}

	resolution := int(in.Resolution)
	if resolution <= 0 {
		resolution = 9
	} else if resolution > 15 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "H3分辨率必须为0-15")
	}

	cell, err := EncodeH3Cell(in.Point, resolution)
	if err != nil {
		return nil, err
	}

	k := int(in.K)

	rings, err := h3.GridDiskDistances(cell, k)
	if err != nil {
		return nil, err
	}

	cells := make([]*gis.GridDiskCell, 0)
	for ringNum, ringCells := range rings {
		for _, c := range ringCells {
			cells = append(cells, &gis.GridDiskCell{
				H3Index: c.String(),
				Ring:    uint32(ringNum),
			})
		}
	}

	return &gis.GridDiskRes{
		Origin: cell.String(),
		Cells:  cells,
	}, nil
}
