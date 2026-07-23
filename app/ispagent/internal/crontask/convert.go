package crontask

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"zero-service/app/ispagent/model/gormmodel"
	"zero-service/common/crontask"
)

// fromTaskConfig 将 crontask.TaskConfig 转为 GormTaskConfig 持久化模型。
// Extra 中的 ISP 字段会被反序列化并平铺到 GormTaskConfig 对应列。
func fromTaskConfig(cfg *crontask.TaskConfig) *gormmodel.GormTaskConfig {
	g := &gormmodel.GormTaskConfig{
		TaskCode: cfg.TaskCode,
		TaskName: cfg.TaskName,
		RRuleStr: cfg.RRuleStr,
		Priority: cfg.Priority,
		Payload:  string(cfg.Payload),
		Extra:    string(cfg.Extra),
		Status:   int(cfg.Status),
		NextRun:  toNullTime(cfg.NextRun),
		LastRun:  toNullTime(cfg.LastRun),
	}
	g.Id = cfg.ID

	if fields := DeserializeExtra(string(cfg.Extra)); fields != nil {
		applyFields(g, fields)
	}

	return g
}

// toTaskConfig 将 GormTaskConfig 转为 crontask.TaskConfig。
// ISP 字段会被序列化到 TaskConfig.Extra JSON 中。
func toTaskConfig(g *gormmodel.GormTaskConfig) *crontask.TaskConfig {
	cfg := &crontask.TaskConfig{
		ID:       g.Id,
		TaskCode: g.TaskCode,
		TaskName: g.TaskName,
		RRuleStr: g.RRuleStr,
		Priority: g.Priority,
		Payload:  json.RawMessage(g.Payload),
		Status:   crontask.TaskStatus(g.Status),
	}
	if g.NextRun.Valid {
		cfg.NextRun = g.NextRun.Time
	}
	if g.LastRun.Valid {
		cfg.LastRun = g.LastRun.Time
	}

	fields := toFields(g)
	cfg.Extra = json.RawMessage(SerializeExtra(fields))
	return cfg
}

func toNullTime(value time.Time) sql.NullTime {
	return sql.NullTime{Time: value, Valid: !value.IsZero()}
}

// applyFields 将 IspTaskFields 的值平铺设置到 GormTaskConfig 对应列。
func applyFields(g *gormmodel.GormTaskConfig, f *IspTaskFields) {
	g.SubstationCode = f.SubstationCode
	g.PatrolType = f.PatrolType
	g.DeviceLevel = f.DeviceLevel
	g.DeviceList = f.DeviceList
	g.FixedStartTime = f.FixedStartTime
	g.CycleMonth = f.CycleMonth
	g.CycleWeek = f.CycleWeek
	g.CycleExecuteTime = f.CycleExecuteTime
	g.CycleStartTime = f.CycleStartTime
	g.CycleEndTime = f.CycleEndTime
	g.IntervalNumber = f.IntervalNumber
	g.IntervalType = f.IntervalType
	g.IntervalExecuteTime = f.IntervalExecuteTime
	g.IntervalStartTime = f.IntervalStartTime
	g.IntervalEndTime = f.IntervalEndTime
	g.InvalidStartTime = f.InvalidStartTime
	g.InvalidEndTime = f.InvalidEndTime
	g.IsEnable = f.IsEnable
	g.IspCreator = f.Creator
	g.IspCreateTime = f.CreateTime
}

// toFields 从 GormTaskConfig 的列值重建 IspTaskFields。
func toFields(g *gormmodel.GormTaskConfig) *IspTaskFields {
	return &IspTaskFields{
		SubstationCode:      g.SubstationCode,
		PatrolType:          g.PatrolType,
		TaskCode:            g.TaskCode,
		TaskName:            g.TaskName,
		Priority:            strconv.Itoa(g.Priority),
		DeviceLevel:         g.DeviceLevel,
		DeviceList:          g.DeviceList,
		FixedStartTime:      g.FixedStartTime,
		CycleMonth:          g.CycleMonth,
		CycleWeek:           g.CycleWeek,
		CycleExecuteTime:    g.CycleExecuteTime,
		CycleStartTime:      g.CycleStartTime,
		CycleEndTime:        g.CycleEndTime,
		IntervalNumber:      g.IntervalNumber,
		IntervalType:        g.IntervalType,
		IntervalExecuteTime: g.IntervalExecuteTime,
		IntervalStartTime:   g.IntervalStartTime,
		IntervalEndTime:     g.IntervalEndTime,
		InvalidStartTime:    g.InvalidStartTime,
		InvalidEndTime:      g.InvalidEndTime,
		IsEnable:            g.IsEnable,
		Creator:             g.IspCreator,
		CreateTime:          g.IspCreateTime,
	}
}
