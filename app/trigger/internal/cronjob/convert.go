package cronjob

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"zero-service/app/trigger/model/gormmodel"
	"zero-service/common/crontask"
)

// CronJobExtra 是 Trigger 业务字段在 TaskConfig.Extra 中的稳定封装。
// BizExtra 保存调用方原始扩展 JSON，避免与 Trigger 保留字段发生冲突。
type CronJobExtra struct {
	// DeptCode 是机构编码。
	DeptCode string `json:"deptCode"`
	// Type 是调用方定义的任务类型。
	Type string `json:"type"`
	// GroupId 是调用方的任务分组 ID。
	GroupId string `json:"groupId,omitempty"`
	// Description 是任务描述。
	Description string `json:"description,omitempty"`
	// StartTime 是调用方提交的规则开始时间；未提交时为空。
	StartTime string `json:"startTime,omitempty"`
	// EndTime 是调用方提交的规则结束时间；未提交时为空。
	EndTime string `json:"endTime,omitempty"`
	// Rule 是调用方提交的 PlanRulePb JSON。
	Rule json.RawMessage `json:"rule"`
	// ExcludeDates 是调用方提交的排除日期列表；未提交时为空。
	ExcludeDates []string `json:"excludeDates,omitempty"`
	// BizExtra 保存调用方原始扩展 JSON。
	BizExtra json.RawMessage `json:"bizExtra,omitempty"`
	// Ext1 至 Ext5 是调用方保留扩展字段。
	Ext1 string `json:"ext1,omitempty"`
	Ext2 string `json:"ext2,omitempty"`
	Ext3 string `json:"ext3,omitempty"`
	Ext4 string `json:"ext4,omitempty"`
	Ext5 string `json:"ext5,omitempty"`
}

func fromTaskConfig(cfg *crontask.TaskConfig) (*gormmodel.CronJob, error) {
	extra, err := ParseExtra(cfg.Extra)
	if err != nil {
		return nil, err
	}
	startTime, err := parseOptionalTime(extra.StartTime)
	if err != nil {
		return nil, fmt.Errorf("解析开始时间失败: %w", err)
	}
	endTime, err := parseOptionalTime(extra.EndTime)
	if err != nil {
		return nil, fmt.Errorf("解析结束时间失败: %w", err)
	}
	excludeDates, err := marshalOptionalStrings(extra.ExcludeDates)
	if err != nil {
		return nil, fmt.Errorf("序列化排除日期失败: %w", err)
	}
	return &gormmodel.CronJob{
		TaskCode:     cfg.TaskCode,
		TaskName:     cfg.TaskName,
		RRuleStr:     cfg.RRuleStr,
		Priority:     cfg.Priority,
		LockTimeout:  cfg.LockTimeout.Milliseconds(),
		Payload:      string(cfg.Payload),
		Extra:        string(cfg.Extra),
		Status:       int(cfg.Status),
		NextRun:      toNullTime(cfg.NextRun),
		LastRun:      toNullTime(cfg.LastRun),
		DeptCode:     extra.DeptCode,
		Type:         extra.Type,
		GroupId:      extra.GroupId,
		Description:  extra.Description,
		StartTime:    startTime,
		EndTime:      endTime,
		Rule:         string(extra.Rule),
		ExcludeDates: excludeDates,
		Ext1:         extra.Ext1,
		Ext2:         extra.Ext2,
		Ext3:         extra.Ext3,
		Ext4:         extra.Ext4,
		Ext5:         extra.Ext5,
	}, nil
}

func toTaskConfig(job *gormmodel.CronJob) (*crontask.TaskConfig, error) {
	extra, err := extraFromModel(job)
	if err != nil {
		return nil, err
	}
	extraJSON, err := json.Marshal(extra)
	if err != nil {
		return nil, fmt.Errorf("序列化 Cron Job Extra 失败: %w", err)
	}
	cfg := &crontask.TaskConfig{
		ID:          job.Id,
		TaskCode:    job.TaskCode,
		TaskName:    job.TaskName,
		RRuleStr:    job.RRuleStr,
		Priority:    job.Priority,
		LockTimeout: time.Duration(job.LockTimeout) * time.Millisecond,
		Payload:     json.RawMessage(job.Payload),
		Extra:       extraJSON,
		Status:      crontask.TaskStatus(job.Status),
	}
	if job.NextRun.Valid {
		cfg.NextRun = job.NextRun.Time
	}
	if job.LastRun.Valid {
		cfg.LastRun = job.LastRun.Time
	}
	return cfg, nil
}

func extraFromModel(job *gormmodel.CronJob) (*CronJobExtra, error) {
	stored, err := ParseExtra(json.RawMessage(job.Extra))
	if err != nil {
		return nil, err
	}
	var excludeDates []string
	if job.ExcludeDates.Valid {
		if err := json.Unmarshal([]byte(job.ExcludeDates.String), &excludeDates); err != nil {
			return nil, fmt.Errorf("解析排除日期失败: %w", err)
		}
	}
	return &CronJobExtra{
		DeptCode:     job.DeptCode,
		Type:         job.Type,
		GroupId:      job.GroupId,
		Description:  job.Description,
		StartTime:    formatOptionalTime(job.StartTime),
		EndTime:      formatOptionalTime(job.EndTime),
		Rule:         json.RawMessage(job.Rule),
		ExcludeDates: excludeDates,
		BizExtra:     stored.BizExtra,
		Ext1:         job.Ext1,
		Ext2:         job.Ext2,
		Ext3:         job.Ext3,
		Ext4:         job.Ext4,
		Ext5:         job.Ext5,
	}, nil
}

// ParseExtra 解析 TaskConfig.Extra 中的 Trigger 业务字段。
func ParseExtra(value json.RawMessage) (*CronJobExtra, error) {
	if len(value) == 0 {
		return nil, errors.New("Cron Job Extra 不能为空")
	}
	var extra CronJobExtra
	if err := json.Unmarshal(value, &extra); err != nil {
		return nil, fmt.Errorf("解析 Cron Job Extra 失败: %w", err)
	}
	return &extra, nil
}

// MarshalExtra 序列化 Trigger 业务字段，供创建 Logic 构造 TaskConfig 使用。
func MarshalExtra(extra *CronJobExtra) (json.RawMessage, error) {
	value, err := json.Marshal(extra)
	if err != nil {
		return nil, fmt.Errorf("序列化 Cron Job Extra 失败: %w", err)
	}
	return value, nil
}

func toNullTime(value time.Time) sql.NullTime {
	return sql.NullTime{Time: value, Valid: !value.IsZero()}
}

func parseOptionalTime(value string) (sql.NullTime, error) {
	if value == "" {
		return sql.NullTime{}, nil
	}
	parsed, err := time.ParseInLocation(dateTimeLayout, value, time.Local)
	if err != nil {
		return sql.NullTime{}, err
	}
	return sql.NullTime{Time: parsed, Valid: true}, nil
}

func formatOptionalTime(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format(dateTimeLayout)
}

func marshalOptionalStrings(values []string) (sql.NullString, error) {
	if len(values) == 0 {
		return sql.NullString{}, nil
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(encoded), Valid: true}, nil
}
