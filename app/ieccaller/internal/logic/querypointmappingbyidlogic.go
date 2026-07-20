package logic

import (
	"context"

	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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
	if l.svcCtx.DevicePointMappingStore == nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "device point mapping model is not initialized")
	}
	mapping, err := l.svcCtx.DevicePointMappingStore.FindOne(l.ctx, in.Id)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询点位映射失败")
	}
	return &ieccaller.QueryPointMappingByIdRes{
		Mapping: toPbDevicePointMapping(mapping),
	}, nil
}
