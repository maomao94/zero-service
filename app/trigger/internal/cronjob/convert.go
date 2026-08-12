package cronjob

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/crontask"

	"google.golang.org/protobuf/encoding/protojson"
)

// CronJobExtra 是 Trigger 业务字段在 TaskConfig.Extra 中的运行时封装。
// 不持久化到数据库，在 fromTaskConfig 时平铺到模型各列，
// 在 ToTaskConfig 时从模型列重建。
type CronJobExtra struct {
	DeptCode       string          `json:"deptCode"`
	Type           string          `json:"type"`
	GroupId        string          `json:"groupId,omitempty"`
	Description    string          `json:"description,omitempty"`
	Rule           json.RawMessage `json:"rule"`
	ExcludeDates   []string        `json:"excludeDates,omitempty"`
	SpecifiedTimes []string        `json:"specifiedTimes,omitempty"`
	ExcludedTimes  []string        `json:"excludedTimes,omitempty"`
	Ext1           string          `json:"ext1,omitempty"`
	Ext2           string          `json:"ext2,omitempty"`
	Ext3           string          `json:"ext3,omitempty"`
	Ext4           string          `json:"ext4,omitempty"`
	Ext5           string          `json:"ext5,omitempty"`
}

// fromTaskConfig 将 TaskConfig 转为 CronJob 模型。
// cfg.Extra 应为 CronJobExtra JSON，在此平铺到各 Trigger 专属列。
func fromTaskConfig(cfg *crontask.TaskConfig) (*gormmodel.CronJob, error) {
	record := &gormmodel.CronJob{
		TaskCode:         cfg.TaskCode,
		TaskName:         cfg.TaskName,
		RRuleStr:         cfg.RRuleStr,
		Priority:         cfg.Priority,
		LockTimeout:      cfg.LockTimeout.Milliseconds(),
		MaxDelay:         cfg.MaxDelay.Milliseconds() / 1000,
		Payload:          string(cfg.Payload),
		Status:           int(cfg.Status),
		StartTime:        toNullTime(cfg.StartTime),
		EndTime:          toNullTime(cfg.EndTime),
		NextRun:          toNullTime(cfg.NextRun),
		ScheduledTime:    toNullTime(cfg.ScheduledTime),
		LastRun:          toNullTime(cfg.LastRun),
		LastScheduledRun: toNullTime(cfg.LastScheduledRun),
	}
	extra, err := ParseExtra(cfg.Extra)
	if err != nil {
		return nil, err
	}
	record.DeptCode = extra.DeptCode
	record.Type = extra.Type
	record.GroupId = extra.GroupId
	record.Description = extra.Description
	record.Rule = string(extra.Rule)
	excludeDates, err := marshalOptionalStrings(extra.ExcludeDates)
	if err != nil {
		return nil, err
	}
	record.ExcludeDates = excludeDates
	specifiedTimes, err := marshalOptionalStrings(extra.SpecifiedTimes)
	if err != nil {
		return nil, err
	}
	record.SpecifiedTimes = specifiedTimes
	excludedTimes, err := marshalOptionalStrings(extra.ExcludedTimes)
	if err != nil {
		return nil, err
	}
	record.ExcludedTimes = excludedTimes
	record.Ext1 = extra.Ext1
	record.Ext2 = extra.Ext2
	record.Ext3 = extra.Ext3
	record.Ext4 = extra.Ext4
	record.Ext5 = extra.Ext5
	return record, nil
}

// ToTaskConfig 将 Cron Job 数据库记录转换为通用任务配置。
// TaskConfig.Extra 由模型各列重建 CronJobExtra 填入。
func ToTaskConfig(job *gormmodel.CronJob) (*crontask.TaskConfig, error) {
	extra, err := extraFromModel(job)
	if err != nil {
		return nil, err
	}
	extraJSON, err := MarshalExtra(extra)
	if err != nil {
		return nil, err
	}
	cfg := &crontask.TaskConfig{
		ID:          job.Id,
		CreateTime:  job.CreateTime,
		UpdateTime:  job.UpdateTime,
		TaskCode:    job.TaskCode,
		TaskName:    job.TaskName,
		RRuleStr:    job.RRuleStr,
		Priority:    job.Priority,
		LockTimeout: time.Duration(job.LockTimeout) * time.Millisecond,
		MaxDelay:    time.Duration(job.MaxDelay) * time.Second,
		Payload:     json.RawMessage(job.Payload),
		Extra:       extraJSON,
		Status:      crontask.TaskStatus(job.Status),
	}
	if job.StartTime.Valid {
		cfg.StartTime = job.StartTime.Time
	}
	if job.EndTime.Valid {
		cfg.EndTime = job.EndTime.Time
	}
	if job.NextRun.Valid {
		cfg.NextRun = job.NextRun.Time
	}
	if job.ScheduledTime.Valid {
		cfg.ScheduledTime = job.ScheduledTime.Time
	}
	if job.LastRun.Valid {
		cfg.LastRun = job.LastRun.Time
	}
	if job.LastScheduledRun.Valid {
		cfg.LastScheduledRun = job.LastScheduledRun.Time
	}
	return cfg, nil
}

