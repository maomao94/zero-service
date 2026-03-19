package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ ModbusSlaveConfigModel = (*customModbusSlaveConfigModel)(nil)

type (
	// ModbusSlaveConfigModel is an interface to be customized, add more methods here,
	// and implement the added methods in customModbusSlaveConfigModel.
	ModbusSlaveConfigModel interface {
		modbusSlaveConfigModel
		withSession(session sqlx.Session) ModbusSlaveConfigModel
	}

	customModbusSlaveConfigModel struct {
		*defaultModbusSlaveConfigModel
	}
)

// NewModbusSlaveConfigModel returns a model for the database table.
func NewModbusSlaveConfigModel(conn sqlx.SqlConn, opts ...ModelOption) ModbusSlaveConfigModel {
	return &customModbusSlaveConfigModel{
		defaultModbusSlaveConfigModel: newModbusSlaveConfigModel(conn, opts...),
	}
}

func (m *customModbusSlaveConfigModel) withSession(session sqlx.Session) ModbusSlaveConfigModel {
	return &customModbusSlaveConfigModel{
		defaultModbusSlaveConfigModel: newModbusSlaveConfigModel(sqlx.NewSqlConnFromSession(session), WithDBType(m.defaultModbusSlaveConfigModel.dbType)),
	}
}
