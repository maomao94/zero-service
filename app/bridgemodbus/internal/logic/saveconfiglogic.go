package logic

import (
	"context"
	"zero-service/model"

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
	// 查找现有配置
	exist, err := l.svcCtx.ModbusSlaveConfigModel.FindOneByModbusCode(l.ctx, in.ModbusCode)
	if err != nil && err != model.ErrNotFound {
		return nil, err
	}

	// 如果配置存在，则更新
	if exist != nil {
		exist.SlaveAddress = in.SlaveAddress
		exist.Slave = int64(in.Slave)
		_, err = l.svcCtx.ModbusSlaveConfigModel.Update(l.ctx, nil, exist)
		if err != nil {
			return nil, err
		}
		return &bridgemodbus.SaveConfigRes{
			Id: int64(exist.Id),
		}, nil
	}

	// 配置不存在，创建新配置
	insertBuilder := l.svcCtx.ModbusSlaveConfigModel.InsertBuilder()
	insertBuilder = insertBuilder.
		Columns("modbus_code", "slave_address", "slave").
		Values(in.ModbusCode, in.SlaveAddress, in.Slave)
	result, err := l.svcCtx.ModbusSlaveConfigModel.InsertWithBuilder(l.ctx, nil, insertBuilder)
	if err != nil {
		return nil, err
	}
	lastId, _ := result.LastInsertId()
	return &bridgemodbus.SaveConfigRes{
		Id: lastId,
	}, nil
}
