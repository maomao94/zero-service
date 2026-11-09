// Package confirm 负责计划任务二次确认的通知、锁与生命周期处理。
package confirm

import (
	"context"
	"strings"

	"zero-service/common/tool"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// EventBatchConfirmRequired 批次待确认事件名，前端按此事件注册监听。
	EventBatchConfirmRequired = "plan:batch-confirm-required"
	// DefaultRoomPrefix 机构级房间前缀，最终房间为 dept:{deptCode}。
	DefaultRoomPrefix = "dept:"
	// DefaultFallbackRoom 批次未携带机构时的兜底房间。
	DefaultFallbackRoom = "plan-confirm"
)

// BatchConfirmPayload 是推送给前端的批次待确认通知负载，字段风格与 live 通知保持一致。
type BatchConfirmPayload struct {
	PlanPk          string `json:"planPk"`
	PlanId          string `json:"planId"`
	PlanName        string `json:"planName"`
	BatchPk         string `json:"batchPk"`
	BatchId         string `json:"batchId"`
	BatchNum        string `json:"batchNum"`
	DeptCode        string `json:"deptCode"`
	PlanTriggerTime string `json:"planTriggerTime"`
	ConfirmDeadline string `json:"confirmDeadline"`
	LeadMinutes     int    `json:"leadMinutes"`
	ItemCount       int64  `json:"itemCount"`
	NotifiedAt      string `json:"notifiedAt"`
}

// Notifier 通过 socketpush 向 Web 推送批次确认通知。
// SocketPushCli 未配置时退化为只记录日志，不影响确认流程。
type Notifier struct {
	cli          socketpush.SocketPushClient
	roomPrefix   string
	fallbackRoom string
	event        string
}

func NewNotifier(cli socketpush.SocketPushClient, roomPrefix, fallbackRoom, event string) *Notifier {
	roomPrefix = strings.TrimSpace(roomPrefix)
	if roomPrefix == "" {
		roomPrefix = DefaultRoomPrefix
	}
	fallbackRoom = strings.TrimSpace(fallbackRoom)
	if fallbackRoom == "" {
		fallbackRoom = DefaultFallbackRoom
	}
	if event == "" {
		event = EventBatchConfirmRequired
	}
	return &Notifier{cli: cli, roomPrefix: roomPrefix, fallbackRoom: fallbackRoom, event: event}
}

// Room 返回机构级房间名；机构为空时使用兜底房间。
func (n *Notifier) Room(deptCode string) string {
	deptCode = strings.TrimSpace(deptCode)
	if deptCode == "" {
		return n.fallbackRoom
	}
	return n.roomPrefix + deptCode
}

// NotifyBatchConfirmRequired 推送一条批次待确认通知到机构级房间。
func (n *Notifier) NotifyBatchConfirmRequired(ctx context.Context, payload BatchConfirmPayload) error {
	if n == nil || n.cli == nil {
		logx.WithContext(ctx).Infof("批次确认通知未配置 socketpush，跳过推送 batch_pk=%s plan_id=%s", payload.BatchPk, payload.PlanId)
		return nil
	}
	data, err := jsonx.Marshal(payload)
	if err != nil {
		return err
	}
	reqId, err := tool.SimpleUUID()
	if err != nil {
		return err
	}
	_, err = n.cli.BroadcastRoom(ctx, &socketpush.BroadcastRoomReq{
		ReqId:   reqId,
		Room:    n.Room(payload.DeptCode),
		Event:   n.event,
		Payload: string(data),
	})
	return err
}
