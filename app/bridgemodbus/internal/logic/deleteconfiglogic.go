package logic

import (
	"context"
	"zero-service/common/tool"
	"zero-service/model/gormmodel"
	"zero-service/third_party/extproto"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteConfigLogic {
	return &DeleteConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 删除配置（支持批量）
func (l *DeleteConfigLogic) DeleteConfig(in *bridgemodbus.DeleteConfigReq) (*bridgemodbus.DeleteConfigRes, error) {
	err := l.svcCtx.DB.WithContext(l.ctx).Delete(&gormmodel.ModbusSlaveConfig{}, in.Ids).Error
	if err != nil {
		logx.Errorf("Batch delete failed, ids=%v, err=%v", in.Ids, err)
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_DB, "批量删除配置失败")
	}
	return &bridgemodbus.DeleteConfigRes{}, nil
}
