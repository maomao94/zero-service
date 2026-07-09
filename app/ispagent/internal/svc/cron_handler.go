package svc

import (
	"context"
	"fmt"
	"time"

	ctask "zero-service/app/ispagent/internal/crontask"
	"zero-service/common/crontask"
	"zero-service/common/isp"

	"github.com/dromara/carbon/v2"
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

		planStartTime := task.NextRun.Format("2006-01-02 15:04:05")
		startTime := carbon.Now().ToDateTimeString()
		taskPatrolledID := fmt.Sprintf("%s_%s_%s",
			fields.SubstationCode, task.TaskCode, task.NextRun.Format("20060102150405"))

		logx.WithContext(ctx).Infof("[ispagent] cron 触发 task_code=%s patrol_id=%s plan=%s start=%s",
			task.TaskCode, taskPatrolledID, planStartTime, startTime)

		sendStatus := func(state int) {
			items := []isp.Item{{
				"task_patrolled_id":   taskPatrolledID,
				"task_name":           task.TaskName,
				"task_code":           task.TaskCode,
				"task_state":          fmt.Sprintf("%d", state),
				"plan_start_time":     planStartTime,
				"start_time":          startTime,
				"task_progress":       "0",
				"task_estimated_time": "",
				"description":         "",
			}}
			if _, err := svcCtx.IspClient.Execute(ctx, isp.TypeTaskStatusData, isp.CommandReport,
				fields.SubstationCode, items); err != nil {
				logx.WithContext(ctx).Errorf("[ispagent] 上报任务状态 state=%d 失败: %v", state, err)
			}
		}

		// 开始执行
		sendStatus(2)

		threading.GoSafe(func() {
			// 模拟执行延迟
			time.Sleep(60 * time.Second)

			// 执行完成
			sendStatus(1)

			logx.WithContext(ctx).Infof("[ispagent] cron 任务完成 task_code=%s", task.TaskCode)
		})

		logx.WithContext(ctx).Infof("[ispagent] cron 任务开始执行 task_code=%s", task.TaskCode)
		return nil
	}
}
