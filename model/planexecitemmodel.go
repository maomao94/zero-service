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
	PlanExecItemModel interface {
		planExecItemModel
		withSession(session sqlx.Session) PlanExecItemModel
		// 锁定需要触发的执行项
		LockTriggerItem(ctx context.Context, expireIn time.Duration) (*PlanExecItem, error)
		// 更新执行项状态为执行中
		UpdateStatusToRunning(ctx context.Context, id int64) error
		// 更新执行项状态为已完成
		UpdateStatusToCompleted(ctx context.Context, id int64, lastMessage, lastReason string, statusIn []int, statusOut []int) error
		// 更新执行项状态为失败延期
		UpdateStatusToFail(ctx context.Context, id int64, lastResult, lastMessage, lastReason string, statusIn []int, statusOut []int) error
		// 更新执行项状态为延期
		UpdateStatusToDelayed(ctx context.Context, id int64, lastResult, lastMessage, lastReason, nextTriggerTime string, statusIn []int, statusOut []int) error
		// 更新执行项状态为已终止
		UpdateStatusToTerminated(ctx context.Context, id int64, lastMessage, lastReason string, statusIn []int, statusOut []int) error
		// 更新执行项为进行中状态并补充回调数据
		UpdateStatusToOngoing(ctx context.Context, id int64, lastMessage, lastReason string, updateTriggerInfo bool, nextTriggerTime string, statusIn []int, statusOut []int) error
		// 通用SQL查询方法
		QuerySQL(ctx context.Context, sql string, args ...interface{}) ([]map[string]interface{}, error)
		// 获取批次执行项状态统计
		GetBatchStatusCounts(ctx context.Context, batchPk int64) ([]ExecItemStatusCountEx, error)
		// 获取批次总执行项数
		GetBatchTotalExecItems(ctx context.Context, batchPk int64) (int64, error)
	}

	customPlanExecItemModel struct {
		*defaultPlanExecItemModel
		unstableExpiry mathx.Unstable
	}

	ExecItemStatusCountEx struct {
		Status int32 `db:"status"`
		Count  int64 `db:"count"`
	}
)

const (
	retryInterval   = time.Second * 10
	expiryDeviation = 0.05
)

func NewPlanExecItemModel(conn sqlx.SqlConn) PlanExecItemModel {
	return &customPlanExecItemModel{
		defaultPlanExecItemModel: newPlanExecItemModel(conn, DatabaseTypeMySQL),
		unstableExpiry:           mathx.NewUnstable(expiryDeviation),
	}
}

func NewPlanExecItemModelWithDBType(conn sqlx.SqlConn, dbType DatabaseType) PlanExecItemModel {
	return &customPlanExecItemModel{
		defaultPlanExecItemModel: newPlanExecItemModel(conn, dbType),
		unstableExpiry:           mathx.NewUnstable(expiryDeviation),
	}
}

func (m *customPlanExecItemModel) withSession(session sqlx.Session) PlanExecItemModel {
	return NewPlanExecItemModelWithDBType(sqlx.NewSqlConnFromSession(session), m.dbType)
}

