package model

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ PlanBatchModel = (*customPlanBatchModel)(nil)

type (
	// PlanBatchModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPlanBatchModel.
	PlanBatchModel interface {
		planBatchModel
		withSession(session sqlx.Session) PlanBatchModel
		UpdateBatchCompletedTime(ctx context.Context, id int64) error
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

func (m *customPlanBatchModel) UpdateBatchCompletedTime(ctx context.Context, id int64) error {
	now := time.Now()
	subQuery := "SELECT 1 FROM plan_exec_item i WHERE i.del_state = 0 AND i.batch_pk = b.id AND i.status NOT IN (?, ?)"
	builder := squirrel.
		Update(m.table+" AS b").
		Set("completed_time", now).
		Where("b.id = ?", id).
		Where("b.completed_time IS NULL").
		Where("NOT EXISTS ("+subQuery+")", StatusCompleted, StatusTerminated)
	if m.dbType == DatabaseTypePostgres {
		builder = builder.PlaceholderFormat(squirrel.Dollar)
	}
	sql, args, err := builder.ToSql()
	if err != nil {
		return err
	}
	_, err = m.conn.ExecCtx(ctx, sql, args...)
	return err
}
