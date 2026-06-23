package logic

import (
	"context"

	"github.com/paulmach/orb"

	"zero-service/app/gis/gis"
	"zero-service/app/gis/internal/svc"
	"zero-service/common/gisx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListFencesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListFencesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListFencesLogic {
	return &ListFencesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ListFences 分页查询围栏列表，支持按名称模糊搜索。
func (l *ListFencesLogic) ListFences(in *gis.ListFencesReq) (*gis.ListFencesRes, error) {
	list, total, err := l.svcCtx.FenceStore.ListFences(l.ctx, in.Page, in.PageSize, in.Name)
	if err != nil {
		return nil, err
	}

	items := make([]*gis.FenceDetail, len(list))
	for i, f := range list {
		items[i] = fenceInfoToDetail(&f)
	}

	return &gis.ListFencesRes{
		List:  items,
		Total: total,
	}, nil
}

// fenceInfoToDetail 将 store 层的 FenceInfo 转换为 pb FenceDetail 响应。
func fenceInfoToDetail(f *gisx.FenceInfo) *gis.FenceDetail {
	exteriorRing := orb.Ring(nil)
	if len(f.Polygon) > 0 {
		exteriorRing = f.Polygon[0]
	}
	points := make([]*gis.Point, len(exteriorRing))
	for i, p := range exteriorRing {
		points[i] = &gis.Point{Lat: p.Y(), Lon: p.X()}
	}
	return &gis.FenceDetail{
		FenceId:          f.FenceId,
		Name:             f.Name,
		Points:           points,
		H3Resolution:     uint32(f.H3Resolution),
		GeohashPrecision: uint32(f.GeohashPrecision),
		H3Cells:          f.H3Cells,
		Geohashes:        f.Geohashes,
		CreatedAt:        f.CreatedAt.UnixMilli(),
		UpdatedAt:        f.UpdatedAt.UnixMilli(),
	}
}
