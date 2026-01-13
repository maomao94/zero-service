package model

import (
	"context"
	"time"

	"zero-service/common/tool"

	"github.com/Masterminds/squirrel"
	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/mathx"
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
		UpdateStatusToCompleted(ctx context.Context, id int64, lastResult, lastMsg string) error
		// 更新执行项状态为失败
		UpdateStatusToFailed(ctx context.Context, id int64, lastResult, lastMsg string) error
		// 更新执行项状态为延期
		UpdateStatusToDelayed(ctx context.Context, id int64, lastResult, lastMsg string, nextTriggerTime string) error
	}

	customPlanExecItemModel struct {
		*defaultPlanExecItemModel
		unstableExpiry mathx.Unstable
	}
)

const (
	retryInterval = time.Second * 10
)

func NewPlanExecItemModel(conn sqlx.SqlConn) PlanExecItemModel {
	return &customPlanExecItemModel{
		defaultPlanExecItemModel: newPlanExecItemModel(conn, DatabaseTypeMySQL),
	}
}

func NewPlanExecItemModelWithDBType(conn sqlx.SqlConn, dbType DatabaseType) PlanExecItemModel {
	return &customPlanExecItemModel{
		defaultPlanExecItemModel: newPlanExecItemModel(conn, dbType),
	}
}

func (m *customPlanExecItemModel) withSession(session sqlx.Session) PlanExecItemModel {
	return NewPlanExecItemModelWithDBType(sqlx.NewSqlConnFromSession(session), m.dbType)
}

// LockTriggerItem 锁定需要触发的执行项
func (m *customPlanExecItemModel) LockTriggerItem(ctx context.Context, expireIn time.Duration) (*PlanExecItem, error) {
	// 准备SQL查询，获取需要触发的执行项
	// 条件：
	// 1. 执行项状态为待执行(0)或失败(3)或延期(4)
	// 2. 下次触发时间 <= 当前时间
	// 3. 执行项未终止
	// 4. 执行项未暂停
	// 5. 关联的计划状态为启用(1)
	// 6. 关联的计划未终止
	// 7. 关联的计划未暂停
	// 8. 关联的计划未删除
	currentTime := time.Now()
	currentTimeStr := carbon.CreateFromStdTime(currentTime).ToDateTimeMicroString()
	nextTriggerTime := currentTime.Add(expireIn)
	nextTriggerTimeStr := carbon.CreateFromStdTime(nextTriggerTime).ToDateTimeMicroString()
	selectBuilder := squirrel.Select(
		"pei.id", "pei.plan_pk", "pei.plan_id", "pei.item_id", "pei.item_name", "pei.point_id", "pei.next_trigger_time", "pei.service_addr", "pei.payload", "pei.plan_trigger_time", "pei.request_timeout",
	).From(m.table+" as pei").
		Join("plan p ON p.plan_id = pei.plan_id").
		Where("pei.is_terminated = ?", 0).
		Where("pei.is_paused = ?", 0).
		Where("p.del_state = ?", 0).
		Where("p.status = ?", 1).
		Where("p.is_terminated = ?", 0).
		Where("p.is_paused = ?", 0).
		Where("pei.next_trigger_time <= ?", currentTimeStr).
		Where("pei.status IN (?, ?, ?, ?)", 0, 1, 3, 4)
	if m.dbType == DatabaseTypePostgreSQL {
		selectBuilder = selectBuilder.OrderBy("RANDOM()")
	} else {
		selectBuilder = selectBuilder.OrderBy("RAND()")
	}
	selectBuilder = selectBuilder.Limit(1)
	if m.dbType == DatabaseTypePostgreSQL {
		selectBuilder = selectBuilder.PlaceholderFormat(squirrel.Dollar)
	}
	selectSQL, selectArgs, err := selectBuilder.ToSql()
	if err != nil {
		return nil, err
	}
	var execItem PlanExecItem
	err = m.conn.QueryRowPartialCtx(ctx, &execItem, selectSQL, selectArgs...)
	switch err {
	case sqlx.ErrNotFound:
		return nil, ErrNotFound
	case nil:
		updateBuilder := squirrel.Update(m.table).
			Set("status", 1).
			Set("next_trigger_time", nextTriggerTimeStr).
			Set("last_trigger_time", currentTimeStr).
			Where("id = ?", execItem.Id).
			Where("is_terminated = ?", 0).
			Where("is_paused = ?", 0).
			Where("next_trigger_time <= ?", currentTimeStr).
			Where("status IN (?, ?, ?, ?)", 0, 1, 3, 4)
		if m.dbType == DatabaseTypePostgreSQL {
			updateBuilder = updateBuilder.PlaceholderFormat(squirrel.Dollar)
		}
		updateSQL, updateArgs, updateErr := updateBuilder.ToSql()
		if updateErr != nil {
			return nil, updateErr
		}
		result, updateErr := m.conn.ExecCtx(ctx, updateSQL, updateArgs...)
		if updateErr != nil {
			return nil, updateErr
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return nil, ErrNotFound
		}
		return &execItem, nil
	default:
		return nil, err
	}
}

