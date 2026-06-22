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

type GridDiskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGridDiskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GridDiskLogic {
	return &GridDiskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GridDiskLogic) GridDisk(in *gis.GridDiskReq) (*gis.GridDiskRes, error) {
	if in.H3Index == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "h3_index")
	}

	origin := h3.CellFromString(in.H3Index)
	if !origin.IsValid() {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "无效的 H3 index")
	}

	k := int(in.K)

	rings, err := h3.GridDiskDistances(origin, k)
	if err != nil {
		return nil, err
	}

	cells := make([]*gis.GridDiskCell, 0)
	for ringNum, ringCells := range rings {
		for _, cell := range ringCells {
			cells = append(cells, &gis.GridDiskCell{
				H3Index: cell.String(),
				Ring:    uint32(ringNum),
			})
		}
	}

	return &gis.GridDiskRes{
		Origin: in.H3Index,
		Cells:  cells,
	}, nil
}
