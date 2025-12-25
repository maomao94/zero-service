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

type PageListPointMappingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPageListPointMappingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PageListPointMappingLogic {
	return &PageListPointMappingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 分页查询点位绑定列表
func (l *PageListPointMappingLogic) PageListPointMapping(in *ieccaller.PageListPointMappingReq) (*ieccaller.PageListPointMappingRes, error) {
	if l.svcCtx.DevicePointMappingModel == nil {
		return nil, errors.New("device point mapping model is not initialized")
	}
	builder := l.svcCtx.DevicePointMappingModel.SelectBuilder()
	if in.TagStation != "" {
		builder = builder.Where("tag_station = ?", in.TagStation)
	}
	if in.Coa > 0 {
		builder = builder.Where("coa = ?", in.Coa)
	}
	if in.DeviceId != "" {
		builder = builder.Where("device_id = ?", in.DeviceId)
	}
	mappings, total, err := l.svcCtx.DevicePointMappingModel.FindPageListByPageWithTotal(l.ctx, builder, in.Page, in.PageSize, "id desc")
	if err != nil {
		return nil, err
	}
	pbMappings := make([]*ieccaller.PbDevicePointMapping, 0, len(mappings))
	for _, mapping := range mappings {
		pbMapping := &ieccaller.PbDevicePointMapping{}
		if err := copier.CopyWithOption(pbMapping, mapping, copierx.Option); err != nil {
			return nil, err
		}
		pbMappings = append(pbMappings, pbMapping)
	}
	return &ieccaller.PageListPointMappingRes{
		Mappings: pbMappings,
		Total:    total,
	}, nil
}
