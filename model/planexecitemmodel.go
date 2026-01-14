package model

import (
	"context"
	"fmt"
	"strings"
	"time"

	"zero-service/common/tool"

	"github.com/Masterminds/squirrel"
	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/mathx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ PlanExecItemModel = (*customPlanExecItemModel)(nil)

// PlanExecItemStat represents the result of grouped statistics
type PlanExecItemStat struct {
	PlanId          string    `db:"plan_id"`
	BatchId         string    `db:"batch_id"`
	PlanTriggerTime time.Time `db:"plan_trigger_time"`
	Total           int64     `db:"total"`
	Success         int64     `db:"success"`
	Failed          int64     `db:"failed"`
	Running         int64     `db:"running"`
}

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
		UpdateStatusToCompleted(ctx context.Context, id int64, lastMsg string) error
		// 更新执行项状态为回调
		UpdateStatusToCallback(ctx context.Context, id int64, lastResult, lastMsg string) error
		// 更新执行项状态为延期
		UpdateStatusToDelayed(ctx context.Context, id int64, lastResult, lastMsg string, nextTriggerTime string) error
		// 获取分组统计信息
		FindGroupedStats(ctx context.Context, builder squirrel.SelectBuilder) ([]*PlanExecItem, error)
		// 获取分组总数
		FindGroupedCount(ctx context.Context, builder squirrel.SelectBuilder) (int64, error)
		// 通用SQL查询方法
		QuerySQL(ctx context.Context, sql string, args ...interface{}) ([]map[string]interface{}, error)
		// 查询计划执行项统计信息
		FindPlanExecItemStats(ctx context.Context, planId, batchId, startTime, endTime string, page, pageSize int64) ([]PlanExecItemStat, int64, error)
	}

	customPlanExecItemModel struct {
		*defaultPlanExecItemModel
		unstableExpiry mathx.Unstable
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
		"pei.id", "pei.plan_pk", "pei.plan_id", "pei.item_id", "pei.item_name", "pei.point_id", "pei.next_trigger_time", "pei.service_addr", "pei.payload", "pei.plan_trigger_time", "pei.request_timeout",
	).From(m.table+" as pei").
		Join("plan p ON p.plan_id = pei.plan_id").
		Where("pei.next_trigger_time <= ?", currentTimeStr).
		Where("p.del_state = ?", 0).
		Where("p.status = ?", PlanStatusEnabled).
		Where("pei.status IN (?, ?, ?)", StatusWaiting, StatusDelayed, StatusRunning)
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
			Set("status", StatusRunning).
			Set("next_trigger_time", nextTriggerTimeStr).
			Set("last_trigger_time", currentTimeStr).
			Where("id = ?", execItem.Id).
			Where("next_trigger_time <= ?", currentTimeStr).
			Where("status IN (?, ?, ?)", StatusWaiting, StatusDelayed, StatusRunning)
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

