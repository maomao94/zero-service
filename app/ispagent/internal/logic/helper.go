package logic

import (
	"fmt"

	"zero-service/app/ispagent/ispagent"
	"zero-service/app/ispagent/model/gormmodel"
	"zero-service/common/crontask"

	"github.com/dromara/carbon/v2"
)

// toTaskConfigItemPb 将任务配置持久化模型转换为 RPC 视图。
func toTaskConfigItemPb(record *gormmodel.GormTaskConfig) (*ispagent.TaskConfigItem, error) {
	description, err := crontask.DescribeRRule(record.RRuleStr)
	if err != nil {
		return nil, fmt.Errorf("生成任务 %s 规则描述失败: %w", record.TaskCode, err)
	}
	item := &ispagent.TaskConfigItem{
		Id:                  record.Id,
		TaskCode:            record.TaskCode,
		TaskName:            record.TaskName,
		Priority:            int32(record.Priority),
		LockTimeout:         record.LockTimeout,
		RruleStr:            record.RRuleStr,
		Status:              int32(record.Status),
		SubstationCode:      record.SubstationCode,
		PatrolType:          record.PatrolType,
		DeviceLevel:         int32(record.DeviceLevel),
		DeviceList:          record.DeviceList,
		IspEnable:           record.IsEnable,
		IspCreator:          record.IspCreator,
		IspCreateTime:       record.IspCreateTime,
		FixedStartTime:      record.FixedStartTime,
		CycleMonth:          record.CycleMonth,
		CycleWeek:           record.CycleWeek,
		CycleExecuteTime:    record.CycleExecuteTime,
		CycleStartTime:      record.CycleStartTime,
		CycleEndTime:        record.CycleEndTime,
		IntervalNumber:      record.IntervalNumber,
		IntervalType:        record.IntervalType,
		IntervalExecuteTime: record.IntervalExecuteTime,
		IntervalStartTime:   record.IntervalStartTime,
		IntervalEndTime:     record.IntervalEndTime,
		InvalidStartTime:    record.InvalidStartTime,
		InvalidEndTime:      record.InvalidEndTime,
		ScheduleDescription: description,
	}
	if record.NextRun.Valid {
		item.NextRun = carbon.CreateFromStdTime(record.NextRun.Time).ToDateTimeString()
	}
	if record.LastRun.Valid {
		item.LastRun = carbon.CreateFromStdTime(record.LastRun.Time).ToDateTimeString()
	}
	return item, nil
}