func (m *customPlanExecItemModel) LockTriggerItem(ctx context.Context, expireIn time.Duration) (*PlanExecItem, error) {
	// 准备SQL查询，获取需要触发的执行项
	// 条件：
	// 1. 执行项状态为待执行、延期等待或执行中
	// 2. 下次触发时间 <= 当前时间
	// 3. 关联的计划状态为启用
	// 4. 关联的计划未删除
	currentTime := time.Now()
	currentTimeStr := carbon.CreateFromStdTime(currentTime).ToDateTimeMicroString()
	nextTriggerTime := currentTime.Add(expireIn)
	nextTriggerTimeStr := carbon.CreateFromStdTime(nextTriggerTime).ToDateTimeMicroString()
	selectBuilder := squirrel.Select(
		"pei.version", "pei.id", "pei.plan_pk", "pei.plan_id", "pei.item_id", "pei.item_name", "pei.point_id", "pei.next_trigger_time", "pei.payload", "pei.plan_trigger_time", "pei.request_timeout",
	).From(m.table+" AS pei").
		Join("plan p ON p.id = pei.plan_pk").
		Join("plan_batch pb ON pb.id = pei.batch_pk").
		Where("pei.del_state = ?", 0).
		Where("pei.status IN (?, ?,?)", StatusWaiting, StatusDelayed, StatusRunning).
		Where("pei.next_trigger_time <= ?", currentTimeStr).
		Where("p.del_state = ?", 0).
		Where("p.status = ?", PlanStatusEnabled).
		Where("pb.del_state = ?", 0).
		Where("pb.status = ?", PlanStatusEnabled)
	if m.dbType == DatabaseTypePostgres {
		selectBuilder = selectBuilder.OrderBy("RANDOM()")
	} else {
		selectBuilder = selectBuilder.OrderBy("RAND()")
	}
	selectBuilder = selectBuilder.Limit(1)
	if m.dbType == DatabaseTypePostgres {
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
			Set("status", StatusRunning).
			Set("next_trigger_time", nextTriggerTimeStr).
			Set("last_trigger_time", currentTimeStr).
			Set("version", execItem.Version+1).
			Where("id = ?", execItem.Id).
			Where("next_trigger_time <= ?", currentTimeStr).
			Where("status IN (?, ?,?)", StatusWaiting, StatusDelayed, StatusRunning).
			Where("version = ?", execItem.Version)
		if m.dbType == DatabaseTypePostgres {
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

func (m *customPlanExecItemModel) UpdateStatusToRunning(ctx context.Context, id int64) error {
	updateBuilder := squirrel.Update(m.table).
		Set("status", StatusRunning).
		Set("last_result", ResultOngoing).
		Set("last_trigger_time", time.Now()).
		Where("id = ?", id)
	if m.dbType == DatabaseTypePostgres {
		updateBuilder = updateBuilder.PlaceholderFormat(squirrel.Dollar)
	}
	updateSQL, updateArgs, err := updateBuilder.ToSql()
	if err != nil {
		return err
	}
	_, err = m.conn.ExecCtx(ctx, updateSQL, updateArgs...)
	return err
}

func (m *customPlanExecItemModel) UpdateStatusToCompleted(ctx context.Context, id int64, lastMessage, lastReason string, statusIn []int, statusOut []int) error {
	currentTime := time.Now()
	currentTimeStr := carbon.CreateFromStdTime(currentTime).ToDateTimeMicroString()
	updateBuilder := squirrel.Update(m.table).
		Set("status", StatusCompleted).
		Set("last_result", ResultCompleted).
		Set("last_message", lastMessage).
		Set("last_reason", lastReason).
		Set("last_trigger_time", currentTimeStr).
		Set("trigger_count", squirrel.Expr("trigger_count + 1")).
		Where("id = ?", id)
	if len(statusIn) > 0 {
		updateBuilder = updateBuilder.Where(squirrel.Eq{"status": statusIn})
	}
	if len(statusOut) > 0 {
		updateBuilder = updateBuilder.Where(squirrel.NotEq{"status": statusOut})
	}
	if m.dbType == DatabaseTypePostgres {
		updateBuilder = updateBuilder.PlaceholderFormat(squirrel.Dollar)
	}
	updateSQL, updateArgs, err := updateBuilder.ToSql()
	if err != nil {
		return err
	}
	_, err = m.conn.ExecCtx(ctx, updateSQL, updateArgs...)
	return err
}

func (m *customPlanExecItemModel) UpdateStatusToFail(ctx context.Context, id int64, lastResult, lastMessage, lastReason string, statusIn []int, statusOut []int) error {
	currentTime := time.Now()
	currentTimeStr := carbon.CreateFromStdTime(currentTime).ToDateTimeMicroString()
	selectBuilder := squirrel.Select("trigger_count").From(m.table).Where("id = ?", id)
	if m.dbType == DatabaseTypePostgres {
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
	expiry := m.unstableExpiry.AroundDuration(retryInterval)
	nextTriggerTime, isExceeded := tool.CalculateNextTriggerTime(triggerCount+1, expiry)
	nextTriggerTimeStr := carbon.CreateFromStdTime(nextTriggerTime).ToDateTimeMicroString()
	var updateBuilder squirrel.UpdateBuilder
	if isExceeded {
		updateBuilder = squirrel.Update(m.table).
			Set("status", StatusTerminated).
			Set("last_result", ResultOngoing).
			Set("last_message", lastMessage).
			Set("last_reason", lastReason).
			Set("next_trigger_time", nextTriggerTimeStr).
			Set("last_trigger_time", currentTimeStr).
			Set("trigger_count", squirrel.Expr("trigger_count + 1")).
			Set("terminated_reason", "超过重试上限，调度平台自动终止").
			Where("id = ?", id)
	} else {
		updateBuilder = squirrel.Update(m.table).
			Set("status", StatusDelayed).
			Set("last_result", lastResult).
			Set("last_message", lastMessage).
			Set("last_reason", lastReason).
			Set("next_trigger_time", nextTriggerTimeStr).
			Set("last_trigger_time", currentTimeStr).
			Set("trigger_count", squirrel.Expr("trigger_count + 1")).
			Set("paused_time", currentTimeStr).
			Set("paused_reason", "调度平台自动延期").
			Where("id = ?", id)
	}
	if len(statusIn) > 0 {
		updateBuilder = updateBuilder.Where(squirrel.Eq{"status": statusIn})
	}
	if len(statusOut) > 0 {
		updateBuilder = updateBuilder.Where(squirrel.NotEq{"status": statusOut})
	}
	if m.dbType == DatabaseTypePostgres {
		updateBuilder = updateBuilder.PlaceholderFormat(squirrel.Dollar)
	}
	updateSQL, updateArgs, err := updateBuilder.ToSql()
	if err != nil {
		return err
	}
	_, err = m.conn.ExecCtx(ctx, updateSQL, updateArgs...)
	return err
}

func (m *customPlanExecItemModel) UpdateStatusToDelayed(ctx context.Context, id int64, lastResult, lastMessage, lastReason, nextTriggerTimeStr string, statusIn []int, statusOut []int) error {
	ct := carbon.Parse(nextTriggerTimeStr)
	if ct.Error != nil {
		return ct.Error
	}
	currentTime := time.Now()
	currentTimeStr := carbon.CreateFromStdTime(currentTime).ToDateTimeMicroString()
	updateBuilder := squirrel.Update(m.table).
		Set("status", StatusDelayed).
		Set("last_result", lastResult).
		Set("last_message", lastMessage).
		Set("last_reason", lastReason).
		Set("next_trigger_time", nextTriggerTimeStr).
		Set("last_trigger_time", currentTimeStr).
		Set("trigger_count", squirrel.Expr("trigger_count + 1")).
		Where("id = ?", id)
	if len(statusIn) > 0 {
		updateBuilder = updateBuilder.Where(squirrel.Eq{"status": statusIn})
	}
	if len(statusOut) > 0 {
		updateBuilder = updateBuilder.Where(squirrel.NotEq{"status": statusOut})
	}
	if m.dbType == DatabaseTypePostgres {
		updateBuilder = updateBuilder.PlaceholderFormat(squirrel.Dollar)
	}
	updateSQL, updateArgs, err := updateBuilder.ToSql()
	if err != nil {
		return err
	}
	_, err = m.conn.ExecCtx(ctx, updateSQL, updateArgs...)
	return err
}

func (m *customPlanExecItemModel) UpdateStatusToTerminated(ctx context.Context, id int64, lastMessage, lastReason string, statusIn []int, statusOut []int) error {
	currentTime := time.Now()
	currentTimeStr := carbon.CreateFromStdTime(currentTime).ToDateTimeMicroString()
	updateBuilder := squirrel.Update(m.table).
		Set("status", StatusTerminated).
		Set("last_result", ResultTerminated).
		Set("last_message", lastMessage).
		Set("last_reason", lastReason).
		Set("last_trigger_time", currentTimeStr).
		Set("trigger_count", squirrel.Expr("trigger_count + 1")).
		Where("id = ?", id)
	if len(statusIn) > 0 {
		updateBuilder = updateBuilder.Where(squirrel.Eq{"status": statusIn})
	}
	if len(statusOut) > 0 {
		updateBuilder = updateBuilder.Where(squirrel.NotEq{"status": statusOut})
	}
	if m.dbType == DatabaseTypePostgres {
		updateBuilder = updateBuilder.PlaceholderFormat(squirrel.Dollar)
	}
	updateSQL, updateArgs, err := updateBuilder.ToSql()
	if err != nil {
		return err
	}
	_, err = m.conn.ExecCtx(ctx, updateSQL, updateArgs...)
	return err
}

func (m *customPlanExecItemModel) UpdateStatusToOngoing(ctx context.Context, id int64, lastMessage, lastReason string, updateTriggerInfo bool, nextTriggerTime string, statusIn []int, statusOut []int) error {
	currentTime := time.Now()
	currentTimeStr := carbon.CreateFromStdTime(currentTime).ToDateTimeMicroString()
	updateBuilder := squirrel.Update(m.table).
		Set("status", StatusRunning).
		Set("last_result", ResultOngoing).
		Set("last_message", lastMessage).
		Set("last_reason", lastReason)

	// 根据参数决定是否更新触发相关信息
	if updateTriggerInfo {
		updateBuilder = updateBuilder.
			Set("last_trigger_time", currentTimeStr).
			Set("trigger_count", squirrel.Expr("trigger_count + 1"))
	}

	if nextTriggerTime != "" {
		updateBuilder = updateBuilder.Set("next_trigger_time", nextTriggerTime)
	}

	updateBuilder = updateBuilder.Where("id = ?", id)

	if len(statusIn) > 0 {
		updateBuilder = updateBuilder.Where(squirrel.Eq{"status": statusIn})
	}
	if len(statusOut) > 0 {
		updateBuilder = updateBuilder.Where(squirrel.NotEq{"status": statusOut})
	}
	if m.dbType == DatabaseTypePostgres {
		updateBuilder = updateBuilder.PlaceholderFormat(squirrel.Dollar)
	}
	updateSQL, updateArgs, err := updateBuilder.ToSql()
	if err != nil {
		return err
	}
	_, err = m.conn.ExecCtx(ctx, updateSQL, updateArgs...)
	return err
}

func (m *customPlanExecItemModel) GetBatchStatusCounts(ctx context.Context, batchPk int64) ([]ExecItemStatusCountEx, error) {
	selectBuilder := m.SelectBuilder().Columns("status", "COUNT(*) as count").
		Where("batch_pk = ?", batchPk).
		Where("del_state = ?", 0).
		GroupBy("status")
	selectSQL, selectArgs, err := selectBuilder.ToSql()
	if err != nil {
		return nil, err
	}
	var statusCounts []ExecItemStatusCountEx
	if err := m.conn.QueryRowsCtx(ctx, &statusCounts, selectSQL, selectArgs...); err != nil {
		return nil, err
	}
	return statusCounts, nil
}

func (m *customPlanExecItemModel) GetBatchTotalExecItems(ctx context.Context, batchPk int64) (int64, error) {
	selectBuilder := m.SelectBuilder().Columns("COUNT(*) as count").
		From(m.table).
		Where("batch_pk = ?", batchPk).
		Where("del_state = ?", 0)
	selectSQL, selectArgs, err := selectBuilder.ToSql()
	if err != nil {
		return 0, err
	}
	var count int64
	if err := m.conn.QueryRowCtx(ctx, &count, selectSQL, selectArgs...); err != nil {
		return 0, err
	}
	return count, nil
}

func (m *customPlanExecItemModel) QuerySQL(ctx context.Context, sql string, args ...interface{}) ([]map[string]interface{}, error) {
	return nil, nil
}
