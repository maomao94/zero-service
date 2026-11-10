package gormmodel

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"zero-service/common/gormx"
	"zero-service/model"

	"github.com/dromara/carbon/v2"
	"gorm.io/gorm"
)

// PlanConfirm 记录计划任务的一次人工确认。
//
// 当前只实现批次级确认（ConfirmTypeBatch），target_pk 指向 plan_batch.id；
// 预留 ConfirmTypeExecItem，未来下沉到执行项时 target_pk 指向 plan_exec_item.id。
// 记录是否存在即表示该目标是否需要确认，不依赖 plan_batch.status。
type PlanConfirm struct {
	gormx.LegacyStringBaseModel
	gormx.VersionMixin

	CreateUser sql.NullString `gorm:"column:create_user;size:64;comment:创建人"`
	UpdateUser sql.NullString `gorm:"column:update_user;size:64;comment:更新人"`
	DeptCode   sql.NullString `gorm:"column:dept_code;size:64;comment:机构code"`

	PlanPk      string         `gorm:"column:plan_pk;size:64;comment:关联的计划主键ID;index:idx_plan_confirm_plan_pk"`
	BatchPk     string         `gorm:"column:batch_pk;size:64;comment:批次主键ID;index:idx_plan_confirm_batch_pk"`
	ExecPk      sql.NullString `gorm:"column:exec_pk;size:64;comment:执行项主键ID（执行项级确认使用，暂未实现）"`
	ConfirmType int            `gorm:"column:confirm_type;comment:确认类型：1-批次确认，2-执行项确认（暂未实现）;uniqueIndex:uk_plan_confirm_target,priority:1"`
	TargetPk    string         `gorm:"column:target_pk;size:64;comment:确认目标主键ID;uniqueIndex:uk_plan_confirm_target,priority:2"`

	LeadMinutes   int            `gorm:"column:lead_minutes;comment:提前通知分钟数，0-到点通知"`
	NotifyAt      time.Time      `gorm:"column:notify_at;comment:计划通知时间;index:idx_plan_confirm_notify,priority:3"`
	NotifiedTime  sql.NullTime   `gorm:"column:notified_time;comment:实际通知时间;index:idx_plan_confirm_notify,priority:2"`
	Deadline      time.Time      `gorm:"column:deadline;comment:确认截止时间;index:idx_plan_confirm_expire,priority:2"`
	ConfirmedTime sql.NullTime   `gorm:"column:confirmed_time;comment:处理时间（确认/拒绝）"`
	ConfirmedUser sql.NullString `gorm:"column:confirmed_user;size:64;comment:处理人（确认/拒绝）"`
	ExpiredTime   sql.NullTime   `gorm:"column:expired_time;comment:确认终止时间：expired（超时）和 closed（人工终止）写入"`
	// ConfirmStatus 确认状态：0-未确认，1-已确认（允许/拒绝/超时/关闭均归为已确认）。
	// 两个扫描都以 confirm_status 为等值前缀：先按状态排除已处理记录，再按时间范围取最老一条。
	ConfirmStatus int            `gorm:"column:confirm_status;comment:确认状态：0-未确认，1-已确认;index:idx_plan_confirm_notify,priority:1;index:idx_plan_confirm_expire,priority:1"`
	Reason        sql.NullString `gorm:"column:reason;size:2000;comment:处理说明（拒绝原因/终止原因）"`
}

func (PlanConfirm) TableName() string { return "plan_confirm" }

// DefaultConfirmDeadline 返回批次确认默认截止时间：触发当天 23:59:59（东八区）。
// carbon.EndOfDay() 返回 23:59:59.999999999，写入秒级精度的列时会被四舍五入成次日 00:00:00，
// 因此统一截断到秒，保证落库值与语义一致。
func DefaultConfirmDeadline(triggerTime time.Time) time.Time {
	return carbon.CreateFromStdTime(triggerTime, carbon.Shanghai).EndOfDay().StdTime().Truncate(time.Second)
}

