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
func NewModbusSlaveConfigModel(conn sqlx.SqlConn) ModbusSlaveConfigModel {
	return NewModbusSlaveConfigModelWithDBType(conn, DatabaseTypeMySQL)
}

// NewModbusSlaveConfigModelWithDBType returns a model for the database table with db type.
func NewModbusSlaveConfigModelWithDBType(conn sqlx.SqlConn, dbType DatabaseType) ModbusSlaveConfigModel {
	return &customModbusSlaveConfigModel{
		defaultModbusSlaveConfigModel: newModbusSlaveConfigModelWithDBType(conn, dbType),
	}
}

func (m *customModbusSlaveConfigModel) withSession(session sqlx.Session) ModbusSlaveConfigModel {
	return NewModbusSlaveConfigModelWithDBType(sqlx.NewSqlConnFromSession(session), m.defaultModbusSlaveConfigModel.dbType)
}
