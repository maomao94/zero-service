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

type QueryPointMappingByIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewQueryPointMappingByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryPointMappingByIdLogic {
	return &QueryPointMappingByIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 根据ID查询点位绑定信息
func (l *QueryPointMappingByIdLogic) QueryPointMappingById(in *ieccaller.QueryPointMappingByIdReq) (*ieccaller.QueryPointMappingByIdRes, error) {
	if l.svcCtx.DevicePointMappingModel == nil {
		return nil, errors.New("device point mapping model is not initialized")
	}
	mapping, err := l.svcCtx.DevicePointMappingModel.FindOne(l.ctx, in.Id)
	if err != nil {
		return nil, err
	}
	pbMapping := &ieccaller.PbDevicePointMapping{}
	if err := copier.CopyWithOption(pbMapping, mapping, copierx.Option); err != nil {
		return nil, err
	}
	return &ieccaller.QueryPointMappingByIdRes{
		Mapping: pbMapping,
	}, nil
}
