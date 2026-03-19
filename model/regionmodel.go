package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ RegionModel = (*customRegionModel)(nil)

type (
	// RegionModel is an interface to be customized, add more methods here,
	// and implement the added methods in customRegionModel.
	RegionModel interface {
		regionModel
		withSession(session sqlx.Session) RegionModel
	}

	customRegionModel struct {
		*defaultRegionModel
	}
)

// NewRegionModel returns a model for the database table.
func NewRegionModel(conn sqlx.SqlConn, opts ...ModelOption) RegionModel {
	return &customRegionModel{
		defaultRegionModel: newRegionModel(conn, opts...),
	}
}

func (m *customRegionModel) withSession(session sqlx.Session) RegionModel {
	return &customRegionModel{
		defaultRegionModel: newRegionModel(sqlx.NewSqlConnFromSession(session), WithDBType(m.dbType)),
	}
}
