package planscope

import (
	"context"
	"fmt"

	"zero-service/app/trigger/model/gormmodel"

	"github.com/zeromicro/go-zero/core/logx"
)

// execRef 串联计划业务 ID、批次 ID、执行单 ID，便于日志检索与对齐「计划 → 批次 → 执行项」关系。
func execRef(exec *gormmodel.PlanExecItem) string {
	if exec == nil {
		return ""
	}
	return fmt.Sprintf("%s/%s/%s", exec.PlanId, exec.BatchId, exec.ExecId)
}

// batchRef 串联计划业务 ID 与批次 ID（无执行项维度时使用）。
func batchRef(planID string, batch *gormmodel.PlanBatch) string {
	if batch == nil || planID == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", planID, batch.BatchId)
}

const (
	EntryRPC      = "rpc"
	EntryCron     = "cron"
	EntryCallback = "callback"
)

// NotifyEvent* 与日志字段 notify_event 对应，语义对齐 streamevent.PlanEventType（BATCH_FINISHED / PLAN_FINISHED）。
const (
	NotifyEventBatchFinished = "BATCH_FINISHED"
	NotifyEventPlanFinished  = "PLAN_FINISHED"
)

type Scope struct {
	Entry  string
	Tag    string
	Fields []logx.LogField
}

func (s Scope) Logger(ctx context.Context) logx.Logger {
	return logx.WithContext(ctx).WithFields(s.Fields...)
}

func (s Scope) WithFields(extra ...logx.LogField) Scope {
	s.Fields = append(s.Fields, extra...)
	return s
}

func planFields(entry, tag string, plan *gormmodel.Plan) Scope {
	fields := []logx.LogField{
		logx.Field("entry", entry),
		logx.Field("tag", tag),
	}
	if plan != nil {
		fields = append(fields,
			logx.Field("plan_id", plan.PlanId),
			logx.Field("plan_pk", plan.Id),
			logx.Field("plan_name", plan.PlanName.String),
			logx.Field("ref", plan.PlanId),
		)
	}
	return Scope{Entry: entry, Tag: tag, Fields: fields}
}

func execFields(entry, tag string, exec *gormmodel.PlanExecItem) Scope {
	fields := []logx.LogField{
		logx.Field("entry", entry),
		logx.Field("tag", tag),
	}
	if exec != nil {
		fields = append(fields,
			logx.Field("plan_id", exec.PlanId),
			logx.Field("plan_pk", exec.PlanPk),
			logx.Field("batch_id", exec.BatchId),
			logx.Field("batch_pk", exec.BatchPk),
			logx.Field("item_pk", exec.Id),
			logx.Field("item_id", exec.ItemId),
			logx.Field("exec_id", exec.ExecId),
			logx.Field("ref", execRef(exec)),
		)
	}
	return Scope{Entry: entry, Tag: tag, Fields: fields}
}

func PlanScope(plan *gormmodel.Plan) Scope {
	return planFields(EntryRPC, "plan", plan)
}

func ExecScope(exec *gormmodel.PlanExecItem) Scope {
	return execFields(EntryRPC, "plan-exec", exec)
}

func ExecCron(exec *gormmodel.PlanExecItem) Scope {
	return execFields(EntryCron, "plan-exec", exec)
}

func ExecCallback(exec *gormmodel.PlanExecItem) Scope {
	return execFields(EntryCallback, "plan-exec", exec)
}

func CronLockScope() Scope {
	return Scope{
		Entry: EntryCron,
		Tag:   "cron-lock",
		Fields: []logx.LogField{
			logx.Field("entry", EntryCron),
			logx.Field("tag", "cron-lock"),
		},
	}
}

func TriggerScope(exec *gormmodel.PlanExecItem, plan *gormmodel.Plan) Scope {
	fields := []logx.LogField{
		logx.Field("entry", EntryCron),
		logx.Field("tag", "plan-trigger"),
	}
	if exec != nil {
		fields = append(fields,
			logx.Field("plan_id", exec.PlanId),
			logx.Field("plan_pk", exec.PlanPk),
			logx.Field("batch_id", exec.BatchId),
			logx.Field("batch_pk", exec.BatchPk),
			logx.Field("item_pk", exec.Id),
			logx.Field("item_id", exec.ItemId),
			logx.Field("exec_id", exec.ExecId),
			logx.Field("ref", execRef(exec)),
		)
	}
	if plan != nil {
		fields = append(fields, logx.Field("plan_name", plan.PlanName.String))
	}
	return Scope{Entry: EntryCron, Tag: "plan-trigger", Fields: fields}
}

func BatchScope(plan *gormmodel.Plan, batch *gormmodel.PlanBatch) Scope {
	fields := []logx.LogField{
		logx.Field("entry", EntryRPC),
		logx.Field("tag", "plan-batch"),
	}
	if batch != nil {
		var planPk string
		var planID string
		if plan != nil {
			planPk = plan.Id
			planID = plan.PlanId
		} else {
			planPk = batch.PlanPk
			planID = batch.PlanId
		}
		fields = append(fields,
			logx.Field("plan_id", planID),
			logx.Field("plan_pk", planPk),
			logx.Field("batch_id", batch.BatchId),
			logx.Field("batch_pk", batch.Id),
			logx.Field("batch_num", batch.BatchNum.String),
			logx.Field("ref", batchRef(planID, batch)),
		)
	}
	if plan != nil {
		fields = append(fields, logx.Field("plan_name", plan.PlanName.String))
	}
	return Scope{Entry: EntryRPC, Tag: "plan-batch", Fields: fields}
}

func CallbackScope(exec *gormmodel.PlanExecItem, plan *gormmodel.Plan, batch *gormmodel.PlanBatch) Scope {
	fields := []logx.LogField{
		logx.Field("entry", EntryCallback),
		logx.Field("tag", "plan-callback"),
	}
	if exec != nil {
		fields = append(fields,
			logx.Field("plan_id", exec.PlanId),
			logx.Field("plan_pk", exec.PlanPk),
			logx.Field("batch_id", exec.BatchId),
			logx.Field("batch_pk", exec.BatchPk),
			logx.Field("item_pk", exec.Id),
			logx.Field("item_id", exec.ItemId),
			logx.Field("exec_id", exec.ExecId),
			logx.Field("ref", execRef(exec)),
		)
	}
	if plan != nil {
		fields = append(fields, logx.Field("plan_name", plan.PlanName.String))
	}
	if batch != nil {
		fields = append(fields, logx.Field("batch_num", batch.BatchNum.String))
	}
	return Scope{Entry: EntryCallback, Tag: "plan-callback", Fields: fields}
}