// ToProto 将 Cron Job 任务配置转换为对外管理视图。
func ToProto(cfg *crontask.TaskConfig) (*trigger.CronJobPb, error) {
	extra, err := ParseExtra(cfg.Extra)
	if err != nil {
		return nil, err
	}
	var rule trigger.PlanRulePb
	if err := protojson.Unmarshal(extra.Rule, &rule); err != nil {
		return nil, fmt.Errorf("解析 Cron Job 规则失败: %w", err)
	}
	scheduleDescription, err := crontask.DescribeRRule(cfg.RRuleStr)
	if err != nil {
		return nil, fmt.Errorf("生成 Cron Job 规则描述失败: %w", err)
	}
	result := &trigger.CronJobPb{
		JobId:               cfg.ID,
		TaskCode:            cfg.TaskCode,
		TaskName:            cfg.TaskName,
		Priority:            int32(cfg.Priority),
		LockTimeout:         cfg.LockTimeout.Milliseconds(),
		MaxDelay:            int64(cfg.MaxDelay.Seconds()),
		Payload:             string(cfg.Payload),
		Status:              int32(cfg.Status),
		Type:                extra.Type,
		GroupId:             extra.GroupId,
		Description:         extra.Description,
		StartTime:           formatTime(cfg.StartTime),
		EndTime:             formatTime(cfg.EndTime),
		Rule:                &rule,
		ExcludeDates:        append([]string(nil), extra.ExcludeDates...),
		SpecifiedTimes:      append([]string(nil), extra.SpecifiedTimes...),
		ExcludedTimes:       append([]string(nil), extra.ExcludedTimes...),
		ScheduleDescription: scheduleDescription,
		RruleStr:            cfg.RRuleStr,
		Ext1:                extra.Ext1,
		Ext2:                extra.Ext2,
		Ext3:                extra.Ext3,
		Ext4:                extra.Ext4,
		Ext5:                extra.Ext5,
		CreateTime:          formatTime(cfg.CreateTime),
		UpdateTime:          formatTime(cfg.UpdateTime),
		DeptCode:            extra.DeptCode,
	}
	if !cfg.NextRun.IsZero() {
		result.NextRun = formatTime(cfg.NextRun)
	}
	if !cfg.LastRun.IsZero() {
		result.LastRun = formatTime(cfg.LastRun)
	}
	return result, nil
}

func extraFromModel(job *gormmodel.CronJob) (*CronJobExtra, error) {
	excludeDates, err := unmarshalOptionalStrings(job.ExcludeDates, "排除日期")
	if err != nil {
		return nil, err
	}
	specifiedTimes, err := unmarshalOptionalStrings(job.SpecifiedTimes, "指定执行时间")
	if err != nil {
		return nil, err
	}
	excludedTimes, err := unmarshalOptionalStrings(job.ExcludedTimes, "精确排除时间")
	if err != nil {
		return nil, err
	}
	return &CronJobExtra{
		DeptCode:       job.DeptCode,
		Type:           job.Type,
		GroupId:        job.GroupId,
		Description:    job.Description,
		Rule:           json.RawMessage(job.Rule),
		ExcludeDates:   excludeDates,
		SpecifiedTimes: specifiedTimes,
		ExcludedTimes:  excludedTimes,
		Ext1:           job.Ext1,
		Ext2:           job.Ext2,
		Ext3:           job.Ext3,
		Ext4:           job.Ext4,
		Ext5:           job.Ext5,
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

// MarshalExtra 序列化 Trigger 业务字段。
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

func unmarshalOptionalStrings(value sql.NullString, field string) ([]string, error) {
	if !value.Valid || value.String == "" {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal([]byte(value.String), &values); err != nil {
		return nil, fmt.Errorf("解析%s失败: %w", field, err)
	}
	return values, nil
}
