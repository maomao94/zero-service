package gormmodel

import (
	"context"
	"database/sql"
	"time"
	"unicode/utf8"

	"zero-service/common/gormx"
	"zero-service/common/tool"
	"zero-service/model"

	"github.com/zeromicro/go-zero/core/mathx"
	"gorm.io/gorm"
)

// ============================================================================
// GORM struct definitions
// ============================================================================

type Plan struct {
	gormx.LegacyStringBaseModel
	gormx.VersionMixin

	CreateUser       sql.NullString `gorm:"column:create_user;size:64;comment:创建人"`
	UpdateUser       sql.NullString `gorm:"column:update_user;size:64;comment:更新人"`
	DeptCode         sql.NullString `gorm:"column:dept_code;size:64;comment:机构code"`
	PlanId           string         `gorm:"column:plan_id;size:64;comment:计划唯一标识;uniqueIndex"`
	PlanName         sql.NullString `gorm:"column:plan_name;size:128;comment:计划任务名称"`
	Type             sql.NullString `gorm:"column:type;size:64;comment:任务类型;index"`
	GroupId          sql.NullString `gorm:"column:group_id;size:64;comment:计划组ID,用于分组管理计划任务;index"`
	RecurrenceRule   string         `gorm:"column:recurrence_rule;type:text;comment:重复规则，JSON格式存储"`
	StartTime        time.Time      `gorm:"column:start_time;comment:规则生效开始时间;index"`
	EndTime          time.Time      `gorm:"column:end_time;comment:规则生效结束时间;index"`
	Status           int64          `gorm:"column:status;comment:状态：0-禁用，1-启用，2-暂停，3-终止;index"`
	ScanFlg          int64          `gorm:"column:scan_flg;comment:扫表标记, 0-未扫表, 1-已扫表"`
	TerminatedReason sql.NullString `gorm:"column:terminated_reason;size:2000;comment:终止原因"`
	PausedTime       sql.NullTime   `gorm:"column:paused_time;comment:暂停时间;index"`
	PausedReason     sql.NullString `gorm:"column:paused_reason;size:256;comment:暂停原因"`
	FinishedTime     sql.NullTime   `gorm:"column:finished_time;comment:结束时间"`
	Description      sql.NullString `gorm:"column:description;size:256;comment:备注信息"`
	Ext1             sql.NullString `gorm:"column:ext_1;size:256;comment:扩展字段1"`
	Ext2             sql.NullString `gorm:"column:ext_2;size:256;comment:扩展字段2"`
	Ext3             sql.NullString `gorm:"column:ext_3;size:256;comment:扩展字段3"`
	Ext4             sql.NullString `gorm:"column:ext_4;size:256;comment:扩展字段4"`
	Ext5             sql.NullString `gorm:"column:ext_5;size:256;comment:扩展字段5"`
}

func (Plan) TableName() string { return "plan" }

