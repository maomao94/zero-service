package logic

import (
	"context"

	"zero-service/app/gis/gis"
	"zero-service/app/gis/internal/svc"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateFenceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateFenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateFenceLogic {
	return &CreateFenceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CreateFence 新增电子围栏。
// 流程：校验参数 → 构建多边形 → 计算 H3 + geohash cells → 生成 ID → 持久化。
func (l *CreateFenceLogic) CreateFence(in *gis.CreateFenceReq) (*gis.CreateFenceRes, error) {
	polygon, err := pbPointToOrbPolygon(in.Points)
	if err != nil {
		return nil, err
	}

	resolution, err := resolveH3Resolution(in.H3Resolution)
	if err != nil {
		return nil, err
	}
	geohashPrecision, err := resolveGeohashPrecision(in.GeohashPrecision)
	if err != nil {
		return nil, err
	}

	cellStrings, geohashes, err := computeFenceCells(polygon, resolution, geohashPrecision)
	if err != nil {
		return nil, err
	}

	fenceId, err := tool.SimpleUUID()
	if err != nil {
		return nil, err
	}

	if err := l.svcCtx.FenceStore.CreateFence(l.ctx, fenceId, in.Name, polygon, resolution, geohashPrecision, cellStrings, geohashes); err != nil {
		l.Logger.Errorf("创建围栏失败, err=%v", err)
		return nil, err
	}

	return &gis.CreateFenceRes{
		FenceId:   fenceId,
		H3Cells:   cellStrings,
		Geohashes: geohashes,
	}, nil
}
