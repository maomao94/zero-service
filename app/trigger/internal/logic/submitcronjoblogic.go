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

type SubmitCronJobLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSubmitCronJobLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitCronJobLogic {
	return &SubmitCronJobLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 按 TaskCode 幂等提交完整 Cron Job 配置，不存在时创建，存在时更新
func (l *SubmitCronJobLogic) SubmitCronJob(in *trigger.SubmitCronJobReq) (*trigger.SubmitCronJobRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	task, err := buildCronJobTask(cronJobTaskData{
		taskCode: in.TaskCode, taskName: in.TaskName, taskType: in.Type,
		groupID: in.GroupId, description: in.Description, deptCode: in.DeptCode, rule: in.Rule,
		startTime: in.StartTime, endTime: in.EndTime, excludeDates: in.ExcludeDates,
		priority: in.Priority, payload: in.Payload, bizExtra: in.Extra,
		lockTimeout: in.LockTimeout, maxDelay: in.MaxDelay, skipTimeFilter: in.SkipTimeFilter,
		ext1: in.Ext1, ext2: in.Ext2, ext3: in.Ext3, ext4: in.Ext4, ext5: in.Ext5,
	})
	if err != nil {
		return nil, err
	}
	existing, err := l.svcCtx.CronJobStore.GetByCode(l.ctx, task.TaskCode)
	if err == nil {
		task.ID = existing.ID
		task.Status = existing.Status
		if err := l.svcCtx.CronJobStore.Update(l.ctx, task); err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "提交更新 Cron Job 失败")
		}
	} else if errors.Is(err, crontask.ErrNotFound) {
		if err := l.svcCtx.CronJobStore.Insert(l.ctx, task); err != nil {
			if !errors.Is(err, crontask.ErrDuplicate) {
				return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "提交创建 Cron Job 失败")
			}
			existing, err = l.svcCtx.CronJobStore.GetByCode(l.ctx, task.TaskCode)
			if errors.Is(err, crontask.ErrNotFound) {
				return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_ALREADY_EXIST)
			}
			if err != nil {
				return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "重新查询待提交 Cron Job 失败")
			}
			task.ID = existing.ID
			task.Status = existing.Status
			if err := l.svcCtx.CronJobStore.Update(l.ctx, task); err != nil {
				return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "提交更新 Cron Job 失败")
			}
		}
	} else {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询待提交 Cron Job 失败")
	}

	nextRun := ""
	if !task.NextRun.IsZero() {
		nextRun = tool.CarbonFromTimeStartOfSecond(task.NextRun).ToDateTimeString()
	}
	return &trigger.SubmitCronJobRes{JobId: task.ID, NextRun: nextRun, TaskCode: task.TaskCode}, nil
}
