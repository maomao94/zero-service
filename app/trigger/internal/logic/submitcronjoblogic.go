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

func (l *SubmitCronJobLogic) SubmitCronJob(in *trigger.SubmitCronJobReq) (*trigger.SubmitCronJobRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	existing, err := l.svcCtx.CronJobStore.GetByCode(l.ctx, in.TaskCode)
	if err == nil {
		updated, err := NewUpdateCronJobLogic(l.ctx, l.svcCtx).UpdateCronJob(&trigger.UpdateCronJobReq{
			JobId: existing.ID, TaskName: in.TaskName,
			Description: in.Description, Rule: in.Rule,
			StartTime: in.StartTime, EndTime: in.EndTime,
			ExcludeDates: in.ExcludeDates, Priority: in.Priority, Payload: in.Payload,
			SpecifiedTimes: in.SpecifiedTimes, ExcludedTimes: in.ExcludedTimes,
			LockTimeout: in.LockTimeout, MaxDelay: in.MaxDelay, SkipTimeFilter: in.SkipTimeFilter,
			Ext1: in.Ext1, Ext2: in.Ext2, Ext3: in.Ext3, Ext4: in.Ext4, Ext5: in.Ext5,
		})
		if err != nil {
			return nil, err
		}
		return &trigger.SubmitCronJobRes{
			JobId: updated.JobId, NextRun: updated.NextRun,
			TaskCode: updated.TaskCode, GroupId: updated.GroupId,
		}, nil
	}
	if !errors.Is(err, crontask.ErrNotFound) {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询待提交 Cron Job 失败")
	}
	if in.Type == "" || in.DeptCode == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "Submit 新建 Cron Job 时 type 和 dept_code 不能为空")
	}
	created, err := NewCreateCronJobLogic(l.ctx, l.svcCtx).CreateCronJob(&trigger.CreateCronJobReq{
		TaskCode: in.TaskCode, TaskName: in.TaskName, Type: in.Type,
		GroupId: in.GroupId, Description: in.Description, DeptCode: in.DeptCode,
		Rule: in.Rule, StartTime: in.StartTime, EndTime: in.EndTime,
		ExcludeDates: in.ExcludeDates, Priority: in.Priority, Payload: in.Payload,
		SpecifiedTimes: in.SpecifiedTimes, ExcludedTimes: in.ExcludedTimes,
		LockTimeout: in.LockTimeout, MaxDelay: in.MaxDelay, SkipTimeFilter: in.SkipTimeFilter,
		Ext1: in.Ext1, Ext2: in.Ext2, Ext3: in.Ext3, Ext4: in.Ext4, Ext5: in.Ext5,
	})
	if err != nil {
		return nil, err
	}
	return &trigger.SubmitCronJobRes{
		JobId: created.JobId, NextRun: created.NextRun,
		TaskCode: in.TaskCode, GroupId: created.GroupId,
	}, nil
}
