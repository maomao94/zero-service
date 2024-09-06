package logic

import (
	"context"
	"github.com/Masterminds/squirrel"
	"github.com/duke-git/lancet/v2/validator"
	"github.com/jinzhu/copier"
	"zero-service/zerorpc/internal/svc"
	"zero-service/zerorpc/zerorpc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRegionListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetRegionListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRegionListLogic {
	return &GetRegionListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetRegionListLogic) GetRegionList(in *zerorpc.GetRegionListReq) (*zerorpc.GetRegionListRes, error) {
	parentCode := in.ParentCode
	if validator.IsEmptyString(parentCode) {
		parentCode = "00"
	}
	builder := l.svcCtx.RegionModel.SelectBuilder().Where(squirrel.Eq{"parent_code": parentCode})
	row, err := l.svcCtx.RegionModel.FindAll(l.ctx, builder, "")
	if err != nil {
		return nil, err
	}
	var r []*zerorpc.Region
	copier.Copy(&r, row)
	return &zerorpc.GetRegionListRes{
		Region: r,
	}, nil
}
