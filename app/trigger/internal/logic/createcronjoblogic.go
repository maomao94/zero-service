package logic

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"zero-service/app/trigger/internal/cronjob"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
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

// 创建基于 RRULE 的周期任务，返回 Trigger 生成的 JobId
func (l *CreateCronJobLogic) CreateCronJob(in *trigger.CreateCronJobReq) (*trigger.CreateCronJobRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	payload, err := optionalJSON(in.Payload, "payload")
	if err != nil {
		return nil, err
	}
	bizExtra, err := optionalJSON(in.Extra, "extra")
	if err != nil {
		return nil, err
	}
	schedule, err := cronjob.CompileSchedule(
		in.Rule,
		in.StartTime,
		in.EndTime,
		in.ExcludeDates,
		in.SkipTimeFilter,
		tool.NowStartOfSecond().StdTime(),
	)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, err, "Cron Job 规则无效")
	}
	extra, err := cronjob.MarshalExtra(&cronjob.CronJobExtra{
		DeptCode:     in.DeptCode,
		Type:         in.Type,
		GroupId:      in.GroupId,
		Description:  in.Description,
		StartTime:    in.StartTime,
		EndTime:      in.EndTime,
		Rule:         schedule.RuleJSON,
		ExcludeDates: append([]string(nil), in.ExcludeDates...),
		BizExtra:     bizExtra,
		Ext1:         in.Ext1,
		Ext2:         in.Ext2,
		Ext3:         in.Ext3,
		Ext4:         in.Ext4,
		Ext5:         in.Ext5,
	})
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, err, "Cron Job 扩展字段无效")
	}
	task := &crontask.TaskConfig{
		TaskCode:    in.TaskCode,
		TaskName:    in.TaskName,
		RRuleStr:    schedule.RRuleStr,
		Priority:    int(in.Priority),
		LockTimeout: time.Duration(in.LockTimeout) * time.Millisecond,
		Payload:     payload,
		Extra:       extra,
		Status:      crontask.StatusEnabled,
		NextRun:     schedule.NextRun,
	}
	if err := l.svcCtx.CronJobStore.Insert(l.ctx, task); err != nil {
		if errors.Is(err, crontask.ErrDuplicate) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_ALREADY_EXIST)
		}
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "创建 Cron Job 失败")
	}
	nextRun := ""
	if !task.NextRun.IsZero() {
		nextRun = tool.CarbonFromTimeStartOfSecond(task.NextRun).ToDateTimeString()
	}
	return &trigger.CreateCronJobRes{JobId: task.ID, NextRun: nextRun}, nil
}

func optionalJSON(value, field string) (json.RawMessage, error) {
	if value == "" {
		return nil, nil
	}
	if !json.Valid([]byte(value)) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, field+" 必须是合法 JSON")
	}
	return json.RawMessage(value), nil
}
