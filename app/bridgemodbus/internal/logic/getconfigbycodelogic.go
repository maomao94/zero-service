package logic

import (
	"context"
	"zero-service/common/copierx"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetConfigByCodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetConfigByCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConfigByCodeLogic {
	return &GetConfigByCodeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 根据编码查询详情
func (l *GetConfigByCodeLogic) GetConfigByCode(in *bridgemodbus.GetConfigByCodeReq) (*bridgemodbus.GetConfigByCodeRes, error) {
	cfg, err := l.svcCtx.ModbusSlaveConfigModel.FindOneByModbusCode(l.ctx, in.ModbusCode)
	if err != nil {
		return nil, err
	}
	var pbCfg bridgemodbus.PbModbusConfig
	_ = copier.CopyWithOption(&pbCfg, cfg, copierx.Option)
	return &bridgemodbus.GetConfigByCodeRes{
		Cfg: &pbCfg,
	}, nil
}
