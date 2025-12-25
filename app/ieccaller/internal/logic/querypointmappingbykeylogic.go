package logic

import (
	"context"
	"errors"

	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/common/copierx"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type QueryPointMappingByKeyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewQueryPointMappingByKeyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryPointMappingByKeyLogic {
	return &QueryPointMappingByKeyLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 根据tagStation、coa、ioa查询点位绑定信息
func (l *QueryPointMappingByKeyLogic) QueryPointMappingByKey(in *ieccaller.QueryPointMappingByKeyReq) (*ieccaller.QueryPointMappingByKeyRes, error) {
	if l.svcCtx.DevicePointMappingModel == nil {
		return nil, errors.New("device point mapping model is not initialized")
	}
	mapping, err := l.svcCtx.DevicePointMappingModel.FindOneByTagStationCoaIoa(l.ctx, in.TagStation, in.Coa, in.Ioa)
	if err != nil {
		return nil, err
	}
	pbMapping := &ieccaller.PbDevicePointMapping{}
	if err := copier.CopyWithOption(pbMapping, mapping, copierx.Option); err != nil {
		return nil, err
	}
	return &ieccaller.QueryPointMappingByKeyRes{
		Mapping: pbMapping,
	}, nil
}
