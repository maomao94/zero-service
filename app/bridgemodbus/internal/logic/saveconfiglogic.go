package logic

import (
	"context"

	"zero-service/common/gormx"
	"zero-service/model/gormmodel"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SaveConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSaveConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SaveConfigLogic {
	return &SaveConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 保存（新增或更新）配置
func (l *SaveConfigLogic) SaveConfig(in *bridgemodbus.SaveConfigReq) (*bridgemodbus.SaveConfigRes, error) {
	var id int64
	err := l.svcCtx.DB.WithContext(l.ctx).Transact(func(tx *gormx.DB) error {
		var exist gormmodel.ModbusSlaveConfig
		err := tx.Where("modbus_code = ?", in.ModbusCode).First(&exist).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 新增
			newCfg := &gormmodel.ModbusSlaveConfig{
				ModbusCode:   in.ModbusCode,
				SlaveAddress: in.SlaveAddress,
				Slave:        int64(in.Slave),
			}
			if err := tx.Create(newCfg).Error; err != nil {
				return err
			}
			id = int64(newCfg.Id)
		} else {
			// 更新
			exist.SlaveAddress = in.SlaveAddress
			exist.Slave = int64(in.Slave)
			if err := tx.Save(&exist).Error; err != nil {
				return err
			}
			id = int64(exist.Id)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &bridgemodbus.SaveConfigRes{
		Id: id,
	}, nil
}
