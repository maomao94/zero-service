package cron

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"
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
	log.Print("Starting cron service \n")

	go s.scanLoop()
}

func (s *CronService) Stop() {
	if s.cancelChan != nil {
		// Close the cancellation channel to signal goroutine termination
		close(s.cancelChan)
		s.cancelChan = nil
		log.Print("Stopping cron service \n")
	}
}

func (s *CronService) scanLoop() {
	for {
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
		logx.WithContext(ctx).Errorf("Error locking plan exec item: %v", err)
		return false
	}
	if execItem == nil {
		return false
	}

	queryExecItem, queryErr := s.svcCtx.PlanExecItemModel.FindOne(ctx, execItem.Id)
	if queryErr != nil {
		logx.WithContext(ctx).Errorf("Error querying plan exec item: %v", queryErr)
	}
	plan, queryErr := s.svcCtx.PlanModel.FindOneByPlanId(ctx, execItem.PlanId)
	if queryErr != nil {
		logx.WithContext(ctx).Errorf("Error querying plan: %v", queryErr)
	}

	logx.WithContext(ctx).Infof("Found plan exec item to trigger: id=%d, planPk=%d, planId=%s, planName=%s, batchPk=%d, batchId=%s, itemPk=%d, itemId=%s, itemName=%s, pointId=%s, nextTriggerTime=%v",
		queryExecItem.Id,
		queryExecItem.PlanPk,
		queryExecItem.PlanId,
		plan.PlanName.String,
		queryExecItem.BatchPk,
		queryExecItem.BatchId,
		queryExecItem.Id,
		queryExecItem.ItemId,
		queryExecItem.ItemName.String,
		queryExecItem.PointId.String,
		queryExecItem.NextTriggerTime,
	)

	ctx = logx.ContextWithFields(ctx, logx.Field("planId", queryExecItem.PlanId), logx.Field("planType", plan.Type.String),
		logx.Field("batchId", queryExecItem.BatchId),
		logx.Field("itemId", queryExecItem.ItemId),
		logx.Field("execId", queryExecItem.ExecId))
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
	logx.WithContext(ctx).Debugf("Executing callback for exec item %d, planId: %s, itemId: %s",
		execItem.Id, execItem.PlanId, execItem.ItemId)

	if err := s.svcCtx.PlanExecItemModel.UpdateStatusToRunning(ctx, execItem.Id, ""); err != nil {
		logx.WithContext(ctx).Errorf("Error updating plan exec item %d to running: %v", execItem.Id, err)
		return
	}
	callPlan := &streamevent.PbPlan{
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
			errCh <- fmt.Errorf("Error acquiring lock for plan exec item %d: %v", execItem.Id, taskErr)
			return
		}
		if !b {
			errCh <- fmt.Errorf("Error acquiring lock for plan exec item %d", execItem.Id)
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
		logx.WithContext(ctx).Errorf("gRPC call failed for exec item %d: %v", execItem.Id, err)
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
		logx.WithContext(ctx).Infof("gRPC call completed for exec item %d", execItem.Id)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToCompleted(ctx, execItem.Id, res.Message, res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); err != nil {
			logx.WithContext(ctx).Errorf("Error updating plan exec item %d to completed: %v", execItem.Id, err)
		}
	case model.ResultFailed:
		logx.WithContext(ctx).Infof("gRPC call returned failure for exec item %d: %s", execItem.Id, res.Message)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToFail(ctx, execItem.Id, model.ResultFailed, res.Message, res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); err != nil {
			logx.WithContext(ctx).Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, err)
		}
	case model.ResultDelayed:
		logx.WithContext(ctx).Infof("gRPC call returned delayed for exec item %d: %s", execItem.Id, res.Message)
		currentTime := carbon.Now()
		delayTriggerTime := currentTime.AddMinutes(5).ToDateTimeString()
		delayReason := res.Reason
		if res.DelayConfig == nil {
			logx.WithContext(ctx).Errorf("No delay config provided for exec item %d", execItem.Id)
		} else {
			if len(res.DelayConfig.DelayReason) != 0 {
				delayReason = fmt.Sprintf("%s, %s", res.DelayConfig.DelayReason, res.Message)
			}
			delayTime := carbon.ParseByLayout(res.DelayConfig.NextTriggerTime, carbon.DateTimeLayout)
			isTrue := true
			if delayTime.Error != nil || delayTime.IsInvalid() {
				logx.WithContext(ctx).Errorf("Invalid delay time format for exec item %d: %s", execItem.Id, res.DelayConfig.NextTriggerTime)
				isTrue = false
			} else {
				if delayTime.Lt(currentTime) {
					logx.WithContext(ctx).Errorf("Delay time for exec item %d is in the past: %v, current time: %v", execItem.Id, delayTime.ToDateTimeString(), currentTime.ToDateTimeString())
					isTrue = false
				}
			}
			if isTrue {
				delayTriggerTime = delayTime.ToDateTimeString()
			}
		}
		delayReason = fmt.Sprintf("%s, 下次触发时间: %s", delayReason, delayTriggerTime)
		logEntry.Reason = sql.NullString{String: delayReason, Valid: true}
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToDelayed(ctx, execItem.Id, res.ExecResult, res.Message, delayReason, delayTriggerTime,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); err != nil {
			logx.WithContext(ctx).Errorf("Error updating plan exec item %d to delayed: %v", execItem.Id, err)
		}
	case model.ResultOngoing:
		logx.WithContext(ctx).Infof("gRPC call returned ongoing for exec item %d: %s", execItem.Id, res.Message)
		currentTime := carbon.Now()
		delayTriggerTime := currentTime.AddMinutes(5).ToDateTimeString()
		delayReason := res.Reason
		if res.DelayConfig == nil {
			logx.WithContext(ctx).Debugf("No delay config provided for exec item %d", execItem.Id)
		} else {
			if len(res.DelayConfig.DelayReason) != 0 {
				delayReason = fmt.Sprintf("%s, %s", res.DelayConfig.DelayReason, res.Message)
			}
			delayTime := carbon.ParseByLayout(res.DelayConfig.NextTriggerTime, carbon.DateTimeLayout)
			isTrue := true
			if delayTime.Error != nil || delayTime.IsInvalid() {
				logx.WithContext(ctx).Errorf("Invalid delay time format for exec item %d: %s", execItem.Id, res.DelayConfig.NextTriggerTime)
				isTrue = false
			} else {
				if delayTime.Lt(currentTime) {
					logx.WithContext(ctx).Errorf("Delay time for exec item %d is in the past: %v, current time: %v", execItem.Id, delayTime.ToDateTimeString(), currentTime.ToDateTimeString())
					isTrue = false
				}
			}
			if isTrue {
				delayTriggerTime = delayTime.ToDateTimeString()
			}
		}
		delayReason = fmt.Sprintf("%s, 下次触发时间: %s", delayReason, delayTriggerTime)
		logEntry.Reason = sql.NullString{String: delayReason, Valid: true}
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToOngoing(ctx, execItem.Id, res.ExecResult, delayReason, true, delayTriggerTime,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); err != nil {
			logx.WithContext(ctx).Errorf("Error updating plan exec item %d to ongoing: %v", execItem.Id, err)
		}
	case model.ResultTerminated:
		logx.WithContext(ctx).Infof("gRPC call returned terminated for exec item %d: %s", execItem.Id, res.Message)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToTerminated(ctx, execItem.Id, res.Message, res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); err != nil {
			logx.WithContext(ctx).Errorf("Error updating plan exec item %d to terminated: %v", execItem.Id, err)
		}
	default:
		logx.WithContext(ctx).Errorf("Unknown execResult %s for exec item %d", res.ExecResult, execItem.Id)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToCompleted(ctx, execItem.Id, res.Message, res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); err != nil {
			logx.WithContext(ctx).Errorf("Error updating plan exec item %d to completed: %v", execItem.Id, err)
		}
	}
	// 插入执行日志
	if _, err := s.svcCtx.PlanExecLogModel.Insert(ctx, nil, logEntry); err != nil {
		logx.WithContext(ctx).Errorf("Error inserting plan exec log for item %d: %v", execItem.Id, err)
	}

	batchCount, err := s.svcCtx.PlanBatchModel.UpdateBatchFinishedTime(ctx, execItem.BatchPk)
	if err != nil {
		logx.WithContext(ctx).Errorf("Error updating batch %s completed time: %v", execItem.BatchId, err)
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
		logx.WithContext(ctx).Errorf("Error updating plan %s completed time: %v", execItem.PlanId, err)
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
	logx.WithContext(ctx).Infof("Successfully executed callback for plan exec item: id=%d", execItem.Id)
}
