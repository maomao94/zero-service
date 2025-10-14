package logic

import (
	"context"
	"time"
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
	exist, err := l.svcCtx.ModbusSlaveConfigModel.FindOneByModbusCode(l.ctx, in.ModbusCode)
	if err != nil && err != model.ErrNotFound {
		return nil, err
	}
	if err == nil && exist != nil {
		exist.SlaveAddress = in.SlaveAddress
		exist.Slave = int64(in.Slave)
		_, err = l.svcCtx.ModbusSlaveConfigModel.Update(l.ctx, nil, exist)
		if err != nil {
			return nil, err
		}
		return &bridgemodbus.SaveConfigRes{
			Id: int64(exist.Id),
		}, nil
	} else {
		var lastId int64
		insertBuilder := l.svcCtx.ModbusSlaveConfigModel.InsertBuilder()
		insertBuilder = insertBuilder.
			Columns("delete_time", "modbus_code", "slave_address", "slave").
			Values(time.Unix(0, 0), in.ModbusCode, in.SlaveAddress, in.Slave)
		query, args, err := insertBuilder.ToSql()
		if err != nil {
			return nil, err
		}
		result, err := l.svcCtx.ModbusSlaveConfigModel.ExecCtx(l.ctx, nil, query, args...)
		if err != nil {
			return nil, err
		}
		lastId, _ = result.LastInsertId()
		return &bridgemodbus.SaveConfigRes{
			Id: lastId,
		}, nil
	}
}
