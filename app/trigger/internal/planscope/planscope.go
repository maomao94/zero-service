// Package planscope builds grep-friendly scope lines for plan / plan_batch / plan_exec_item logs.
//
// Conventions (plain-text grep / log platform "message" column):
//   - Business logs tied to these tables must include a Scope line (entry= and dimension tags).
//   - By entry: entry=rpc (RPC writes), entry=cron (scan + streamevent), entry=callback (CallbackPlanExecItem).
//   - By table keys in the message: plan_pk/plan_id→plan; batch_pk/batch_id→plan_batch; item_pk/exec_id/item_id→plan_exec_item.
package planscope

import (
	"fmt"

	"zero-service/model"
)

// Entry tags for log message lines (appear near the start of the scope string).
const (
	EntryRPC      = "entry=rpc"
	EntryCron     = "entry=cron"
	EntryCallback = "entry=callback"
)

func planNameQuoted(plan *model.Plan) string {
	if plan == nil {
		return ""
	}
	return plan.PlanName.String
}

func batchNumStr(batch *model.PlanBatch) string {
	if batch == nil {
		return ""
	}
	return batch.BatchNum.String
}

// PlanScope is the log message prefix for plan-only RPCs (terminate / pause / resume plan, etc.).
func PlanScope(plan *model.Plan) string {
	if plan == nil {
		return EntryRPC + " [plan]"
	}
	return fmt.Sprintf("%s [plan] plan_id=%s plan_pk=%d plan_name=%q",
		EntryRPC, plan.PlanId, plan.Id, planNameQuoted(plan))
}

func execScopeLine(entry string, exec *model.PlanExecItem) string {
	if exec == nil {
		return fmt.Sprintf("%s [plan-exec]", entry)
	}
	return fmt.Sprintf("%s [plan-exec] plan_id=%s plan_pk=%d | batch_id=%s batch_pk=%d | item_pk=%d item_id=%s exec_id=%s",
		entry,
		exec.PlanId, exec.PlanPk, exec.BatchId, exec.BatchPk, exec.Id, exec.ItemId, exec.ExecId)
}

// ExecScope is the message prefix when only an exec item is known (RPC).
func ExecScope(exec *model.PlanExecItem) string {
	return execScopeLine(EntryRPC, exec)
}

// ExecCron is the message prefix for cron paths with only an exec item (lock/reload errors, etc.).
func ExecCron(exec *model.PlanExecItem) string {
	return execScopeLine(EntryCron, exec)
}

// ExecCallback is the message prefix for CallbackPlanExecItem before plan/batch are loaded.
func ExecCallback(exec *model.PlanExecItem) string {
	return execScopeLine(EntryCallback, exec)
}

// CronLockScope is the message prefix when LockTriggerItem fails before any exec item context exists.
func CronLockScope() string {
	return EntryCron + " [cron-lock]"
}

// TriggerScope is the message prefix for scan / streamevent HandlerPlanTaskEvent paths.
func TriggerScope(exec *model.PlanExecItem, plan *model.Plan) string {
	if exec == nil {
		return EntryCron + " [plan-trigger]"
	}
	return fmt.Sprintf("%s [plan-trigger] plan_id=%s plan_pk=%d plan_name=%q | batch_id=%s batch_pk=%d | item_pk=%d item_id=%s exec_id=%s",
		EntryCron,
		exec.PlanId, exec.PlanPk, planNameQuoted(plan), exec.BatchId, exec.BatchPk, exec.Id, exec.ItemId, exec.ExecId)
}

// BatchScope is the message prefix when plan and batch are known but no exec item (e.g. batch terminate).
func BatchScope(plan *model.Plan, batch *model.PlanBatch) string {
	if batch == nil {
		return EntryRPC + " [plan-batch]"
	}
	var planPk int64
	var planID string
	if plan != nil {
		planPk = plan.Id
		planID = plan.PlanId
	} else {
		planPk = batch.PlanPk
		planID = batch.PlanId
	}
	return fmt.Sprintf("%s [plan-batch] plan_id=%s plan_pk=%d plan_name=%q | batch_id=%s batch_pk=%d batch_num=%s",
		EntryRPC,
		planID, planPk, planNameQuoted(plan), batch.BatchId, batch.Id, batchNumStr(batch))
}

// CallbackScope is the message prefix for CallbackPlanExecItem RPC (includes batch_num).
func CallbackScope(exec *model.PlanExecItem, plan *model.Plan, batch *model.PlanBatch) string {
	if exec == nil {
		return EntryCallback + " [plan-callback]"
	}
	return fmt.Sprintf("%s [plan-callback] plan_id=%s plan_pk=%d plan_name=%q | batch_id=%s batch_pk=%d batch_num=%s | item_pk=%d item_id=%s exec_id=%s",
		EntryCallback,
		exec.PlanId, exec.PlanPk, planNameQuoted(plan), exec.BatchId, exec.BatchPk, batchNumStr(batch), exec.Id, exec.ItemId, exec.ExecId)
}
