package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ PlanModel = (*customPlanModel)(nil)

type (
	// PlanModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPlanModel.
	PlanModel interface {
		planModel
		withSession(session sqlx.Session) PlanModel
	}

	customPlanModel struct {
		*defaultPlanModel
	}
)

// NewPlanModel returns a model for the database table.
func NewPlanModel(conn sqlx.SqlConn) PlanModel {
	return &customPlanModel{
		defaultPlanModel: newPlanModel(conn),
	}
}

func (m *customPlanModel) withSession(session sqlx.Session) PlanModel {
	return NewPlanModel(sqlx.NewSqlConnFromSession(session))
}
