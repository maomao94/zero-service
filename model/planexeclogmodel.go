package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ PlanExecLogModel = (*customPlanExecLogModel)(nil)

type (
	// PlanExecLogModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPlanExecLogModel.
	PlanExecLogModel interface {
		planExecLogModel
		withSession(session sqlx.Session) PlanExecLogModel
	}

	customPlanExecLogModel struct {
		*defaultPlanExecLogModel
	}
)

func NewPlanExecLogModel(conn sqlx.SqlConn, opts ...ModelOption) PlanExecLogModel {
	return &customPlanExecLogModel{
		defaultPlanExecLogModel: newPlanExecLogModel(conn, opts...),
	}
}

func (m *customPlanExecLogModel) withSession(session sqlx.Session) PlanExecLogModel {
	return &customPlanExecLogModel{
		defaultPlanExecLogModel: newPlanExecLogModel(sqlx.NewSqlConnFromSession(session), WithDBType(m.dbType)),
	}
}
