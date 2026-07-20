package logic

import (
	"context"

	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/common/tool"
	"zero-service/model/gormmodel"
	"zero-service/third_party/extproto"

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
	if l.svcCtx.DevicePointMappingStore == nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "device point mapping model is not initialized")
	}
	mappings, total, err := l.svcCtx.DevicePointMappingStore.FindPage(l.ctx, gormmodel.DevicePointMappingFilter{
		TagStation: in.TagStation,
		Coa:        in.Coa,
		DeviceId:   in.DeviceId,
	}, in.Page, in.PageSize)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "分页查询点位映射失败")
	}
	pbMappings := make([]*ieccaller.PbDevicePointMapping, 0, len(mappings))
	for i := range mappings {
		pbMappings = append(pbMappings, toPbDevicePointMapping(&mappings[i]))
	}
	return &ieccaller.PageListPointMappingRes{
		Mappings: pbMappings,
		Total:    total,
	}, nil
}
