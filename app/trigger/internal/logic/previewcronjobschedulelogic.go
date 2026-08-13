package logic

import (
	"context"
	"errors"
	"time"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/carbonx"
	"zero-service/common/crontask"
	"zero-service/common/rrulex"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type PreviewCronJobScheduleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPreviewCronJobScheduleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreviewCronJobScheduleLogic {
	return &PreviewCronJobScheduleLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 预览指定 Cron Job 从当前时间之后的计划执行时间，不改变任务状态
func (l *PreviewCronJobScheduleLogic) PreviewCronJobSchedule(in *trigger.PreviewCronJobScheduleReq) (*trigger.PreviewCronJobScheduleRes, error) {
	if in == nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, errors.New("request is nil"), "预览 Cron Job 调度请求无效")
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if l.svcCtx == nil || l.svcCtx.CronJobStore == nil || l.svcCtx.CronJobScheduler == nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, errors.New("Cron Job preview dependencies are not initialized"), "预览 Cron Job 调度规则失败")
	}
	task, err := l.svcCtx.CronJobStore.GetByID(l.ctx, in.JobId)
	if err != nil {
		if errors.Is(err, crontask.ErrNotFound) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
		}
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询 Cron Job 失败")
	}

	count := int(in.Count)
	if count == 0 {
		count = 10
	}
	runs, err := l.svcCtx.CronJobScheduler.PreviewNextRuns(task, time.Now(), count)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, err, "解析 Cron Job 调度规则失败")
	}
	description, err := rrulex.Describe(task.RRuleStr)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, err, "生成 Cron Job 调度规则描述失败")
	}
	executionTimes := make([]string, 0, len(runs))
	for _, run := range runs {
		executionTimes = append(executionTimes, carbonx.FormatDateTime(run))
	}

	return &trigger.PreviewCronJobScheduleRes{
		JobId:               task.ID,
		TaskCode:            task.TaskCode,
		ExecutionTimes:      executionTimes,
		ScheduleDescription: description,
		RruleStr:            task.RRuleStr,
	}, nil
}
