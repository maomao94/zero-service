package logic

import (
	"context"

	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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
	if l.svcCtx.DevicePointMappingStore == nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "device point mapping model is not initialized")
	}
	mapping, err := l.svcCtx.DevicePointMappingStore.FindOneByTagStationCoaIoa(l.ctx, in.TagStation, in.Coa, in.Ioa)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询点位映射失败")
	}
	return &ieccaller.QueryPointMappingByKeyRes{
		Mapping: toPbDevicePointMapping(mapping),
	}, nil
}
