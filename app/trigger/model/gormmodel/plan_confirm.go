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
	DeptCode   sql.NullString `gorm:"column:dept_code;size:64;comment:机构code;index:idx_plan_confirm_dept_code"`

	PlanPk      string         `gorm:"column:plan_pk;size:64;comment:关联的计划主键ID;index:idx_plan_confirm_plan_pk"`
	BatchPk     string         `gorm:"column:batch_pk;size:64;comment:批次主键ID;index:idx_plan_confirm_batch_pk"`
	ExecPk      sql.NullString `gorm:"column:exec_pk;size:64;comment:执行项主键ID（执行项级确认使用，暂未实现）"`
	ConfirmType int            `gorm:"column:confirm_type;comment:确认类型：1-批次确认，2-执行项确认（暂未实现）;uniqueIndex:uk_plan_confirm_target,priority:1"`
	TargetPk    string         `gorm:"column:target_pk;size:64;comment:确认目标主键ID;uniqueIndex:uk_plan_confirm_target,priority:2"`

	LeadMinutes   int            `gorm:"column:lead_minutes;comment:提前通知分钟数，0-到点通知"`
	NotifyAt      time.Time      `gorm:"column:notify_at;comment:计划通知时间;index:idx_plan_confirm_notify,priority:2"`
	NotifiedTime  sql.NullTime   `gorm:"column:notified_time;comment:实际通知时间;index:idx_plan_confirm_notify,priority:1"`
	Deadline      time.Time      `gorm:"column:deadline;comment:确认截止时间;index:idx_plan_confirm_expire,priority:3"`
	ConfirmedTime sql.NullTime   `gorm:"column:confirmed_time;comment:确认时间;index:idx_plan_confirm_expire,priority:1"`
	ConfirmedUser sql.NullString `gorm:"column:confirmed_user;size:64;comment:确认人"`
	ExpiredTime   sql.NullTime   `gorm:"column:expired_time;comment:超时终止时间;index:idx_plan_confirm_expire,priority:2"`
}

func (PlanConfirm) TableName() string { return "plan_confirm" }

