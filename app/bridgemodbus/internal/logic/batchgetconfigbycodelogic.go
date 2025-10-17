package logic

import (
	"context"
	"zero-service/common/copierx"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/Masterminds/squirrel"
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
	builder := l.svcCtx.ModbusSlaveConfigModel.SelectBuilder().Where(squirrel.Eq{"modbus_code": in.ModbusCode})
	list, err := l.svcCtx.ModbusSlaveConfigModel.FindAll(l.ctx, builder)
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
