package logic

import (
	"context"
	"errors"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/carbonx"
	"zero-service/common/crontask"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateCronJobLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateCronJobLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateCronJobLogic {
	return &CreateCronJobLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateCronJobLogic) CreateCronJob(in *trigger.CreateCronJobReq) (*trigger.CreateCronJobRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	task, groupID, err := buildCronJobTask(cronJobTaskData{
		taskCode: in.TaskCode, taskName: in.TaskName, taskType: in.Type,
		groupID: in.GroupId, description: in.Description, deptCode: in.DeptCode, rule: in.Rule,
		startTime: in.StartTime, endTime: in.EndTime, excludeDates: in.ExcludeDates,
		specifiedTimes: in.SpecifiedTimes, excludedTimes: in.ExcludedTimes,
		priority: in.Priority, payload: in.Payload,
		lockTimeout: in.LockTimeout, maxDelay: in.MaxDelay, skipTimeFilter: in.SkipTimeFilter,
		ext1: in.Ext1, ext2: in.Ext2, ext3: in.Ext3, ext4: in.Ext4, ext5: in.Ext5,
	})
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.CronJobStore.Insert(l.ctx, task); err != nil {
		if errors.Is(err, crontask.ErrDuplicate) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_ALREADY_EXIST)
		}
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "创建 Cron Job 失败")
	}
	nextRun := ""
	if !task.NextRun.IsZero() {
		nextRun = carbonx.FormatDateTime(task.NextRun)
	}
	return &trigger.CreateCronJobRes{JobId: task.ID, NextRun: nextRun, GroupId: groupID}, nil
}
