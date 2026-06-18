package logic

import (
	"context"
	"fmt"

	"zero-service/app/gis/gis"

	"zero-service/app/gis/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/zeromicro/go-zero/core/logx"
)

type BatchDistanceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchDistanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchDistanceLogic {
	return &BatchDistanceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// BatchDistance 批量计算点对之间的大圆距离（Haversine 公式，单位：米）。
func (l *BatchDistanceLogic) BatchDistance(in *gis.BatchDistanceReq) (*gis.BatchDistanceRes, error) {
	if len(in.Pairs) == 0 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "pairs")
	}
	meters := make([]float64, len(in.Pairs))
	for i, pair := range in.Pairs {
		if err := ValidatePoints(pair.A, pair.B); err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, err, fmt.Sprintf("第 %d 个点对错误", i))
		}
		a := orb.Point{pair.A.Lon, pair.A.Lat}
		b := orb.Point{pair.B.Lon, pair.B.Lat}
		meters[i] = geo.Distance(a, b)
	}

	return &gis.BatchDistanceRes{
		Meters: meters,
	}, nil
}