type PlanBatch struct {
	gormx.LegacyStringBaseModel
	gormx.VersionMixin

	CreateUser       sql.NullString `gorm:"column:create_user;size:64;comment:创建人"`
	UpdateUser       sql.NullString `gorm:"column:update_user;size:64;comment:更新人"`
	DeptCode         sql.NullString `gorm:"column:dept_code;size:64;comment:机构code"`
	PlanPk           string         `gorm:"column:plan_pk;size:64;comment:关联的计划主键ID;index"`
	PlanId           string         `gorm:"column:plan_id;size:64;comment:关联的计划ID;index"`
	BatchId          string         `gorm:"column:batch_id;size:64;comment:批ID;uniqueIndex"`
	BatchName        sql.NullString `gorm:"column:batch_name;size:128;comment:批次名称"`
	BatchNum         sql.NullString `gorm:"column:batch_num;size:128;comment:批次序号;uniqueIndex"`
	Status           int64          `gorm:"column:status;comment:状态：0-禁用，1-启用，2-暂停，3-终止;index"`
	ScanFlg          int64          `gorm:"column:scan_flg;comment:扫表标记, 0-未扫表, 1-已扫表"`
	PlanTriggerTime  sql.NullTime   `gorm:"column:plan_trigger_time;comment:计划触发时间"`
	TerminatedReason sql.NullString `gorm:"column:terminated_reason;size:2000;comment:终止原因"`
	PausedTime       sql.NullTime   `gorm:"column:paused_time;comment:暂停时间"`
	PausedReason     sql.NullString `gorm:"column:paused_reason;size:256;comment:暂停原因"`
	FinishedTime     sql.NullTime   `gorm:"column:finished_time;comment:结束时间"`
	Ext1             sql.NullString `gorm:"column:ext_1;size:256;comment:扩展字段1"`
	Ext2             sql.NullString `gorm:"column:ext_2;size:256;comment:扩展字段2"`
	Ext3             sql.NullString `gorm:"column:ext_3;size:256;comment:扩展字段3"`
	Ext4             sql.NullString `gorm:"column:ext_4;size:256;comment:扩展字段4"`
	Ext5             sql.NullString `gorm:"column:ext_5;size:256;comment:扩展字段5"`
}

func (PlanBatch) TableName() string { return "plan_batch" }

type PlanExecItem struct {
	gormx.LegacyStringBaseModel
	gormx.VersionMixin

	CreateUser       sql.NullString `gorm:"column:create_user;size:64;comment:创建人"`
	UpdateUser       sql.NullString `gorm:"column:update_user;size:64;comment:更新人"`
	DeptCode         sql.NullString `gorm:"column:dept_code;size:64;comment:机构code"`
	PlanPk           string         `gorm:"column:plan_pk;size:64;comment:关联的计划主键ID;index:idx_plan_pk_item_id,priority:1"`
	PlanId           string         `gorm:"column:plan_id;size:64;comment:关联的计划ID;index:idx_plan_id_item_id,priority:1"`
	BatchPk          string         `gorm:"column:batch_pk;size:64;comment:批主键ID;index"`
	BatchId          string         `gorm:"column:batch_id;size:64;comment:批ID;index"`
	ExecId           string         `gorm:"column:exec_id;size:64;comment:执行ID;uniqueIndex"`
	ItemId           string         `gorm:"column:item_id;size:64;comment:执行项ID;index:idx_plan_pk_item_id,priority:2;index:idx_plan_id_item_id,priority:2"`
	ItemType         sql.NullString `gorm:"column:item_type;size:64;comment:执行项类型"`
	ItemName         sql.NullString `gorm:"column:item_name;size:128;comment:执行项名称"`
	ItemRowId        int64          `gorm:"column:item_row_id;comment:执行项行ID"`
	PointId          sql.NullString `gorm:"column:point_id;size:64;comment:点位id;index"`
	Payload          string         `gorm:"column:payload;type:text;comment:业务负载"`
	RequestTimeout   int64          `gorm:"column:request_timeout;comment:请求超时时间（毫秒）"`
	PlanTriggerTime  time.Time      `gorm:"column:plan_trigger_time;comment:计划触发时间"`
	NextTriggerTime  time.Time      `gorm:"column:next_trigger_time;comment:下次触发时间（扫表核心字段）;index:idx_core_scan,priority:2"`
	LastTriggerTime  sql.NullTime   `gorm:"column:last_trigger_time;comment:上次触发时间"`
	TriggerCount     int64          `gorm:"column:trigger_count;comment:触发次数"`
	Status           int64          `gorm:"column:status;comment:状态：0-等待调度，10-延期等待，100-执行中，150-暂停，200-完成，300-终止;index;index:idx_core_scan,priority:3"`
	LastResult       sql.NullString `gorm:"column:last_result;size:256;comment:上次执行结果"`
	LastMessage      sql.NullString `gorm:"column:last_message;size:2000;comment:上次结果描述"`
	LastReason       sql.NullString `gorm:"column:last_reason;size:2000;comment:上次结果原因"`
	TerminatedReason sql.NullString `gorm:"column:terminated_reason;size:2000;comment:终止原因"`
	PausedTime       sql.NullTime   `gorm:"column:paused_time;comment:暂停时间"`
	PausedReason     sql.NullString `gorm:"column:paused_reason;size:256;comment:暂停原因"`
	Ext1             sql.NullString `gorm:"column:ext_1;size:256;comment:扩展字段1"`
	Ext2             sql.NullString `gorm:"column:ext_2;size:256;comment:扩展字段2"`
	Ext3             sql.NullString `gorm:"column:ext_3;size:256;comment:扩展字段3"`
	Ext4             sql.NullString `gorm:"column:ext_4;size:256;comment:扩展字段4"`
	Ext5             sql.NullString `gorm:"column:ext_5;size:256;comment:扩展字段5"`
}

