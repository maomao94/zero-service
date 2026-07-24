package cronjob

import (
	"context"
	"errors"
	"fmt"
	"time"

	"zero-service/common/crontask"
	"zero-service/facade/streamevent/streamevent"

	"google.golang.org/grpc"
)

// EventClient 是 Cron Job 到点回调所需的最小 Eventstream 客户端接口。
type EventClient interface {
	HandleCronJobEvent(ctx context.Context, in *streamevent.HandleCronJobEventReq, opts ...grpc.CallOption) (*streamevent.HandleCronJobEventRes, error)
}

// NewEventHandler 创建把 Cron Job 到点事件转发到 Eventstream 的调度 Handler。
func NewEventHandler(client EventClient) crontask.Handler {
	return func(ctx context.Context, task *crontask.TaskConfig) error {
		if client == nil {
			return errors.New("Eventstream 客户端不能为空")
		}
		extra, err := ParseExtra(task.Extra)
		if err != nil {
			return err
		}
		response, err := client.HandleCronJobEvent(ctx, &streamevent.HandleCronJobEventReq{
			JobId:         task.ID,
			TaskCode:      task.TaskCode,
			TaskName:      task.TaskName,
			Priority:      int32(task.Priority),
			Payload:       string(task.Payload),
			Extra:         string(task.Extra),
			ScheduledTime: formatTime(task.NextRun),
			Type:          extra.Type,
			GroupId:       extra.GroupId,
			Description:   extra.Description,
			Ext1:          extra.Ext1,
			Ext2:          extra.Ext2,
			Ext3:          extra.Ext3,
			Ext4:          extra.Ext4,
			Ext5:          extra.Ext5,
			DeptCode:      extra.DeptCode,
		})
		if err != nil {
			return fmt.Errorf("调用 Eventstream Cron Job 回调失败: %w", err)
		}
		if response == nil {
			return errors.New("Eventstream Cron Job 回调返回为空")
		}
		switch response.Receipt {
		case streamevent.CronJobReceiptPb_CRON_JOB_RECEIPT_SUCCESS:
			return nil
		case streamevent.CronJobReceiptPb_CRON_JOB_RECEIPT_TASK_NOT_FOUND:
			return fmt.Errorf("%w: %s", crontask.ErrDeleteTask, response.Message)
		default:
			return fmt.Errorf("Eventstream Cron Job 回执未知: receipt=%s message=%s", response.Receipt.String(), response.Message)
		}
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(dateTimeLayout)
}
