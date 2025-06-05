package common

import (
	"context"
	"github.com/jinzhu/copier"
	"zero-service/zerorpc/zerorpc"

	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRegionListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取区域列表
func NewGetRegionListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRegionListLogic {
	return &GetRegionListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRegionListLogic) GetRegionList(req *types.GetRegionListRequest) (resp *types.GetRegionListReply, err error) {
	res, err := l.svcCtx.ZeroRpcCli.GetRegionList(l.ctx, &zerorpc.GetRegionListReq{ParentCode: req.ParentCode})
	if err != nil {
		return nil, err
	}
	var r []types.Region
	copier.Copy(&r, res.Region)
	return &types.GetRegionListReply{Region: r}, nil
}
