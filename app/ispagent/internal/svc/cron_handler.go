package svc

import (
	"context"
	"fmt"
	"time"

	ctask "zero-service/app/ispagent/internal/crontask"
	"zero-service/app/ispagent/internal/handler"
	"zero-service/app/ispagent/model/gormmodel"
	"zero-service/common/crontask"
	"zero-service/common/isp"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

func NewCronHandler(svcCtx *ServiceContext) crontask.Handler {
	return func(ctx context.Context, task *crontask.TaskConfig) error {
		fields := ctask.DeserializeExtra(string(task.Extra))
		if fields == nil {
			logx.WithContext(ctx).Errorf("[ispagent] cron task %s extra is nil", task.TaskCode)
			return nil
		}

		planStartTime, taskPatrolledID := taskExecution(ctx, task, fields)
		planStartTimeText := tool.CarbonFromTimeStartOfSecond(planStartTime).ToDateTimeString()

		logx.WithContext(ctx).Infof("[ispagent] 任务触发 task_code=%s patrol_id=%s plan=%s",
			task.TaskCode, taskPatrolledID, planStartTimeText)

		sendStatus := func(state string) {
			items := []isp.Item{{
				"task_patrolled_id":   taskPatrolledID,
				"task_name":           task.TaskName,
				"task_code":           task.TaskCode,
				"task_state":          state,
				"plan_start_time":     planStartTimeText,
				"start_time":          planStartTimeText,
				"task_progress":       "0",
				"task_estimated_time": "",
				"description":         "",
			}}
			if _, err := svcCtx.IspClient.Execute(ctx, isp.TypeTaskStatusData, isp.CommandReport,
				fields.SubstationCode, items); err != nil {
				logx.WithContext(ctx).Errorf("[ispagent] 上报任务状态 state=%s 失败: %v", state, err)
			}
		}

		upsertPatrolState := func(state string) {
			if err := handler.UpsertPatrolTask(ctx, svcCtx.DB, &gormmodel.GormIspPatrolTask{
				SendCode:        svcCtx.Config.IspSetting.SendCode,
				ReceiveCode:     svcCtx.IspClient.ReceiveCode(),
				Code:            fields.SubstationCode,
				TaskPatrolledID: taskPatrolledID,
				TaskName:        task.TaskName,
				TaskCode:        task.TaskCode,
				TaskState:       state,
				PlanStartTime:   planStartTime,
				StartTime:       planStartTime,
				TaskProgress:    "0",
			}); err != nil {
				logx.WithContext(ctx).Errorf("[ispagent] 同步巡视任务表失败: %v", err)
			}
		}

		// 开始执行
		upsertPatrolState(string(gormmodel.PatrolTaskStateRunning))
		sendStatus(string(gormmodel.PatrolTaskStateRunning))

		threading.GoSafe(func() {
			time.Sleep(60 * time.Second)

			upsertPatrolState(string(gormmodel.PatrolTaskStateFinished))
			sendStatus(string(gormmodel.PatrolTaskStateFinished))

			logx.WithContext(ctx).Infof("[ispagent] 任务完成 task_code=%s", task.TaskCode)
		})

		logx.WithContext(ctx).Infof("[ispagent] 任务开始执行 task_code=%s", task.TaskCode)
		return nil
	}
}

func taskExecution(ctx context.Context, task *crontask.TaskConfig, fields *ctask.IspTaskFields) (time.Time, string) {
	if taskPatrolledID, runAt, ok := ctask.ManualExecutionFromContext(ctx); ok {
		return runAt, taskPatrolledID
	}
	runAt := task.NextRun
	runTime := tool.CarbonFromTimeStartOfSecond(runAt)
	return runAt, fmt.Sprintf("%s_%s_%s", fields.SubstationCode, task.TaskCode, runTime.ToShortDateTimeString())
}
