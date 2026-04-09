package logic

import (
	"context"
	"zero-service/common/copierx"
	"zero-service/model/gormmodel"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type BatchGetConfigByCodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchGetConfigByCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetConfigByCodeLogic {
	return &BatchGetConfigByCodeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 根据编码数组查询详情
func (l *BatchGetConfigByCodeLogic) BatchGetConfigByCode(in *bridgemodbus.BatchGetConfigByCodeReq) (*bridgemodbus.BatchGetConfigByCodeRes, error) {
	var list []gormmodel.ModbusSlaveConfig
	err := l.svcCtx.DB.WithContext(l.ctx).Where("modbus_code IN ?", in.ModbusCode).Find(&list).Error
	if err != nil {
		return nil, err
	}
	var configs []*bridgemodbus.PbModbusConfig
	for _, cfg := range list {
		var pbCfg bridgemodbus.PbModbusConfig
		_ = copier.CopyWithOption(&pbCfg, cfg, copierx.Option)
		configs = append(configs, &pbCfg)
	}
	return &bridgemodbus.BatchGetConfigByCodeRes{
		Cfg: configs,
	}, nil
}
