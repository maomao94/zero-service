package logic

import (
	"context"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"
	"zero-service/common/gormx"
	"zero-service/common/tool"
	"zero-service/model/gormmodel"
	"zero-service/third_party/extproto"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
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
	if err := validateSaveConfig(in); err != nil {
		return nil, err
	}
	var id string
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
			id = newCfg.Id
		} else {
			// 更新
			exist.SlaveAddress = in.SlaveAddress
			exist.Slave = int64(in.Slave)
			if err := tx.Save(&exist).Error; err != nil {
				return err
			}
			id = exist.Id
		}
		return nil
	})
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "保存Modbus配置失败")
	}

	// 配置变更后按 code 移除旧客户端，下一次请求读取新配置。
	// modbusCode 已由 validateSaveConfig 保证非空，空 code 的文件配置不经过这里。
	l.svcCtx.Manager.RemoveClient(in.ModbusCode)
	return &bridgemodbus.SaveConfigRes{
		Id: id,
	}, nil
}
