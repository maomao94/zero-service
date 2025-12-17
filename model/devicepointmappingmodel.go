package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ DevicePointMappingModel = (*customDevicePointMappingModel)(nil)

type (
	// DevicePointMappingModel is an interface to be customized, add more methods here,
	// and implement the added methods in customDevicePointMappingModel.
	DevicePointMappingModel interface {
		devicePointMappingModel
		withSession(session sqlx.Session) DevicePointMappingModel
	}

	customDevicePointMappingModel struct {
		*defaultDevicePointMappingModel
	}
)

// NewDevicePointMappingModel returns a model for the database table.
func NewDevicePointMappingModel(conn sqlx.SqlConn) DevicePointMappingModel {
	return &customDevicePointMappingModel{
		defaultDevicePointMappingModel: newDevicePointMappingModel(conn),
	}
}

func (m *customDevicePointMappingModel) withSession(session sqlx.Session) DevicePointMappingModel {
	return NewDevicePointMappingModel(sqlx.NewSqlConnFromSession(session))
}