// BatchConfirmDeadline 返回批次确认截止时间。
// timeoutMinutes > 0 时从触发时间起算，支持跨天；<= 0 时默认触发当天 23:59:59。
// 返回值统一截断到秒，避免数据库按小数秒进位导致截止时间漂移到下一分钟/次日。
func BatchConfirmDeadline(triggerTime time.Time, timeoutMinutes int) time.Time {
	if timeoutMinutes <= 0 {
		return DefaultConfirmDeadline(triggerTime)
	}
	return triggerTime.Add(time.Duration(timeoutMinutes) * time.Minute).Truncate(time.Second)
}

// FindPlanConfirmByTarget 按确认类型和目标主键查询确认记录，不存在时返回 nil。
func FindPlanConfirmByTarget(ctx context.Context, db *gorm.DB, confirmType int, targetPk string) (*PlanConfirm, error) {
	var confirm PlanConfirm
	err := db.WithContext(ctx).
		Where("confirm_type = ? AND target_pk = ?", confirmType, targetPk).
		First(&confirm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &confirm, nil
}

// FindPlanConfirmsByBatchPks 批量查询批次确认记录，key 为 batch_pk。
func FindPlanConfirmsByBatchPks(ctx context.Context, db *gorm.DB, batchPks []string) (map[string]PlanConfirm, error) {
	result := make(map[string]PlanConfirm, len(batchPks))
	if len(batchPks) == 0 {
		return result, nil
	}
	var rows []PlanConfirm
	if err := db.WithContext(ctx).
		Where("confirm_type = ? AND batch_pk IN ?", model.ConfirmTypeBatch, batchPks).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.BatchPk] = row
	}
	return result, nil
}

// ConfirmPlanTarget 确认一个目标，只有待确认状态才会更新成功。
func ConfirmPlanTarget(ctx context.Context, db *gorm.DB, confirmType int, targetPk, user string, now time.Time) (int64, error) {
	result := db.WithContext(ctx).Model(&PlanConfirm{}).
		Where("confirm_type = ? AND target_pk = ?", confirmType, targetPk).
		Where("confirm_status = ?", model.PlanConfirmStatusWaiting).
		Updates(map[string]any{
			"confirm_status": model.PlanConfirmStatusConfirmed,
			"confirmed_time": sql.NullTime{Time: now, Valid: true},
			"confirmed_user": sql.NullString{String: user, Valid: user != ""},
		})
	return result.RowsAffected, result.Error
}

// FindPendingPlanConfirmNotify 查询一条待通知的批次确认记录。
// 已过截止时间的记录由超时扫描处理，这里直接跳过，避免通知队列被旧记录阻塞。
// 并发控制由调用方通过 Redis 锁完成；返回后需在锁内重新校验状态。
// 与 LockTriggerItem 一致：轮询 SELECT 静默成功日志，避免空转刷屏；错误与慢 SQL 仍会记录。
func FindPendingPlanConfirmNotify(ctx context.Context, db *gorm.DB, now time.Time) (*PlanConfirm, error) {
	var candidate PlanConfirm
	err := db.WithContext(gormx.WithoutSQLTrace(ctx)).
		Table("plan_confirm AS pc").
		Select("pc.*").
		Joins("JOIN plan_batch pb ON pb.id = pc.batch_pk AND pb.is_deleted = 0").
		Joins("JOIN plan p ON p.id = pc.plan_pk AND p.is_deleted = 0").
		Where("pc.is_deleted = ?", 0).
		Where("pc.confirm_type = ?", model.ConfirmTypeBatch).
		Where("pc.notified_time IS NULL").
		Where("pc.confirm_status = ?", model.PlanConfirmStatusWaiting).
		Where("pc.notify_at <= ?", now).
		Where("pc.deadline > ?", now).
		Where("pb.status = ? AND pb.finished_time IS NULL", model.PlanStatusEnabled).
		Where("p.status = ? AND p.finished_time IS NULL", model.PlanStatusEnabled).
		Order("pc.notify_at ASC").
		Limit(1).
		Scan(&candidate).Error
	if err != nil {
		return nil, err
	}
	if candidate.Id == "" {
		return nil, nil
	}
	return &candidate, nil
}