// UpdateStatusToRunning 更新执行项状态为执行中
func (m *customPlanExecItemModel) UpdateStatusToRunning(ctx context.Context, id int64) error {
	updateBuilder := squirrel.Update(m.table).
		Set("status", 1).
		Set("last_result", "running").
		Set("last_trigger_time", time.Now()).
		Where("id = ?", id)
	if m.dbType == DatabaseTypePostgreSQL {
		updateBuilder = updateBuilder.PlaceholderFormat(squirrel.Dollar)
	}
	updateSQL, updateArgs, err := updateBuilder.ToSql()
	if err != nil {
		return err
	}
	_, err = m.conn.ExecCtx(ctx, updateSQL, updateArgs...)
	return err
}

// UpdateStatusToCompleted 更新执行项状态为已完成
func (m *customPlanExecItemModel) UpdateStatusToCompleted(ctx context.Context, id int64, lastResult, lastMsg string) error {
	currentTime := time.Now()
	currentTimeStr := carbon.CreateFromStdTime(currentTime).ToDateTimeMicroString()
	updateBuilder := squirrel.Update(m.table).
		Set("status", 2).
		Set("last_result", lastResult).
		Set("last_msg", lastMsg).
		Set("last_trigger_time", currentTimeStr).
		Where("id = ?", id)
	if m.dbType == DatabaseTypePostgreSQL {
		updateBuilder = updateBuilder.PlaceholderFormat(squirrel.Dollar)
	}
	updateSQL, updateArgs, err := updateBuilder.ToSql()
	if err != nil {
		return err
	}
	_, err = m.conn.ExecCtx(ctx, updateSQL, updateArgs...)
	return err
}

// UpdateStatusToFailed 更新执行项状态为失败
func (m *customPlanExecItemModel) UpdateStatusToFailed(ctx context.Context, id int64, lastResult, lastMsg string) error {
	currentTime := time.Now()
	currentTimeStr := carbon.CreateFromStdTime(currentTime).ToDateTimeMicroString()
	selectBuilder := squirrel.Select("trigger_count").From(m.table).Where("id = ?", id)
	if m.dbType == DatabaseTypePostgreSQL {
		selectBuilder = selectBuilder.PlaceholderFormat(squirrel.Dollar)
	}
	selectSQL, selectArgs, err := selectBuilder.ToSql()
	if err != nil {
		return err
	}
	type ItemInfo struct {
		TriggerCount int64 `db:"trigger_count"`
	}
	var itemInfo ItemInfo
	if err := m.conn.QueryRowCtx(ctx, &itemInfo, selectSQL, selectArgs...); err != nil {
		return err
	}
	triggerCount := itemInfo.TriggerCount
	defaultTimeout := retryInterval
	nextTriggerTime, isExceeded := tool.CalculateNextTriggerTime(triggerCount+1, defaultTimeout)
	nextTriggerTimeStr := carbon.CreateFromStdTime(nextTriggerTime).ToDateTimeMicroString()
	var updateBuilder squirrel.UpdateBuilder
	if isExceeded {
		updateBuilder = squirrel.Update(m.table).
			Set("status", 5).
			Set("last_result", lastResult).
			Set("last_msg", lastMsg).
			Set("next_trigger_time", nextTriggerTimeStr).
			Set("last_trigger_time", currentTimeStr).
			Set("trigger_count", squirrel.Expr("trigger_count + 1")).
			Set("is_terminated", 1).
			Set("terminated_time", currentTimeStr).
			Set("terminated_reason", "超过重试上限，自动终止").
			Where("id = ?", id)
	} else {
		updateBuilder = squirrel.Update(m.table).
			Set("status", 3).
			Set("last_result", lastResult).
			Set("last_msg", lastMsg).
			Set("next_trigger_time", nextTriggerTimeStr).
			Set("last_trigger_time", currentTimeStr).
			Set("trigger_count", squirrel.Expr("trigger_count + 1")).
			Where("id = ?", id)
	}
	if m.dbType == DatabaseTypePostgreSQL {
		updateBuilder = updateBuilder.PlaceholderFormat(squirrel.Dollar)
	}
	updateSQL, updateArgs, err := updateBuilder.ToSql()
	if err != nil {
		return err
	}
	_, err = m.conn.ExecCtx(ctx, updateSQL, updateArgs...)
	return err
}

// UpdateStatusToDelayed 更新执行项状态为延期
func (m *customPlanExecItemModel) UpdateStatusToDelayed(ctx context.Context, id int64, lastResult, lastMsg string, nextTriggerTimeStr string) error {
	ct := carbon.Parse(nextTriggerTimeStr)
	if ct.Error != nil {
		return ct.Error
	}
	currentTime := time.Now()
	currentTimeStr := carbon.CreateFromStdTime(currentTime).ToDateTimeMicroString()
	updateBuilder := squirrel.Update(m.table).
		Set("status", 4).
		Set("last_result", lastResult).
		Set("last_msg", lastMsg).
		Set("next_trigger_time", nextTriggerTimeStr).
		Set("last_trigger_time", currentTimeStr).
		Where("id = ?", id)
	if m.dbType == DatabaseTypePostgreSQL {
		updateBuilder = updateBuilder.PlaceholderFormat(squirrel.Dollar)
	}
	updateSQL, updateArgs, err := updateBuilder.ToSql()
	if err != nil {
		return err
	}
	_, err = m.conn.ExecCtx(ctx, updateSQL, updateArgs...)
	return err
}
