package model

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"zero-service/common/tool"

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
	// 1. 执行项状态为待执行(0)或失败(3)或延期(4)
	// 2. 下次触发时间 <= 当前时间
	// 3. 执行项未终止
	// 4. 执行项未暂停
	// 5. 关联的计划状态为启用(1)
	// 6. 关联的计划未终止
	// 7. 关联的计划未暂停
	// 8. 关联的计划未删除
	currentTime := time.Now()
	// Calculate timeout threshold (30 minutes ago) for stuck running items
	timeoutThreshold := currentTime.Add(-30 * time.Minute)

	// Join with plan table and add plan status conditions
	where := fmt.Sprintf(
		`pei.is_terminated = 0 AND pei.is_paused = 0 AND 
		 p.plan_id = pei.plan_id AND p.del_state = 0 AND p.status = 1 AND p.is_terminated = 0 AND p.is_paused = 0 AND
		 (pei.next_trigger_time <= '%s' AND pei.status IN (0, 1, 3, 4))`,
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
		`SELECT pei.id, pei.create_time, pei.update_time, pei.delete_time, pei.del_state, pei.version, 
		 pei.plan_id, pei.item_id, pei.item_name, pei.service_addr, pei.payload, 
		 pei.request_timeout, pei.plan_trigger_time, pei.next_trigger_time, 
		 pei.last_trigger_time, pei.trigger_count, pei.status, 
		 pei.last_result, pei.last_msg, pei.is_terminated, 
		 pei.terminated_time, pei.terminated_reason, pei.is_paused, 
		 pei.paused_time, pei.paused_reason 
		 FROM %s pei 
		 JOIN plan p ON p.plan_id = pei.plan_id 
		 WHERE %s ORDER BY %s LIMIT 1`,
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
			`UPDATE %s SET status = 1, next_trigger_time = '%s', last_trigger_time = '%s' WHERE id = %d and %s`,
			m.table,
			carbon.CreateFromStdTime(nextTriggerTime).ToDateTimeString(),
			carbon.CreateFromStdTime(currentTime).ToDateTimeString(),
			execItem.Id,
			where,
		)

		result, updateErr := m.conn.ExecCtx(ctx, updateSQL)
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
	sql := fmt.Sprintf(
		`UPDATE %s SET status = 1, last_result = 'running', last_trigger_time = '%s', trigger_count = trigger_count + 1 WHERE id = %d`,
		m.table,
		carbon.Now().ToDateTimeString(),
		id,
	)
	_, err := m.conn.ExecCtx(ctx, sql)
	return err
}

// UpdateStatusToCompleted 更新执行项状态为已完成
func (m *customPlanExecItemModel) UpdateStatusToCompleted(ctx context.Context, id int64, lastResult, lastMsg string) error {
	sql := fmt.Sprintf(
		`UPDATE %s SET status = 2, last_result = '%s', last_msg = '%s', last_trigger_time = '%s' WHERE id = %d`,
		m.table,
		lastResult,
		lastMsg,
		carbon.Now().ToDateTimeString(),
		id,
	)
	_, err := m.conn.ExecCtx(ctx, sql)
	return err
}

// UpdateStatusToFailed 更新执行项状态为失败
func (m *customPlanExecItemModel) UpdateStatusToFailed(ctx context.Context, id int64, lastResult, lastMsg string) error {
	nowTime := carbon.Now().ToDateTimeString()
	type ItemInfo struct {
		TriggerCount int64 `db:"trigger_count"`
	}
	var itemInfo ItemInfo
	getInfoSql := fmt.Sprintf(
		`SELECT trigger_count FROM %s WHERE id = %d`,
		m.table,
		id,
	)
	if err := m.conn.QueryRowCtx(ctx, &itemInfo, getInfoSql); err != nil {
		return err
	}

	triggerCount := itemInfo.TriggerCount
	defaultTimeout := retryInterval

	nextTriggerTime, isExceeded := tool.CalculateNextTriggerTime(triggerCount, defaultTimeout)
	carbonNextTriggerTime := carbon.CreateFromStdTime(nextTriggerTime).ToDateTimeString()

	var sql string
	if isExceeded {
		sql = fmt.Sprintf(
			`UPDATE %s SET status = 5, last_result = '%s', last_msg = '%s', 
			 next_trigger_time = '%s', last_trigger_time = '%s', 
			 trigger_count = trigger_count + 1, is_terminated = 1, 
			 terminated_time = '%s', terminated_reason = '超过重试上限，自动终止' WHERE id = %d`,
			m.table,
			lastResult,
			lastMsg,
			carbonNextTriggerTime,
			nowTime,
			nowTime,
			id,
		)
	} else {
		sql := fmt.Sprintf(
			`UPDATE %s SET status = 3, last_result = '%s', last_msg = '%s', 
			 next_trigger_time = '%s', last_trigger_time = '%s', 
			 trigger_count = trigger_count + 1 WHERE id = %d`,
			m.table,
			lastResult,
			lastMsg,
			carbonNextTriggerTime,
			nowTime,
			id,
		)
		_, err := m.conn.ExecCtx(ctx, sql)
		return err
	}

	_, err := m.conn.ExecCtx(ctx, sql)
	return err
}

// UpdateStatusToDelayed 更新执行项状态为延期
func (m *customPlanExecItemModel) UpdateStatusToDelayed(ctx context.Context, id int64, lastResult, lastMsg string, nextTriggerTime string) error {
	sql := fmt.Sprintf(
		`UPDATE %s SET status = 4, last_result = '%s', last_msg = '%s', next_trigger_time = '%s', last_trigger_time = '%s' WHERE id = %d`,
		m.table,
		lastResult,
		lastMsg,
		nextTriggerTime,
		carbon.Now().ToDateTimeString(),
		id,
	)
	_, err := m.conn.ExecCtx(ctx, sql)
	return err
}
