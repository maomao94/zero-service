package logic

import (
	"context"
	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"
	"zero-service/common/copierx"
	"zero-service/common/tool"
	"zero-service/model/gormmodel"
	"zero-service/third_party/extproto"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
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
	var cfg gormmodel.ModbusSlaveConfig
	err := l.svcCtx.DB.WithContext(l.ctx).Where("modbus_code = ?", in.ModbusCode).First(&cfg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
		}
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_DB, "查询配置失败")
	}
	var pbCfg bridgemodbus.PbModbusConfig
	_ = copier.CopyWithOption(&pbCfg, cfg, copierx.Option)
	return &bridgemodbus.GetConfigByCodeRes{
		Cfg: &pbCfg,
	}, nil
}