func (PlanExecItem) TableName() string { return "plan_exec_item" }

type PlanExecLog struct {
	gormx.LegacyStringBaseModel
	gormx.VersionMixin

	CreateUser  sql.NullString `gorm:"column:create_user;size:64;comment:创建人"`
	UpdateUser  sql.NullString `gorm:"column:update_user;size:64;comment:更新人"`
	DeptCode    sql.NullString `gorm:"column:dept_code;size:64;comment:机构code"`
	PlanPk      string         `gorm:"column:plan_pk;size:64;comment:关联的计划主键ID;index"`
	PlanId      string         `gorm:"column:plan_id;size:64;comment:计划任务ID;index"`
	PlanName    sql.NullString `gorm:"column:plan_name;size:128;comment:计划任务名称"`
	BatchPk     string         `gorm:"column:batch_pk;size:64;comment:批主键ID;index"`
	BatchId     string         `gorm:"column:batch_id;size:64;comment:批ID;index"`
	ItemPk      string         `gorm:"column:item_pk;size:64;comment:关联的执行项主键ID;index"`
	ExecId      string         `gorm:"column:exec_id;size:64;comment:执行ID;index"`
	ItemId      string         `gorm:"column:item_id;size:64;comment:执行项ID;index"`
	ItemType    sql.NullString `gorm:"column:item_type;size:64;comment:执行项类型"`
	ItemName    sql.NullString `gorm:"column:item_name;size:128;comment:执行项名称"`
	PointId     sql.NullString `gorm:"column:point_id;size:64;comment:点位id"`
	TriggerTime time.Time      `gorm:"column:trigger_time;comment:触发时间;index"`
	TraceId     sql.NullString `gorm:"column:trace_id;size:64;comment:唯一追踪ID;index"`
	ExecResult  sql.NullString `gorm:"column:exec_result;size:256;comment:执行结果;index"`
	Message     sql.NullString `gorm:"column:message;size:2000;comment:结果描述"`
	Reason      sql.NullString `gorm:"column:reason;size:2000;comment:结果原因"`
}

func (PlanExecLog) TableName() string { return "plan_exec_log" }

// ============================================================================
// Plan / PlanBatch finished time helpers
// ============================================================================

// UpdatePlanFinishedTime marks a plan as finished if all its batches are done.
func UpdatePlanFinishedTime(ctx context.Context, db *gorm.DB, id string) (int64, error) {
	now := time.Now()
	result := db.WithContext(ctx).Model(&Plan{}).
		Where("id = ?", id).
		Where("finished_time IS NULL").
		Where("NOT EXISTS (SELECT 1 FROM plan_batch b WHERE b.is_deleted = 0 AND b.plan_pk = plan.id AND b.finished_time IS NULL)").
		Update("finished_time", now)
	return result.RowsAffected, result.Error
}