// BatchConfirmDeadline 返回批次确认的默认截止时间：批次触发时间当天 23:59:59（东八区）。
func BatchConfirmDeadline(triggerTime time.Time) time.Time {
	return carbon.CreateFromStdTime(triggerTime, carbon.Shanghai).EndOfDay().StdTime()
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

// ConfirmPlanTarget 确认一个目标，只有未确认且未过期时才会更新成功。
func ConfirmPlanTarget(ctx context.Context, db *gorm.DB, confirmType int, targetPk, user string, now time.Time) (int64, error) {
	result := db.WithContext(ctx).Model(&PlanConfirm{}).
		Where("confirm_type = ? AND target_pk = ?", confirmType, targetPk).
		Where("confirmed_time IS NULL").
		Where("expired_time IS NULL").
		Updates(map[string]any{
			"confirmed_time": sql.NullTime{Time: now, Valid: true},
			"confirmed_user": sql.NullString{String: user, Valid: user != ""},
		})
	return result.RowsAffected, result.Error
}

// ClaimPlanConfirmNotify 抢占一条待通知的批次确认记录，多实例下由条件更新保证只发一次。
// 只处理计划与批次均处于启用状态的记录；暂停恢复后 notify_at 已过期会立即补发。
// 没有可通知记录时返回 nil。
func ClaimPlanConfirmNotify(ctx context.Context, db *gorm.DB, now time.Time) (*PlanConfirm, error) {
	var candidate PlanConfirm
	err := db.WithContext(ctx).
		Table("plan_confirm AS pc").
		Select("pc.*").
		Joins("JOIN plan_batch pb ON pb.id = pc.batch_pk AND pb.is_deleted = 0").
		Joins("JOIN plan p ON p.id = pc.plan_pk AND p.is_deleted = 0").
		Where("pc.confirm_type = ?", model.ConfirmTypeBatch).
		Where("pc.notified_time IS NULL").
		Where("pc.confirmed_time IS NULL").
		Where("pc.expired_time IS NULL").
		Where("pc.notify_at <= ?", now).
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
	result := db.WithContext(ctx).Model(&PlanConfirm{}).
		Where("id = ? AND notified_time IS NULL", candidate.Id).
		Update("notified_time", sql.NullTime{Time: now, Valid: true})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	candidate.NotifiedTime = sql.NullTime{Time: now, Valid: true}
	return &candidate, nil
}

// ClaimPlanConfirmExpire 抢占一条已过截止时间且未确认的批次确认记录，多实例下幂等。
// 只处理计划与批次均处于启用状态的记录，人工暂停或已结束的批次不会触发超时终止。
// 没有可终止记录时返回 nil。
func ClaimPlanConfirmExpire(ctx context.Context, db *gorm.DB, now time.Time) (*PlanConfirm, error) {
	var candidate PlanConfirm
	err := db.WithContext(ctx).
		Table("plan_confirm AS pc").
		Select("pc.*").
		Joins("JOIN plan_batch pb ON pb.id = pc.batch_pk AND pb.is_deleted = 0").
		Joins("JOIN plan p ON p.id = pc.plan_pk AND p.is_deleted = 0").
		Where("pc.confirm_type = ?", model.ConfirmTypeBatch).
		Where("pc.confirmed_time IS NULL").
		Where("pc.expired_time IS NULL").
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
	result := db.WithContext(ctx).Model(&PlanConfirm{}).
		Where("id = ? AND expired_time IS NULL", candidate.Id).
		Update("expired_time", sql.NullTime{Time: now, Valid: true})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	candidate.ExpiredTime = sql.NullTime{Time: now, Valid: true}
	return &candidate, nil
}

// ExpirePlanConfirmsByBatch 将批次下未确认的确认记录标记为过期。
func ExpirePlanConfirmsByBatch(ctx context.Context, db *gorm.DB, batchPk string, now time.Time) (int64, error) {
	result := db.WithContext(ctx).Model(&PlanConfirm{}).
		Where("confirm_type = ? AND batch_pk = ?", model.ConfirmTypeBatch, batchPk).
		Where("confirmed_time IS NULL").
		Where("expired_time IS NULL").
		Update("expired_time", sql.NullTime{Time: now, Valid: true})
	return result.RowsAffected, result.Error
}

// ExpirePlanConfirmsByPlan 将计划下未确认的确认记录标记为过期。
func ExpirePlanConfirmsByPlan(ctx context.Context, db *gorm.DB, planPk string, now time.Time) (int64, error) {
	result := db.WithContext(ctx).Model(&PlanConfirm{}).
		Where("confirm_type = ? AND plan_pk = ?", model.ConfirmTypeBatch, planPk).
		Where("confirmed_time IS NULL").
		Where("expired_time IS NULL").
		Update("expired_time", sql.NullTime{Time: now, Valid: true})
	return result.RowsAffected, result.Error
}

// TerminateExecItemsByBatch 将批次下未结束的执行项全部置为终止。
func TerminateExecItemsByBatch(ctx context.Context, db *gorm.DB, batchPk, reason string, now time.Time) (int64, error) {
	result := db.WithContext(ctx).Model(&PlanExecItem{}).
		Where("batch_pk = ?", batchPk).
		Where("status NOT IN ?", []int{model.StatusCompleted, model.StatusTerminated}).
		Updates(map[string]any{
			"status":            model.StatusTerminated,
			"last_result":       model.ResultTerminated,
			"last_message":      reason,
			"last_reason":       reason,
			"terminated_reason": reason,
			"last_trigger_time": now,
			"trigger_count":     gorm.Expr("trigger_count + 1"),
		})
	return result.RowsAffected, result.Error
}
