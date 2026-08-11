package logic

import (
	"context"
	"errors"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/crontask"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateCronJobLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateCronJobLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCronJobLogic {
	return &UpdateCronJobLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 按 Trigger JobId 更新完整 Cron Job 配置，保留原 TaskCode
func (l *UpdateCronJobLogic) UpdateCronJob(in *trigger.UpdateCronJobReq) (*trigger.UpdateCronJobRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	existing, err := l.svcCtx.CronJobStore.GetByID(l.ctx, in.JobId)
	if err != nil {
		if errors.Is(err, crontask.ErrNotFound) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
		}
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询待更新 Cron Job 失败")
	}
	task, err := buildCronJobTask(cronJobTaskData{
		taskCode: existing.TaskCode, taskName: in.TaskName, taskType: in.Type,
		groupID: in.GroupId, description: in.Description, deptCode: in.DeptCode, rule: in.Rule,
		startTime: in.StartTime, endTime: in.EndTime, excludeDates: in.ExcludeDates,
		priority: in.Priority, payload: in.Payload, bizExtra: in.Extra,
		lockTimeout: in.LockTimeout, maxDelay: in.MaxDelay, skipTimeFilter: in.SkipTimeFilter,
		ext1: in.Ext1, ext2: in.Ext2, ext3: in.Ext3, ext4: in.Ext4, ext5: in.Ext5,
	})
	if err != nil {
		return nil, err
	}
	task.ID = existing.ID
	task.Status = existing.Status
	if err := l.svcCtx.CronJobStore.Update(l.ctx, task); err != nil {
		if errors.Is(err, crontask.ErrNotFound) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
		}
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "更新 Cron Job 失败")
	}
	nextRun := ""
	if !task.NextRun.IsZero() {
		nextRun = tool.CarbonFromTimeStartOfSecond(task.NextRun).ToDateTimeString()
	}
	return &trigger.UpdateCronJobRes{JobId: task.ID, NextRun: nextRun, TaskCode: task.TaskCode}, nil
}