// UpdatePlanBatchFinishedTime marks a batch as finished if all its exec items are done.
func UpdatePlanBatchFinishedTime(ctx context.Context, db *gorm.DB, id string) (int64, error) {
	now := time.Now()
	result := db.WithContext(ctx).Model(&PlanBatch{}).
		Where("id = ?", id).
		Where("finished_time IS NULL").
		Where("NOT EXISTS (SELECT 1 FROM plan_exec_item i WHERE i.is_deleted = 0 AND i.batch_pk = plan_batch.id AND i.status NOT IN (?, ?))", model.StatusCompleted, model.StatusTerminated).
		Update("finished_time", now)
	return result.RowsAffected, result.Error
}

// CalculatePlanProgress computes the completion percentage for a plan.
func CalculatePlanProgress(ctx context.Context, db *gorm.DB, planPk string) (float32, error) {
	type stats struct {
		Total    int64
		Finished int64
	}
	var s stats
	if err := db.WithContext(ctx).Model(&PlanExecItem{}).
		Select("COUNT(*) AS total, SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) AS finished", model.StatusCompleted).
		Where("plan_pk = ?", planPk).
		Scan(&s).Error; err != nil {
		return 0, err
	}
	if s.Total == 0 {
		return 0, nil
	}
	return float32(s.Finished) / float32(s.Total) * 100, nil
}

// ============================================================================
// PlanExecItem status updates
// ============================================================================

// GetBatchStatusCounts returns status counts for a plan batch.
func GetBatchStatusCounts(ctx context.Context, db *gorm.DB, batchPk string) ([]model.ExecItemStatusCountEx, error) {
	var rows []model.ExecItemStatusCountEx
	err := db.WithContext(ctx).Model(&PlanExecItem{}).
		Select("status, COUNT(*) AS count").
		Where("batch_pk = ?", batchPk).
		Group("status").
		Scan(&rows).Error
	return rows, err
}

// GetBatchTotalExecItems returns the total count of exec items in a batch.
func GetBatchTotalExecItems(ctx context.Context, db *gorm.DB, batchPk string) (int64, error) {
	var count int64
	err := db.WithContext(ctx).Model(&PlanExecItem{}).
		Where("batch_pk = ?", batchPk).
		Count(&count).Error
	return count, err
}

// LockTriggerItem atomically locks one pending exec item for dispatch.
func LockTriggerItem(ctx context.Context, db *gorm.DB, dbType gormx.DatabaseType, expireIn time.Duration) (*PlanExecItem, error) {
	currentTime := time.Now()
	nextTriggerTime := currentTime.Add(expireIn)
	var item PlanExecItem
	query := db.WithContext(gormx.WithoutSQLTrace(ctx)).Table("plan_exec_item AS pei").
		Select("pei.version, pei.id, pei.plan_pk, pei.plan_id, pei.batch_pk, pei.batch_id, pei.exec_id, pei.item_id, pei.item_name, pei.point_id, pei.next_trigger_time, pei.payload, pei.plan_trigger_time, pei.request_timeout").
		Joins("JOIN plan p ON p.id = pei.plan_pk").
		Joins("JOIN plan_batch pb ON pb.id = pei.batch_pk").
		Where("pei.is_deleted = ?", 0).
		Where("pei.status IN (?, ?, ?)", model.StatusWaiting, model.StatusDelayed, model.StatusRunning).
		Where("pei.next_trigger_time <= ?", currentTime).
		Where("p.is_deleted = ?", 0).
		Where("p.status = ?", model.PlanStatusEnabled).
		Where("pb.is_deleted = ?", 0).
		Where("pb.status = ?", model.PlanStatusEnabled).
		Order(clauseOrderBy(dbType)).
		Limit(1)
	if err := query.Scan(&item).Error; err != nil {
		return nil, err
	}
	if item.Id == "" {
		return nil, model.ErrNotFound
	}
	result := db.WithContext(ctx).Model(&PlanExecItem{}).
		Where("id = ?", item.Id).
		Where("next_trigger_time <= ?", currentTime).
		Where("status IN (?, ?, ?)", model.StatusWaiting, model.StatusDelayed, model.StatusRunning).
		Where("version = ?", item.Version.Int64).
		Updates(map[string]any{"status": model.StatusRunning, "next_trigger_time": nextTriggerTime, "last_trigger_time": currentTime, "version": item.Version.Int64 + 1})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, model.ErrNotFound
	}
	return &item, nil
}

