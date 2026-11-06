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
	if err := validateConfigBatch("ids", in.GetIds(), 36); err != nil {
		return nil, err
	}

	// 先记录待删除配置的 modbusCode，删除成功后同步关闭对应客户端。
	var modbusCodes []string
	err := l.svcCtx.DB.WithContext(l.ctx).
		Model(&gormmodel.ModbusSlaveConfig{}).
		Where("id IN ?", in.Ids).
		Pluck("modbus_code", &modbusCodes).Error
	if err != nil {
		logx.Errorf("Query modbus_code failed, ids=%v, err=%v", in.Ids, err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询待删除配置失败")
	}

	err = l.svcCtx.DB.WithContext(l.ctx).Delete(&gormmodel.ModbusSlaveConfig{}, in.Ids).Error
	if err != nil {
		logx.Errorf("Batch delete failed, ids=%v, err=%v", in.Ids, err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "批量删除配置失败")
	}

	for _, modbusCode := range modbusCodes {
		// 空 code 使用文件配置，不能误清默认客户端。
		if modbusCode != "" {
			l.svcCtx.Manager.RemoveClient(modbusCode)
		}
	}
	return &bridgemodbus.DeleteConfigRes{}, nil
}