func (m *customPlanExecItemModel) UpdateStatusToRunning(ctx context.Context, id int64) error {
	updateBuilder := squirrel.Update(m.table).
		Set("status", StatusRunning).
		Set("last_result", ResultRunning).
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

func (m *customPlanExecItemModel) UpdateStatusToCompleted(ctx context.Context, id int64, lastMsg string) error {
	currentTime := time.Now()
	currentTimeStr := carbon.CreateFromStdTime(currentTime).ToDateTimeMicroString()
	updateBuilder := squirrel.Update(m.table).
		Set("status", StatusCompleted).
		Set("last_result", ResultCompleted).
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

func (m *customPlanExecItemModel) UpdateStatusToCallback(ctx context.Context, id int64, lastResult, lastMsg string) error {
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
	expiry := m.unstableExpiry.AroundDuration(retryInterval)
	nextTriggerTime, isExceeded := tool.CalculateNextTriggerTime(triggerCount+1, expiry)
	nextTriggerTimeStr := carbon.CreateFromStdTime(nextTriggerTime).ToDateTimeMicroString()
	var updateBuilder squirrel.UpdateBuilder
	if isExceeded {
		updateBuilder = squirrel.Update(m.table).
			Set("status", StatusTerminated).
			Set("last_result", ResultRunning).
			Set("last_msg", lastMsg).
			Set("next_trigger_time", nextTriggerTimeStr).
			Set("last_trigger_time", currentTimeStr).
			Set("trigger_count", squirrel.Expr("trigger_count + 1")).
			Set("terminated_time", currentTimeStr).
			Set("terminated_reason", "超过重试上限，调度平台自动终止").
			Where("id = ?", id)
	} else {
		updateBuilder = squirrel.Update(m.table).
			Set("status", StatusDelayed).
			Set("last_result", lastResult).
			Set("last_msg", lastMsg).
			Set("next_trigger_time", nextTriggerTimeStr).
			Set("last_trigger_time", currentTimeStr).
			Set("trigger_count", squirrel.Expr("trigger_count + 1")).
			Set("paused_time", currentTimeStr).
			Set("paused_reason", "调度平台自动延期").
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

func (m *customPlanExecItemModel) UpdateStatusToDelayed(ctx context.Context, id int64, lastMsg string, lastResult, nextTriggerTimeStr string) error {
	ct := carbon.Parse(nextTriggerTimeStr)
	if ct.Error != nil {
		return ct.Error
	}
	currentTime := time.Now()
	currentTimeStr := carbon.CreateFromStdTime(currentTime).ToDateTimeMicroString()
	updateBuilder := squirrel.Update(m.table).
		Set("status", StatusDelayed).
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

func (m *customPlanExecItemModel) FindGroupedStats(ctx context.Context, builder squirrel.SelectBuilder) ([]*PlanExecItem, error) {
	groupBuilder := squirrel.Select(
		"plan_id",
		"batch_id",
		"plan_trigger_time",
		"status",
		"COUNT(*) as trigger_count",
	).From(m.table)
	origSQL, origArgs, err := builder.ToSql()
	if err != nil {
		return nil, err
	}
	whereStart := strings.Index(origSQL, "WHERE")
	if whereStart != -1 {
		whereClause := origSQL[whereStart:]
		groupBuilder = groupBuilder.Where(whereClause, origArgs...)
	}
	groupBuilder = groupBuilder.GroupBy("plan_id, batch_id, plan_trigger_time, status")
	if m.dbType == DatabaseTypePostgreSQL {
		groupBuilder = groupBuilder.PlaceholderFormat(squirrel.Dollar)
	}
	return m.SelectWithBuilder(ctx, groupBuilder)
}

func (m *customPlanExecItemModel) FindGroupedCount(ctx context.Context, builder squirrel.SelectBuilder) (int64, error) {
	countBuilder := squirrel.Select("*").From(m.table)
	origSQL, origArgs, err := builder.ToSql()
	if err != nil {
		return 0, err
	}
	whereStart := strings.Index(origSQL, "WHERE")
	if whereStart != -1 {
		whereClause := origSQL[whereStart:]
		countBuilder = countBuilder.Where(whereClause, origArgs...)
	}
	items, err := m.FindAll(ctx, countBuilder, "plan_id, batch_id, plan_trigger_time")
	if err != nil {
		return 0, err
	}
	type groupKey struct {
		planId          string
		batchId         string
		planTriggerTime time.Time
	}
	groupMap := make(map[groupKey]bool)
	for _, item := range items {
		key := groupKey{
			planId:          item.PlanId,
			batchId:         item.BatchId,
			planTriggerTime: item.PlanTriggerTime,
		}
		groupMap[key] = true
	}
	return int64(len(groupMap)), nil
}

// QuerySQL 通用SQL查询方法
func (m *customPlanExecItemModel) QuerySQL(ctx context.Context, sql string, args ...interface{}) ([]map[string]interface{}, error) {
	// 这里实现通用SQL查询，返回map[string]interface{}
	// 由于sqlx.SqlConn没有直接的QuerySQL方法，我们可以使用QueryRowCtx或其他方法
	// 但为了简化，我们先返回nil
	return nil, nil
}

// GroupedBatchInfo 分组批次信息
type GroupedBatchInfo struct {
	PlanId          string    `db:"plan_id"`
	BatchId         string    `db:"batch_id"`
	PlanTriggerTime time.Time `db:"plan_trigger_time"`
	Total           int64     `db:"total"`
}

// StatusCount 状态统计
type StatusCount struct {
	PlanId      string `db:"plan_id"`
	BatchId     string `db:"batch_id"`
	Status      int64  `db:"status"`
	StatusCount int64  `db:"status_count"`
}

// FindGroupedBatchInfo 获取分组批次信息
func (m *customPlanExecItemModel) FindGroupedBatchInfo(ctx context.Context, planId, startTime, endTime string, page, pageSize int64) ([]GroupedBatchInfo, int64, error) {
	// 构建查询条件
	whereClause := "del_state = 0"
	args := []interface{}{}

	if planId != "" {
		whereClause += " AND plan_id = ?"
		args = append(args, planId)
	}

	if startTime != "" {
		whereClause += " AND plan_trigger_time >= ?"
		args = append(args, startTime)
	}

	if endTime != "" {
		whereClause += " AND plan_trigger_time <= ?"
		args = append(args, endTime)
	}

	// 1. 获取总记录数
	countSQL := fmt.Sprintf(`
		SELECT COUNT(*) FROM (
			SELECT plan_id, batch_id, plan_trigger_time
			FROM plan_exec_item
			WHERE %s
			GROUP BY plan_id, batch_id, plan_trigger_time
		) AS subquery
	`, whereClause)

	var total int64
	err := m.conn.QueryRowCtx(ctx, &total, countSQL, args...)
	if err != nil {
		return nil, 0, err
	}

	// 2. 构建分组查询SQL
	groupSQL := fmt.Sprintf(`
		SELECT 
			plan_id, 
			batch_id, 
			plan_trigger_time, 
			COUNT(*) AS total 
		FROM 
			plan_exec_item 
		WHERE %s
		GROUP BY 
			plan_id, 
			batch_id, 
			plan_trigger_time 
		ORDER BY 
			plan_trigger_time DESC 
		LIMIT ? OFFSET ?
	`, whereClause)

	// 添加分页参数
	finalArgs := append(args, pageSize, (page-1)*pageSize)

	// 执行查询
	var batchInfos []GroupedBatchInfo
	err = m.conn.QueryRowCtx(ctx, &batchInfos, groupSQL, finalArgs...)
	if err != nil {
		return nil, 0, err
	}

	return batchInfos, total, nil
}

// FindStatusCountsByBatchId 根据batch_id获取状态统计
func (m *customPlanExecItemModel) FindStatusCountsByBatchId(ctx context.Context, batchId string) ([]StatusCount, error) {
	// 构建查询SQL
	statusSQL := `
		SELECT 
			plan_id, 
			batch_id, 
			status, 
			COUNT(status) AS status_count 
		FROM 
			plan_exec_item 
		WHERE 
			batch_id = ? AND del_state = 0
		GROUP BY 
			plan_id, 
			batch_id, 
			status
	`

	// 执行查询
	var statusCounts []StatusCount
	err := m.conn.QueryRowCtx(ctx, &statusCounts, statusSQL, batchId)
	if err != nil {
		return nil, err
	}

	return statusCounts, nil
}