func clauseOrderBy(dbType gormx.DatabaseType) string {
	switch dbType {
	case gormx.DatabasePostgres, gormx.DatabaseSQLite:
		return "RANDOM()"
	default:
		return "RAND()"
	}
}

// UpdateExecItemStatusToRunning 将执行项状态更新为执行中。
// 与旧 UpdateStatusToRunning 保持一致：lastResult 为空时不更新 last_result。
func UpdateExecItemStatusToRunning(ctx context.Context, db *gorm.DB, id string, lastResult string) error {
	updates := map[string]any{
		"status":            model.StatusRunning,
		"last_trigger_time": time.Now(),
	}
	if lastResult != "" {
		updates["last_result"] = lastResult
	}
	return db.WithContext(ctx).Model(&PlanExecItem{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateExecItemStatusToFail handles exec item failure with retry backoff.
// It reads the current trigger_count, calculates the next retry time, and updates the
// status (delayed or terminated if exceeded) with appropriate status guards.
func UpdateExecItemStatusToFail(ctx context.Context, db *gorm.DB, id string, lastResult, lastMessage, lastReason string, statusIn []int, statusOut []int) error {
	lastMessage = truncateExecItemResultText(lastMessage)
	lastReason = truncateExecItemResultText(lastReason)

	var item PlanExecItem
	if err := db.WithContext(ctx).Model(&PlanExecItem{}).Select("trigger_count").Where("id = ?", id).Scan(&item).Error; err != nil {
		return err
	}
	nextTriggerTime, isExceeded := tool.CalculateNextTriggerTime(item.TriggerCount+1, mathx.NewUnstable(expiryDeviation).AroundDuration(retryInterval))
	updates := map[string]any{
		"status":            model.StatusDelayed,
		"last_result":       lastResult,
		"last_message":      lastMessage,
		"last_reason":       lastReason,
		"next_trigger_time": nextTriggerTime,
		"last_trigger_time": time.Now(),
		"trigger_count":     gorm.Expr("trigger_count + 1"),
	}
	if isExceeded {
		updates["status"] = model.StatusTerminated
		updates["last_result"] = model.ResultOngoing
		updates["terminated_reason"] = "超过重试上限，调度平台自动终止"
	}
	q := db.WithContext(ctx).Model(&PlanExecItem{}).Where("id = ?", id)
	if len(statusIn) > 0 {
		q = q.Where("status IN ?", statusIn)
	}
	if len(statusOut) > 0 {
		q = q.Where("status NOT IN ?", statusOut)
	}
	return q.Updates(updates).Error
}

// UpdateExecItemStatusToCompleted 将执行项状态更新为已完成。
func UpdateExecItemStatusToCompleted(ctx context.Context, db *gorm.DB, id, lastMessage, lastReason string, statusIn, statusOut []int) error {
	lastMessage = truncateExecItemResultText(lastMessage)
	lastReason = truncateExecItemResultText(lastReason)

	q := db.WithContext(ctx).Model(&PlanExecItem{}).Where("id = ?", id)
	if len(statusIn) > 0 {
		q = q.Where("status IN ?", statusIn)
	}
	if len(statusOut) > 0 {
		q = q.Where("status NOT IN ?", statusOut)
	}
	return q.Updates(map[string]any{
		"status":            model.StatusCompleted,
		"last_result":       model.ResultCompleted,
		"last_message":      lastMessage,
		"last_reason":       lastReason,
		"last_trigger_time": time.Now(),
		"trigger_count":     gorm.Expr("trigger_count + 1"),
	}).Error
}

// UpdateExecItemStatusToDelayed 将执行项状态更新为延期等待。
func UpdateExecItemStatusToDelayed(ctx context.Context, db *gorm.DB, id, lastResult, lastMessage, lastReason string, nextTriggerTime time.Time, statusIn, statusOut []int) error {
	lastMessage = truncateExecItemResultText(lastMessage)
	lastReason = truncateExecItemResultText(lastReason)

	q := db.WithContext(ctx).Model(&PlanExecItem{}).Where("id = ?", id)
	if len(statusIn) > 0 {
		q = q.Where("status IN ?", statusIn)
	}
	if len(statusOut) > 0 {
		q = q.Where("status NOT IN ?", statusOut)
	}
	return q.Updates(map[string]any{
		"status":            model.StatusDelayed,
		"last_result":       lastResult,
		"last_message":      lastMessage,
		"last_reason":       lastReason,
		"next_trigger_time": nextTriggerTime,
		"last_trigger_time": time.Now(),
		"trigger_count":     gorm.Expr("trigger_count + 1"),
	}).Error
}

// UpdateExecItemStatusToOngoing 将执行项状态更新为执行中（下游返回 ongoing 回执）。
// nextTriggerTime 为 nil 时不更新该字段。
// updateTriggerInfo 为 false 时不更新 last_trigger_time 和 trigger_count，用于 RPC 回调路径
// （cron 锁定阶段已更新过），避免重复计数。
func UpdateExecItemStatusToOngoing(ctx context.Context, db *gorm.DB, id, lastMessage, lastReason string, statusIn, statusOut []int, nextTriggerTime *time.Time, updateTriggerInfo bool) error {
	lastMessage = truncateExecItemResultText(lastMessage)
	lastReason = truncateExecItemResultText(lastReason)

	q := db.WithContext(ctx).Model(&PlanExecItem{}).Where("id = ?", id)
	if len(statusIn) > 0 {
		q = q.Where("status IN ?", statusIn)
	}
	if len(statusOut) > 0 {
		q = q.Where("status NOT IN ?", statusOut)
	}
	updates := map[string]any{
		"status":       model.StatusRunning,
		"last_result":  model.ResultOngoing,
		"last_message": lastMessage,
		"last_reason":  lastReason,
	}
	if nextTriggerTime != nil {
		updates["next_trigger_time"] = *nextTriggerTime
	}
	if updateTriggerInfo {
		updates["last_trigger_time"] = time.Now()
		updates["trigger_count"] = gorm.Expr("trigger_count + 1")
	}
	return q.Updates(updates).Error
}

// UpdateExecItemStatusToTerminated 将执行项状态更新为已终止。
func UpdateExecItemStatusToTerminated(ctx context.Context, db *gorm.DB, id, lastMessage, lastReason string, statusIn, statusOut []int) error {
	lastMessage = truncateExecItemResultText(lastMessage)
	lastReason = truncateExecItemResultText(lastReason)

	q := db.WithContext(ctx).Model(&PlanExecItem{}).Where("id = ?", id)
	if len(statusIn) > 0 {
		q = q.Where("status IN ?", statusIn)
	}
	if len(statusOut) > 0 {
		q = q.Where("status NOT IN ?", statusOut)
	}
	return q.Updates(map[string]any{
		"status":            model.StatusTerminated,
		"last_result":       model.ResultTerminated,
		"last_message":      lastMessage,
		"last_reason":       lastReason,
		"last_trigger_time": time.Now(),
		"trigger_count":     gorm.Expr("trigger_count + 1"),
		"terminated_reason": lastReason,
	}).Error
}

const (
	retryInterval   = time.Second * 10
	expiryDeviation = 0.05
	maxResultRunes  = 1000
)

func truncateExecItemResultText(s string) string {
	if utf8.RuneCountInString(s) <= maxResultRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxResultRunes])
}
