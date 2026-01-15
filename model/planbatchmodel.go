package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ PlanBatchModel = (*customPlanBatchModel)(nil)

type (
	// PlanBatchModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPlanBatchModel.
	PlanBatchModel interface {
		planBatchModel
		withSession(session sqlx.Session) PlanBatchModel
	}

	customPlanBatchModel struct {
		*defaultPlanBatchModel
	}
)

// NewPlanBatchModel returns a model for the database table.
func NewPlanBatchModel(conn sqlx.SqlConn) PlanBatchModel {
	return NewPlanBatchModelWithDBType(conn, DatabaseTypeMySQL)
}

// NewPlanBatchModelWithDBType returns a model for the database table with db type.
func NewPlanBatchModelWithDBType(conn sqlx.SqlConn, dbType DatabaseType) PlanBatchModel {
	return &customPlanBatchModel{
		defaultPlanBatchModel: newPlanBatchModelWithDBType(conn, dbType),
	}
}

func (m *customPlanBatchModel) withSession(session sqlx.Session) PlanBatchModel {
	return NewPlanBatchModelWithDBType(sqlx.NewSqlConnFromSession(session), m.dbType)
}
