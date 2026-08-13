package logic

import (
	"context"
	"errors"

	"zero-service/app/trigger/internal/cronjob"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/carbonx"
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
	extra, err := cronjob.ParseExtra(existing.Extra)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "解析已有 Cron Job 身份失败")
	}
	task, groupID, err := buildCronJobTask(cronJobTaskData{
		taskCode: existing.TaskCode, taskName: in.TaskName, taskType: extra.Type,
		groupID: extra.GroupId, description: in.Description, deptCode: extra.DeptCode, rule: in.Rule,
		startTime: in.StartTime, endTime: in.EndTime, excludeDates: in.ExcludeDates,
		specifiedTimes: in.SpecifiedTimes, excludedTimes: in.ExcludedTimes,
		priority: in.Priority, payload: in.Payload,
		lockTimeout: in.LockTimeout, maxDelay: in.MaxDelay, skipTimeFilter: in.SkipTimeFilter,
		ext1: in.Ext1, ext2: in.Ext2, ext3: in.Ext3, ext4: in.Ext4, ext5: in.Ext5,
	})
	if err != nil {
		return nil, err
	}
	task.ID = existing.ID
	task.Status = existing.Status
	if err := l.svcCtx.CronJobStore.Update(l.ctx, task); err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "更新 Cron Job 失败")
	}
	nextRun := ""
	if !task.NextRun.IsZero() {
		nextRun = carbonx.FormatDateTime(task.NextRun)
	}
	return &trigger.UpdateCronJobRes{JobId: task.ID, NextRun: nextRun, TaskCode: task.TaskCode, GroupId: groupID}, nil
}
