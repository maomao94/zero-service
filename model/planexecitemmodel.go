package model

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ PlanExecItemModel = (*customPlanExecItemModel)(nil)

type (
	// PlanExecItemModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPlanExecItemModel.
	PlanExecItemModel interface {
		planExecItemModel
		withSession(session sqlx.Session) PlanExecItemModel
		// 锁定需要触发的执行项
		LockTriggerItem(ctx context.Context, expireIn time.Duration) (*PlanExecItem, error)
		// 更新执行项状态为执行中
		UpdateStatusToRunning(ctx context.Context, id int64) error
		// 更新执行项状态为已完成
		UpdateStatusToCompleted(ctx context.Context, id int64, result string) error
		// 更新执行项状态为失败
		UpdateStatusToFailed(ctx context.Context, id int64, errMsg string) error
		// 更新执行项状态为延期
		UpdateStatusToDelayed(ctx context.Context, id int64, delayReason string, nextTriggerTime string) error
	}

	customPlanExecItemModel struct {
		*defaultPlanExecItemModel
	}
)

// NewPlanExecItemModel returns a model for the database table.
func NewPlanExecItemModel(conn sqlx.SqlConn) PlanExecItemModel {
	return &customPlanExecItemModel{
		defaultPlanExecItemModel: newPlanExecItemModel(conn),
	}
}

func (m *customPlanExecItemModel) withSession(session sqlx.Session) PlanExecItemModel {
	return NewPlanExecItemModel(sqlx.NewSqlConnFromSession(session))
}

// LockTriggerItem 锁定需要触发的执行项
func (m *customPlanExecItemModel) LockTriggerItem(ctx context.Context, expireIn time.Duration) (*PlanExecItem, error) {
	// 准备SQL查询，获取需要触发的执行项
	// 条件：
	// 1. 状态为待执行(0)或延期(4)
	// 2. 下次触发时间 <= 当前时间
	// 3. 未终止
	// 4. 未暂停
	currentTime := time.Now()
	// Calculate timeout threshold (30 minutes ago) for stuck running items
	timeoutThreshold := currentTime.Add(-30 * time.Minute)
	where := fmt.Sprintf(
		`is_terminated = 0 AND is_paused = 0 AND (
			(next_trigger_time <= '%s' AND status IN (0, 4)) OR
			(status = 1 AND last_trigger_time IS NOT NULL AND last_trigger_time <= '%s')
		)`,
		carbon.CreateFromStdTime(currentTime).ToDateTimeString(),
		carbon.CreateFromStdTime(timeoutThreshold).ToDateTimeString(),
	)

	// 随机获取一个需要触发的执行项，兼容不同数据库
	var randomFunc string
	connType := reflect.TypeOf(m.conn).String()
	if strings.Contains(connType, "postgres") {
		randomFunc = "RANDOM()"
	} else {
		// Default to MySQL RAND()
		randomFunc = "RAND()"
	}

	ssql := fmt.Sprintf(
		`SELECT %s FROM %s WHERE %s ORDER BY %s LIMIT 1`,
		planExecItemRows,
		m.table,
		where,
		randomFunc,
	)

	var execItem PlanExecItem
	err := m.conn.QueryRowCtx(ctx, &execItem, ssql)
	switch err {
	case sqlx.ErrNotFound:
		return nil, ErrNotFound
	case nil:
		// 更新下次触发时间，防止重复触发
		nextTriggerTime := currentTime.Add(expireIn)
		updateSQL := fmt.Sprintf(
			`UPDATE %s SET status = 1, next_trigger_time = '%s', last_trigger_time = '%s' WHERE id = %d`,
			m.table,
			carbon.CreateFromStdTime(nextTriggerTime).ToDateTimeString(),
			carbon.CreateFromStdTime(currentTime).ToDateTimeString(),
			execItem.Id,
		)

		_, err = m.conn.ExecCtx(ctx, updateSQL)
		if err != nil {
			return nil, err
		}

		return &execItem, nil
	default:
		return nil, err
	}
}

// UpdateStatusToRunning 更新执行项状态为执行中
func (m *customPlanExecItemModel) UpdateStatusToRunning(ctx context.Context, id int64) error {
	sql := fmt.Sprintf(
		`UPDATE %s SET status = 1, last_trigger_time = '%s', trigger_count = trigger_count + 1 WHERE id = %d`,
		m.table,
		carbon.Now().ToDateTimeString(),
		id,
	)
	_, err := m.conn.ExecCtx(ctx, sql)
	return err
}

// UpdateStatusToCompleted 更新执行项状态为已完成
func (m *customPlanExecItemModel) UpdateStatusToCompleted(ctx context.Context, id int64, result string) error {
	sql := fmt.Sprintf(
		`UPDATE %s SET status = 2, last_result = '%s', last_trigger_time = '%s' WHERE id = %d`,
		m.table,
		result,
		carbon.Now().ToDateTimeString(),
		id,
	)
	_, err := m.conn.ExecCtx(ctx, sql)
	return err
}

// UpdateStatusToFailed 更新执行项状态为失败
func (m *customPlanExecItemModel) UpdateStatusToFailed(ctx context.Context, id int64, errMsg string) error {
	sql := fmt.Sprintf(
		`UPDATE %s SET status = 3, last_error = '%s', last_trigger_time = '%s' WHERE id = %d`,
		m.table,
		errMsg,
		carbon.Now().ToDateTimeString(),
		id,
	)
	_, err := m.conn.ExecCtx(ctx, sql)
	return err
}

// UpdateStatusToDelayed 更新执行项状态为延期
func (m *customPlanExecItemModel) UpdateStatusToDelayed(ctx context.Context, id int64, delayReason string, nextTriggerTime string) error {
	sql := fmt.Sprintf(
		`UPDATE %s SET status = 4, last_result = '%s', next_trigger_time = '%s', last_trigger_time = '%s' WHERE id = %d`,
		m.table,
		delayReason,
		nextTriggerTime,
		carbon.Now().ToDateTimeString(),
		id,
	)
	_, err := m.conn.ExecCtx(ctx, sql)
	return err
}
