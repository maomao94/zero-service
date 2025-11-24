package logic

import (
	"context"
	"errors"
	"fmt"

	"zero-service/app/gis/gis"
	"zero-service/app/gis/internal/svc"

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

// 批量计算两点之间的距离（米）
func (l *BatchDistanceLogic) BatchDistance(in *gis.BatchDistanceReq) (*gis.BatchDistanceRes, error) {
	if len(in.Pairs) == 0 {
		return nil, errors.New("pairs 不能为空")
	}

	meters := make([]float64, len(in.Pairs))
	for i, pair := range in.Pairs {
		if err := ValidatePoints(pair.A, pair.B); err != nil {
			return nil, fmt.Errorf("第 %d 个点对错误: %w", i, err)
		}
		a := orb.Point{pair.A.Lon, pair.A.Lat}
		b := orb.Point{pair.B.Lon, pair.B.Lat}
		meters[i] = geo.Distance(a, b)
	}

	return &gis.BatchDistanceRes{
		Meters: meters,
	}, nil
}
