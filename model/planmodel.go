package model

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ PlanModel = (*customPlanModel)(nil)

type (
	// PlanModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPlanModel.
	PlanModel interface {
		planModel
		withSession(session sqlx.Session) PlanModel
		UpdateBatchFinishedTime(ctx context.Context, id int64) (int64, error)
	}

	customPlanModel struct {
		*defaultPlanModel
	}
)

func NewPlanModel(conn sqlx.SqlConn) PlanModel {
	return &customPlanModel{
		defaultPlanModel: newPlanModel(conn),
	}
}

func NewPlanModelWithDBType(conn sqlx.SqlConn, dbType DatabaseType) PlanModel {
	return &customPlanModel{
		defaultPlanModel: newPlanModelWithDBType(conn, dbType),
	}
}

func (m *customPlanModel) withSession(session sqlx.Session) PlanModel {
	return NewPlanModel(sqlx.NewSqlConnFromSession(session))
}

func (m *customPlanModel) UpdateBatchFinishedTime(ctx context.Context, id int64) (int64, error) {
	now := time.Now()
	subQuery := "SELECT 1 FROM plan_batch b WHERE b.del_state = 0 AND b.plan_pk = p.id AND b.finished_time IS NULL"
	builder := squirrel.
		Update(m.table+" AS p").
		Set("p.finished_time", now).
		Where("p.id = ?", id).
		Where("p.finished_time IS NULL").
		Where("NOT EXISTS (" + subQuery + ")")
	if m.dbType == DatabaseTypePostgres {
		builder = builder.PlaceholderFormat(squirrel.Dollar)
	}
	sql, args, err := builder.ToSql()
	if err != nil {
		return 0, err
	}
	sqlResult, err := m.conn.ExecCtx(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	updateCount, err := sqlResult.RowsAffected()
	if err != nil {
		return 0, err
	}
	return updateCount, err
}
