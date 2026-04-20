package cron

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"
	"zero-service/app/trigger/internal/execdelay"
	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"
	"zero-service/model"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type CronService struct {
	cancelChan chan struct{}
	svcCtx     *svc.ServiceContext
	taskRunner *threading.TaskRunner
}

func NewCronService(svcCtx *svc.ServiceContext) *CronService {
	return &CronService{
		svcCtx:     svcCtx,
		taskRunner: threading.NewTaskRunner(16),
	}
}

func (s *CronService) Start() {
	if s.cancelChan != nil {
		return
	}
	// Create cancellation channel for proper goroutine termination
	s.cancelChan = make(chan struct{})
	logx.Info("cron service started")

	go s.scanLoop()
}

func (s *CronService) Stop() {
	if s.cancelChan != nil {
		// Close the cancellation channel to signal goroutine termination
		close(s.cancelChan)
		s.cancelChan = nil
		logx.Info("cron service stopped")
	}
}

func (s *CronService) scanLoop() {
	for {
		itemsProcessed := s.ScanPlanExecItem()
		var sleepDuration time.Duration
		if itemsProcessed {
			sleepDuration = 10 * time.Millisecond
		} else {
			sleepDuration = time.Duration(1000+rand.Intn(1000)) * time.Millisecond // 1~2秒随机
		}
		timer := time.NewTimer(sleepDuration)
		select {
		case <-s.cancelChan:
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}

func (s *CronService) ScanPlanExecItem() bool {
	ctx := context.Background()
	tracer := otel.Tracer(trace.TraceName)
	ctx, span := tracer.Start(ctx, "cron-scan", oteltrace.WithSpanKind(oteltrace.SpanKindProducer))
	defer span.End()

	execItem, err := s.svcCtx.PlanExecItemModel.LockTriggerItem(ctx, 5*60*time.Second)
	if err != nil {
		if err == sqlx.ErrNotFound {
			return false
		}
		logx.WithContext(ctx).Errorf("%s LockTriggerItem 失败: %v", planscope.CronLockScope(), err)
		return false
	}
	if execItem == nil {
		return false
	}

	queryExecItem, queryErr := s.svcCtx.PlanExecItemModel.FindOne(ctx, execItem.Id)
	if queryErr != nil || queryExecItem == nil {
		logx.WithContext(ctx).Errorf("%s 锁定后重载 plan_exec_item 失败: %v", planscope.ExecCron(execItem), queryErr)
		return false
	}
	plan, planErr := s.svcCtx.PlanModel.FindOneByPlanId(ctx, execItem.PlanId)
	if planErr != nil || plan == nil {
		logx.WithContext(ctx).Errorf("%s 锁定后加载 plan 失败: %v", planscope.ExecCron(execItem), planErr)
		return false
	}

	scope := planscope.TriggerScope(queryExecItem, plan)
	logx.WithContext(ctx).Infof("%s 扫表命中待触发项 next_trigger=%s 即将调用 streamevent.HandlerPlanTaskEvent",
		scope, queryExecItem.NextTriggerTime.Format(time.RFC3339Nano))

	// 更新扫表标记为已扫表
	// 更新 plan 表的扫表标记
	planUpdateBuilder := s.svcCtx.PlanModel.UpdateBuilder().Set("scan_flg", 1).Where("id = ?", plan.Id)
	if _, err := s.svcCtx.PlanModel.UpdateWithBuilder(ctx, nil, planUpdateBuilder); err != nil {
		logx.WithContext(ctx).Errorf("%s 更新 plan.scan_flg=1 失败: %v", scope, err)
	}

	// 更新 plan_batch 表的扫表标记
	batchUpdateBuilder := s.svcCtx.PlanBatchModel.UpdateBuilder().Set("scan_flg", 1).Where("id = ?", queryExecItem.BatchPk)
	if _, err := s.svcCtx.PlanBatchModel.UpdateWithBuilder(ctx, nil, batchUpdateBuilder); err != nil {
		logx.WithContext(ctx).Errorf("%s 更新 plan_batch.scan_flg=1 失败: %v", scope, err)
	}

	s.ExecuteCallback(ctx, queryExecItem, plan)
	return true
}

type rawCodec struct{}

func (cb rawCodec) Marshal(v any) ([]byte, error) {
	return tool.ToProtoBytes(v)
}

func (cb rawCodec) Unmarshal(data []byte, v any) error {
	ba, ok := v.(*[]byte)
	if !ok {
		return fmt.Errorf("please pass in *[]byte")
	}
	*ba = append(*ba, data...)
	return nil
}

func (cb rawCodec) Name() string { return "proto_raw" }

func (s *CronService) ExecuteCallback(ctx context.Context, execItem *model.PlanExecItem, plan *model.Plan) {
	traceID := trace.TraceIDFromContext(ctx)
	if execItem.RequestTimeout == 0 {
		execItem.RequestTimeout = s.svcCtx.Config.StreamEventConf.Timeout
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(execItem.RequestTimeout)*time.Millisecond+120*time.Second)
	defer cancel()
	scope := planscope.TriggerScope(execItem, plan)
	logx.WithContext(ctx).Debugf("%s streamevent.HandlerPlanTaskEvent 开始", scope)

	if err := s.svcCtx.PlanExecItemModel.UpdateStatusToRunning(ctx, execItem.Id, ""); err != nil {
		logx.WithContext(ctx).Errorf("%s 更新 plan_exec_item 为 running 失败: %v", scope, err)
		return
	}
	callPlan := &streamevent.PlanPb{
		CreateTime:  carbon.CreateFromStdTime(plan.CreateTime).ToDateTimeString(),
		UpdateTime:  carbon.CreateFromStdTime(plan.UpdateTime).ToDateTimeString(),
		CreateUser:  plan.CreateUser.String,
		UpdateUser:  plan.UpdateUser.String,
		DeptCode:    plan.DeptCode.String,
		Id:          plan.Id,
		PlanId:      plan.PlanId,
		PlanName:    plan.PlanName.String,
		Type:        plan.Type.String,
		GroupId:     plan.GroupId.String,
		Description: plan.Description.String,
		StartTime:   carbon.CreateFromStdTime(plan.StartTime).ToDateTimeString(),
		EndTime:     carbon.CreateFromStdTime(plan.EndTime).ToDateTimeString(),
		Ext1:        plan.Ext1.String,
		Ext2:        plan.Ext2.String,
		Ext3:        plan.Ext3.String,
		Ext4:        plan.Ext4.String,
		Ext5:        plan.Ext5.String,
	}
	req := &streamevent.HandlerPlanTaskEventReq{
		Plan:            callPlan,
		Id:              execItem.Id,
		PlanPk:          execItem.PlanPk,
		PlanId:          execItem.PlanId,
		BatchPk:         execItem.BatchPk,
		BatchId:         execItem.BatchId,
		ExecId:          execItem.ExecId,
		ItemId:          execItem.ItemId,
		ItemType:        execItem.ItemType.String,
		ItemName:        execItem.ItemName.String,
		ItemRowId:       execItem.ItemRowId,
		PointId:         execItem.PointId.String,
		Payload:         execItem.Payload,
		PlanTriggerTime: carbon.NewCarbon(execItem.PlanTriggerTime).ToDateTimeString(),
		LastResult:      execItem.LastResult.String,
		LastMessage:     execItem.LastMessage.String,
		LastReason:      execItem.LastReason.String,
	}
	if execItem.LastTriggerTime.Valid {
		req.LastTriggerTime = carbon.NewCarbon(execItem.LastTriggerTime.Time).ToDateTimeString()
	}
	errCh := make(chan error, 1)
	var err error
	var res *streamevent.HandlerPlanTaskEventRes
	s.taskRunner.Schedule(func() {
		defer close(errCh)
		lockKey := fmt.Sprintf("trigger:lock:plan:exec:%s", execItem.ExecId)
		lock := redis.NewRedisLock(s.svcCtx.Redis, lockKey)
		b, taskErr := lock.AcquireCtx(ctx)
		if taskErr != nil {
			errCh <- fmt.Errorf("%s 回调前 Redis 锁 Acquire 错误: %v", planscope.TriggerScope(execItem, plan), taskErr)
			return
		}
		if !b {
			errCh <- fmt.Errorf("%s 回调前未抢到 Redis 锁(资源忙)", planscope.TriggerScope(execItem, plan))
			return
		}
		defer func() {
			lock.ReleaseCtx(ctx)
		}()
		res, taskErr = s.svcCtx.StreamEventCli.HandlerPlanTaskEvent(ctx, req)
		errCh <- taskErr
	})
	err = <-errCh
	if err != nil {
		// 记录执行日志
		//logEntry := &model.PlanExecLog{
		//	DeptCode:    execItem.DeptCode,
		//	PlanPk:      plan.Id,
		//	PlanId:      execItem.PlanId,
		//	PlanName:    plan.PlanName,
		//	BatchPk:     execItem.BatchPk,
		//	BatchId:     execItem.BatchId,
		//	ItemPk:      execItem.Id,
		//	ExecId:      execItem.ExecId,
		//	ItemId:      execItem.ItemId,
		//	ItemType:    execItem.ItemType,
		//	ItemName:    execItem.ItemName,
		//	PointId:     execItem.PointId,
		//	TriggerTime: time.Now(),
		//	TraceId:     sql.NullString{String: traceID, Valid: traceID != ""},
		//	ExecResult:  sql.NullString{String: model.ResultFailed, Valid: true}, // 失败
		//	Message:     sql.NullString{String: "gRPC call failed: " + err.Error(), Valid: true},
		//}
		logx.WithContext(ctx).Errorf("%s streamevent.HandlerPlanTaskEvent 调用失败: %v", scope, err)
		//if len(lastResullt.String) == 0 || lastResullt.String != model.ResultOngoing {
		//	if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToFail(ctx, execItem.Id, model.ResultFailed, "gRPC call failed: "+err.Error(), "",
		//		[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		//	); updateErr != nil {
		//		logx.WithContext(ctx).Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
		//	}
		//} else {
		//	logEntry.ExecResult = sql.NullString{String: model.ResultOngoing, Valid: true}
		//}
		//if _, err := s.svcCtx.PlanExecLogModel.Insert(ctx, nil, logEntry); err != nil {
		//	logx.WithContext(ctx).Errorf("Error inserting plan exec log for item %d: %v", execItem.Id, err)
		//}
		return
	}
	// 记录执行日志
	logEntry := &model.PlanExecLog{
		DeptCode:    execItem.DeptCode,
		PlanPk:      plan.Id,
		PlanId:      execItem.PlanId,
		PlanName:    plan.PlanName,
		BatchPk:     execItem.BatchPk,
		BatchId:     execItem.BatchId,
		ItemPk:      execItem.Id,
		ExecId:      execItem.ExecId,
		ItemId:      execItem.ItemId,
		ItemType:    execItem.ItemType,
		ItemName:    execItem.ItemName,
		PointId:     execItem.PointId,
		TriggerTime: time.Now(),
		TraceId:     sql.NullString{String: traceID, Valid: traceID != ""},
		ExecResult:  sql.NullString{String: res.ExecResult, Valid: res.ExecResult != ""},
		Message:     sql.NullString{String: res.Message, Valid: res.Message != ""},
		Reason:      sql.NullString{String: res.Reason, Valid: res.Reason != ""},
	}

	switch res.ExecResult {
	case model.ResultCompleted:
		logx.WithContext(ctx).Infof("%s streamevent 返回 execResult=%s message=%s reason=%s", scope, model.ResultCompleted, res.Message, res.Reason)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToCompleted(ctx, execItem.Id, res.Message, res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); err != nil {
			logx.WithContext(ctx).Errorf("%s 更新 plan_exec_item 为 completed 失败: %v", scope, err)
		}
	case model.ResultFailed:
		logx.WithContext(ctx).Infof("%s streamevent 返回 execResult=%s message=%s reason=%s", scope, model.ResultFailed, res.Message, res.Reason)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToFail(ctx, execItem.Id, model.ResultFailed, res.Message, res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); err != nil {
			logx.WithContext(ctx).Errorf("%s 更新 plan_exec_item 为 failed 失败: %v", scope, err)
		}
	case model.ResultDelayed:
		logx.WithContext(ctx).Infof("%s streamevent 返回 execResult=%s message=%s reason=%s", scope, model.ResultDelayed, res.Message, res.Reason)
		currentTime := carbon.Now()
		dr := execdelay.Resolve(res.DelayConfig, res.Message, res.Reason, currentTime, execdelay.ModeDelayed)
		execdelay.LogWarnings(ctx, scope, dr)
		delayTriggerTime := dr.NextTrigger
		delayReason := execdelay.FinalReason(dr.ReasonStem, delayTriggerTime)
		logEntry.Reason = sql.NullString{String: delayReason, Valid: true}
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToDelayed(ctx, execItem.Id, res.ExecResult, res.Message, delayReason, delayTriggerTime,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); err != nil {
			logx.WithContext(ctx).Errorf("%s 更新 plan_exec_item 为 delayed 失败: %v", scope, err)
		}
	case model.ResultOngoing:
		logx.WithContext(ctx).Infof("%s streamevent 返回 execResult=%s message=%s reason=%s", scope, model.ResultOngoing, res.Message, res.Reason)
		currentTime := carbon.Now()
		or := execdelay.Resolve(res.DelayConfig, res.Message, res.Reason, currentTime, execdelay.ModeOngoing)
		execdelay.LogWarnings(ctx, scope, or)
		delayTriggerTime := or.NextTrigger
		delayReason := execdelay.FinalReason(or.ReasonStem, delayTriggerTime)
		logEntry.Reason = sql.NullString{String: delayReason, Valid: true}
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToOngoing(ctx, execItem.Id, res.ExecResult, delayReason, true, delayTriggerTime,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); err != nil {
			logx.WithContext(ctx).Errorf("%s 更新 plan_exec_item 为 ongoing 失败: %v", scope, err)
		}
	case model.ResultTerminated:
		logx.WithContext(ctx).Infof("%s streamevent 返回 execResult=%s message=%s reason=%s", scope, model.ResultTerminated, res.Message, res.Reason)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToTerminated(ctx, execItem.Id, res.Message, res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); err != nil {
			logx.WithContext(ctx).Errorf("%s 更新 plan_exec_item 为 terminated 失败: %v", scope, err)
		}
	default:
		logx.WithContext(ctx).Errorf("%s streamevent 返回未知 execResult=%q，将按 completed 回写库", scope, res.ExecResult)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToCompleted(ctx, execItem.Id, res.Message, res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); err != nil {
			logx.WithContext(ctx).Errorf("%s 未知 execResult 回写 completed 失败: %v", scope, err)
		}
	}
	// 插入执行日志
	if _, err := s.svcCtx.PlanExecLogModel.Insert(ctx, nil, logEntry); err != nil {
		logx.WithContext(ctx).Errorf("%s 插入 plan_exec_log 失败: %v", scope, err)
	}

	batchCount, err := s.svcCtx.PlanBatchModel.UpdateBatchFinishedTime(ctx, execItem.BatchPk)
	if err != nil {
		logx.WithContext(ctx).Errorf("%s 更新 plan_batch.finished_time 失败: %v", scope, err)
	}
	if batchCount > 0 {
		batchNotifyReq := streamevent.NotifyPlanEventReq{
			EventType:  streamevent.PlanEventType_BATCH_FINISHED,
			PlanId:     execItem.PlanId,
			PlanType:   plan.Type.String,
			BatchId:    execItem.BatchId,
			Attributes: map[string]string{},
		}
		s.svcCtx.StreamEventCli.NotifyPlanEvent(ctx, &batchNotifyReq)
	}
	planCount, err := s.svcCtx.PlanModel.UpdateBatchFinishedTime(ctx, execItem.PlanPk)
	if err != nil {
		logx.WithContext(ctx).Errorf("%s 更新 plan.finished_time 失败: %v", scope, err)
	}
	if planCount > 0 {
		planPlanReq := streamevent.NotifyPlanEventReq{
			EventType: streamevent.PlanEventType_PLAN_FINISHED,
			PlanId:    execItem.PlanId,
			PlanType:  plan.Type.String,
			//BatchId:    execItem.BatchId,
			Attributes: map[string]string{},
		}
		s.svcCtx.StreamEventCli.NotifyPlanEvent(ctx, &planPlanReq)
	}
	logx.WithContext(ctx).Infof("%s streamevent.HandlerPlanTaskEvent 结束 batch_notify_rows=%d plan_notify_rows=%d", scope, batchCount, planCount)
}
