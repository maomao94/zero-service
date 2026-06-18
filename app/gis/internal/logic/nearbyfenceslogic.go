package logic

import (
	"context"

	"zero-service/app/gis/gis"
	"zero-service/app/gis/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type NearbyFencesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewNearbyFencesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NearbyFencesLogic {
	return &NearbyFencesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// NearbyFences 查询指定点附近 km 范围内的围栏 ID（粗过滤）。
// 依赖 FenceStore 的空间索引实现；若 store 不可用则返回空结果。
func (l *NearbyFencesLogic) NearbyFences(in *gis.NearbyFencesReq) (*gis.NearbyFencesRes, error) {
	if err := ValidatePoints(in.Point); err != nil {
		return nil, err
	}
	if in.Km <= 0 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "km必须大于0")
	}

	fenceIds, err := l.svcCtx.FenceStore.FindNearbyFenceIds(l.ctx, in.Point.Lat, in.Point.Lon, in.Km)
	if err != nil {
		l.Logger.Infof("FenceStore.FindNearbyFenceIds 不可用: %v", err)
		return &gis.NearbyFencesRes{}, nil
	}

	return &gis.NearbyFencesRes{FenceIds: fenceIds}, nil
}
