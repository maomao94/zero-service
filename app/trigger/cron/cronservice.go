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
	execItem, err := s.svcCtx.PlanExecItemModel.LockTriggerItem(ctx, 5*60*time.Second)
	if err != nil {
		if err == sqlx.ErrNotFound {
			return false
		}
		logx.Errorf("Error locking plan exec item: %v", err)
		return false
	}
	if execItem == nil {
		return false
	}
	queryExecItem, queryErr := s.svcCtx.PlanExecItemModel.FindOne(ctx, execItem.Id)
	if queryErr != nil {
		logx.Errorf("Error querying plan exec item: %v", queryErr)
	}
	plan, queryErr := s.svcCtx.PlanModel.FindOneByPlanId(ctx, execItem.PlanId)
	if queryErr != nil {
		logx.Errorf("Error querying plan: %v", queryErr)
	}
	logx.Infof("Found plan exec item to trigger: id=%d, planPk=%d, planId=%s, planName=%s, itemId=%s, itemName=%s, pointId=%s, nextTriggerTime=%v",
		execItem.Id,
		execItem.PlanPk,
		execItem.PlanId,
		plan.PlanName.String,
		execItem.ItemId,
		execItem.ItemName.String,
		execItem.PointId.String,
		execItem.NextTriggerTime,
	)
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
	if execItem.RequestTimeout == 0 {
		execItem.RequestTimeout = s.svcCtx.Config.StreamEventConf.Timeout
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(execItem.RequestTimeout)*time.Millisecond+120*time.Second)
	defer cancel()
	logx.Debugf("Executing callback for exec item %d, planId: %s, itemId: %s",
		execItem.Id, execItem.PlanId, execItem.ItemId)

	if err := s.svcCtx.PlanExecItemModel.UpdateStatusToRunning(ctx, execItem.Id); err != nil {
		logx.Errorf("Error updating plan exec item %d to running: %v", execItem.Id, err)
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
		PointId:         execItem.PointId.String,
		Payload:         execItem.Payload,
		PlanTriggerTime: carbon.NewCarbon(execItem.PlanTriggerTime).ToDateTimeString(),
		LastTriggerTime: "",
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
		lockKey := fmt.Sprintf("trigger:lock:plan:exec:%d", execItem.ExecId)
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
		logx.Errorf("gRPC call failed for exec item %d: %v", execItem.Id, err)
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToFail(ctx, execItem.Id, model.ResultFailed, "gRPC call failed: "+err.Error(), "",
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
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
			TraceId:     sql.NullString{String: "", Valid: false},
			ExecResult:  sql.NullString{String: model.ResultFailed, Valid: true}, // 失败
			Message:     sql.NullString{String: "gRPC call failed: " + err.Error(), Valid: true},
		}
		if _, err := s.svcCtx.PlanExecLogModel.Insert(ctx, nil, logEntry); err != nil {
			logx.Errorf("Error inserting plan exec log for item %d: %v", execItem.Id, err)
		}
		return
	}
	if err != nil {
		logx.Errorf("Error unmarshaling response for exec item %d: %v", execItem.Id, err)
		if updateErr := s.svcCtx.PlanExecItemModel.UpdateStatusToFail(ctx, execItem.Id, model.ResultFailed, "Error unmarshaling response: "+err.Error(), "",
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); updateErr != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, updateErr)
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
			TraceId:     sql.NullString{String: "", Valid: false},
			ExecResult:  sql.NullString{String: model.ResultFailed, Valid: true}, // 失败
			Message:     sql.NullString{String: "Error unmarshaling response: " + err.Error(), Valid: true},
		}
		if _, err := s.svcCtx.PlanExecLogModel.Insert(ctx, nil, logEntry); err != nil {
			logx.Errorf("Error inserting plan exec log for item %d: %v", execItem.Id, err)
		}
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
		TraceId:     sql.NullString{String: "", Valid: false},
		ExecResult:  sql.NullString{String: res.ExecResult, Valid: res.ExecResult != ""},
		Message:     sql.NullString{String: res.Message, Valid: res.Message != ""},
		Reason:      sql.NullString{String: res.Reason, Valid: res.Reason != ""},
	}

	switch res.ExecResult {
	case model.ResultCompleted: // completed
		logx.Infof("gRPC call succeeded for exec item %d", execItem.Id)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToCompleted(ctx, execItem.Id, res.Message, res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); err != nil {
			logx.Errorf("Error updating plan exec item %d to completed: %v", execItem.Id, err)
		}
	case model.ResultFailed: // Failed
		logx.Infof("gRPC call returned failure for exec item %d: %s", execItem.Id, res.Message)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToFail(ctx, execItem.Id, model.ResultFailed, res.Message, res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); err != nil {
			logx.Errorf("Error updating plan exec item %d to failed: %v", execItem.Id, err)
		}
	case model.ResultDelayed:
		logx.Infof("gRPC call returned delayed for exec item %d: %s", execItem.Id, res.Message)
		currentTime := carbon.Now()
		delayTriggerTime := currentTime.AddMinutes(5).ToDateTimeString()
		delayReason := ""
		if len(res.Message) == 0 {
			delayReason = res.ExecResult
		} else {
			delayReason = res.Message
		}
		if res.DelayConfig == nil {
			logx.Errorf("No delay config provided for exec item %d", execItem.Id)
		} else {
			if len(res.DelayConfig.DelayReason) != 0 {
				delayReason = fmt.Sprintf("reason: %s, message: %s", res.DelayConfig.DelayReason, res.Message)
			}
			delayTime := carbon.ParseByLayout(res.DelayConfig.NextTriggerTime, carbon.DateTimeLayout)
			isTrue := true
			if delayTime.Error != nil || delayTime.IsInvalid() {
				logx.Errorf("Invalid delay time format for exec item %d: %s", execItem.Id, res.DelayConfig.NextTriggerTime)
				isTrue = false
			} else {
				if delayTime.Lt(currentTime) {
					logx.Errorf("Delay time for exec item %d is in the past: %v, current time: %v", execItem.Id, delayTime.ToDateTimeString(), currentTime.ToDateTimeString())
					isTrue = false
				}
			}
			if isTrue {
				delayTriggerTime = delayTime.ToDateTimeString()
			}
		}
		delayReason = fmt.Sprintf("%s, delay time: %s", delayReason, delayTriggerTime,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToDelayed(ctx, execItem.Id, res.ExecResult, res.Message, delayReason, delayTriggerTime,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); err != nil {
			logx.Errorf("Error updating plan exec item %d to delayed: %v", execItem.Id, err)
		}
	case model.ResultOngoing:
		logx.Infof("gRPC call returned ongoing for exec item %d: %s", execItem.Id, res.Message)
	default:
		logx.Errorf("Unknown execResult %s for exec item %d", res.ExecResult, execItem.Id)
		if err := s.svcCtx.PlanExecItemModel.UpdateStatusToCompleted(ctx, execItem.Id, res.Message, res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); err != nil {
			logx.Errorf("Error updating plan exec item %d to completed: %v", execItem.Id, err)
		}
	}
	// 插入执行日志
	if _, err := s.svcCtx.PlanExecLogModel.Insert(ctx, nil, logEntry); err != nil {
		logx.Errorf("Error inserting plan exec log for item %d: %v", execItem.Id, err)
	}

	batchCount, err := s.svcCtx.PlanBatchModel.UpdateBatchFinishedTime(ctx, execItem.BatchPk)
	if err != nil {
		logx.Errorf("Error updating batch %s completed time: %v", execItem.BatchId, err)
	}
	if batchCount > 0 {
		batchNotifyReq := streamevent.NotifyPlanEventReq{
			EventType:  1,
			PlanId:     execItem.PlanId,
			PlanType:   plan.Type.String,
			BatchId:    execItem.BatchId,
			Attributes: map[string]string{},
		}
		s.svcCtx.StreamEventCli.NotifyPlanEvent(ctx, &batchNotifyReq)
	}
	planCount, err := s.svcCtx.PlanModel.UpdateBatchFinishedTime(ctx, execItem.PlanPk)
	if err != nil {
		logx.Errorf("Error updating plan %s completed time: %v", execItem.PlanId, err)
	}
	if planCount > 0 {
		planPlanReq := streamevent.NotifyPlanEventReq{
			EventType: 0,
			PlanId:    execItem.PlanId,
			PlanType:  plan.Type.String,
			//BatchId:    execItem.BatchId,
			Attributes: map[string]string{},
		}
		s.svcCtx.StreamEventCli.NotifyPlanEvent(ctx, &planPlanReq)
	}
	logx.Infof("Successfully executed callback for plan exec item: id=%d", execItem.Id)
}