// MarkPlanConfirmNotified 标记批次确认记录已发送通知。
func MarkPlanConfirmNotified(ctx context.Context, db *gorm.DB, id string, now time.Time) error {
	return db.WithContext(ctx).Model(&PlanConfirm{}).
		Where("id = ?", id).
		Update("notified_time", sql.NullTime{Time: now, Valid: true}).Error
}

// FindPendingPlanConfirmExpire 查询一条已过截止时间且未确认的批次确认记录。
// 并发控制由调用方通过 Redis 锁完成；返回后需在锁内重新校验状态。
// 与 LockTriggerItem 一致：轮询 SELECT 静默成功日志，避免空转刷屏；错误与慢 SQL 仍会记录。
func FindPendingPlanConfirmExpire(ctx context.Context, db *gorm.DB, now time.Time) (*PlanConfirm, error) {
	var candidate PlanConfirm
	err := db.WithContext(gormx.WithoutSQLTrace(ctx)).
		Table("plan_confirm AS pc").
		Select("pc.*").
		Joins("JOIN plan_batch pb ON pb.id = pc.batch_pk AND pb.is_deleted = 0").
		Joins("JOIN plan p ON p.id = pc.plan_pk AND p.is_deleted = 0").
		Where("pc.is_deleted = ?", 0).
		Where("pc.confirm_type = ?", model.ConfirmTypeBatch).
		Where("pc.confirm_status = ?", model.PlanConfirmStatusWaiting).
		Where("pc.deadline <= ?", now).
		Where("pb.status = ? AND pb.finished_time IS NULL", model.PlanStatusEnabled).
		Where("p.status = ? AND p.finished_time IS NULL", model.PlanStatusEnabled).
		Order("pc.deadline ASC").
		Limit(1).
		Scan(&candidate).Error
	if err != nil {
		return nil, err
	}
	if candidate.Id == "" {
		return nil, nil
	}
	return &candidate, nil
}

// ClosePlanConfirmsByBatch 将批次下所有未确认的记录标记为已确认（计划/批次被终止）。
// 不区分 confirm_type：批次终止后，批次级和执行项级确认都应统一关闭。
func ClosePlanConfirmsByBatch(ctx context.Context, db *gorm.DB, batchPk, user, reason string, now time.Time) (int64, error) {
	return closePlanConfirms(ctx, db, "batch_pk", batchPk, user, reason, now)
}

// ClosePlanConfirmsByPlan 将计划下所有未确认的记录标记为已确认（计划被终止）。
// 不区分 confirm_type：计划终止后，批次级和执行项级确认都应统一关闭。
func ClosePlanConfirmsByPlan(ctx context.Context, db *gorm.DB, planPk, user, reason string, now time.Time) (int64, error) {
	return closePlanConfirms(ctx, db, "plan_pk", planPk, user, reason, now)
}

// closePlanConfirms 按指定维度关闭未确认记录。column 仅允许内部固定值（batch_pk / plan_pk）。
func closePlanConfirms(ctx context.Context, db *gorm.DB, column, value, user, reason string, now time.Time) (int64, error) {
	result := db.WithContext(ctx).Model(&PlanConfirm{}).
		Where(column+" = ?", value).
		Where("confirm_status = ?", model.PlanConfirmStatusWaiting).
		Updates(map[string]any{
			"confirm_status": model.PlanConfirmStatusConfirmed,
			"confirmed_time": sql.NullTime{Time: now, Valid: true},
			"confirmed_user": sql.NullString{String: user, Valid: user != ""},
			"expired_time":   sql.NullTime{Time: now, Valid: true},
			"reason":         sql.NullString{String: reason, Valid: reason != ""},
		})
	return result.RowsAffected, result.Error
}
