package cron

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"time"
	"zero-service/app/trigger/internal/execdelay"
	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/internal/triggerutil"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/common/gormx"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/streamevent"
	"zero-service/model"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
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

	dbType := gormx.GetDatabaseTypeFromDialector(s.svcCtx.DB.DB)
	execItem, err := gormmodel.LockTriggerItem(ctx, s.svcCtx.DB.WithContext(ctx).DB, dbType, 5*60*time.Second)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return false
		}
		planscope.CronLockScope().Logger(ctx).Errorf("定时扫表：抢占待调度执行项失败: %v", err)
		return false
	}
	if execItem == nil {
		return false
	}

	queryExecItem := &gormmodel.PlanExecItem{}
	queryErr := s.svcCtx.DB.WithContext(ctx).DB.Where("id = ?", execItem.Id).First(queryExecItem).Error
	if queryErr != nil {
		planscope.ExecCron(execItem).Logger(ctx).Errorf("扫表锁定后重新加载执行项失败: %v", queryErr)
		return false
	}
	plan := &gormmodel.Plan{}
	planErr := s.svcCtx.DB.WithContext(ctx).DB.Where("id = ?", queryExecItem.PlanPk).First(plan).Error
	if planErr != nil {
		planscope.ExecCron(execItem).Logger(ctx).Errorf("扫表锁定后加载计划失败: %v", planErr)
		return false
	}

	scope := planscope.TriggerScope(queryExecItem, plan)
	scope.Logger(ctx).Infof(
		"定时扫表：执行项已到调度时间，开始调用下游执行任务（HandlerPlanTaskEvent），next_trigger=%s",
		queryExecItem.NextTriggerTime.Format(time.RFC3339Nano),
	)

	// 更新计划 scan_flg
	if err := s.svcCtx.DB.WithContext(ctx).Model(&gormmodel.Plan{}).Where("id = ?", queryExecItem.PlanPk).Update("scan_flg", 1).Error; err != nil {
		scope.Logger(ctx).Errorf("更新计划 scan_flg 失败: %v", err)
	}

	// 更新批次 scan_flg
	if err := s.svcCtx.DB.WithContext(ctx).Model(&gormmodel.PlanBatch{}).Where("id = ?", queryExecItem.BatchPk).Update("scan_flg", 1).Error; err != nil {
		scope.Logger(ctx).Errorf("更新批次 scan_flg 失败: %v", err)
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

func (s *CronService) ExecuteCallback(ctx context.Context, execItem *gormmodel.PlanExecItem, plan *gormmodel.Plan) {
	traceID := trace.TraceIDFromContext(ctx)
	if execItem.RequestTimeout == 0 {
		execItem.RequestTimeout = s.svcCtx.Config.StreamEventConf.Timeout
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(execItem.RequestTimeout)*time.Millisecond+120*time.Second)
	defer cancel()
	scope := planscope.TriggerScope(execItem, plan)
	log := scope.Logger(ctx)
	log.Debug("计划执行回调：调用下游 HandlerPlanTaskEvent 开始")
	db := s.svcCtx.DB.WithContext(ctx).DB

	if err := gormmodel.UpdateExecItemStatusToRunning(ctx, db, execItem.Id, ""); err != nil {
		log.Errorf("回写执行项状态为「执行中」失败: %v", err)
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
		lock.SetExpire(triggerutil.RedisLockExpireSeconds(execItem.RequestTimeout))
		b, taskErr := lock.AcquireCtx(ctx)
		if taskErr != nil {
			errCh <- fmt.Errorf("调用下游前获取 Redis 锁失败: %w", taskErr)
			return
		}
		if !b {
			errCh <- fmt.Errorf("调用下游前未获取到 Redis 锁（资源忙）")
			return
		}
		defer func() {
			ok, releaseErr := lock.ReleaseCtx(ctx)
			if releaseErr != nil {
				logx.WithContext(ctx).Errorf("执行回调 Redis 锁释放失败: %v", releaseErr)
			} else if !ok {
				// 多见于锁 TTL 短于 RPC（默认未 SetExpire 仅 ~500ms）；已 SetExpire 后仅极端情况（key 被删/过期竞态）
				logx.WithContext(ctx).Debug("执行回调 Redis 锁释放：未删除 key（多因锁已过期或 token 不一致）")
			}
		}()
		res, taskErr = s.svcCtx.StreamEventCli.HandlerPlanTaskEvent(ctx, req)
		errCh <- taskErr
	})
	err = <-errCh
	if err != nil {
		log.Errorf("调用下游 HandlerPlanTaskEvent 失败: %v", err)
		return
	}

	execLog := &gormmodel.PlanExecLog{
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

	resultLog := scope.WithFields(
		logx.Field("exec_result", res.ExecResult),
		logx.Field("message", res.Message),
		logx.Field("reason", res.Reason),
	).Logger(ctx)

	switch res.ExecResult {
	case model.ResultCompleted:
		resultLog.Info("下游返回：执行完成（completed）")
		err = gormmodel.UpdateExecItemStatusToCompleted(ctx, db, execItem.Id, res.Message, res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		)
		if err != nil {
			log.Errorf("回写执行项为「已完成」失败: %v", err)
		}
	case model.ResultFailed:
		resultLog.Info("下游返回：执行失败（failed）")
		if err := gormmodel.UpdateExecItemStatusToFail(ctx, db, execItem.Id, model.ResultFailed, res.Message, res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		); err != nil {
			log.Errorf("回写执行项为「失败」失败: %v", err)
		}
	case model.ResultDelayed:
		resultLog.Info("下游返回：延期重试（delayed），将按规则更新下次触发时间")
		currentTime := carbon.Now()
		dr := execdelay.Resolve(res.DelayConfig, res.Message, res.Reason, currentTime, execdelay.ModeDelayed)
		execdelay.LogWarnings(ctx, scope, dr)
		delayReason := execdelay.FinalReason(dr.ReasonStem, dr.NextTrigger)
		execLog.Reason = sql.NullString{String: delayReason, Valid: true}
		nextTime := carbon.Parse(dr.NextTrigger)
		if nextTime.Error != nil {
			log.Errorf("解析延期触发时间失败: %v", nextTime.Error)
		} else {
			err = gormmodel.UpdateExecItemStatusToDelayed(ctx, db, execItem.Id, res.ExecResult, res.Message, delayReason, nextTime.StdTime(),
				[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
			)
			if err != nil {
				log.Errorf("回写执行项为「延期等待」失败: %v", err)
			}
		}
	case model.ResultOngoing:
		resultLog.Info("下游返回：仍在进行（ongoing），将按规则更新下次触发时间")
		currentTime := carbon.Now()
		or := execdelay.Resolve(res.DelayConfig, res.Message, res.Reason, currentTime, execdelay.ModeOngoing)
		execdelay.LogWarnings(ctx, scope, or)
		delayReason := execdelay.FinalReason(or.ReasonStem, or.NextTrigger)
		execLog.Reason = sql.NullString{String: delayReason, Valid: true}
		var nextTime *time.Time
		if or.NextTrigger != "" {
			nt := carbon.Parse(or.NextTrigger)
			if nt.Error != nil {
				log.Errorf("解析 ongoing 触发时间失败: %v", nt.Error)
			} else {
				t := nt.StdTime()
				nextTime = &t
			}
		}
		err = gormmodel.UpdateExecItemStatusToOngoing(ctx, db, execItem.Id, res.ExecResult, delayReason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated}, nextTime, true,
		)
		if err != nil {
			log.Errorf("回写执行项为「进行中」失败: %v", err)
		}
	case model.ResultTerminated:
		resultLog.Info("下游返回：已终止（terminated）")
		err = gormmodel.UpdateExecItemStatusToTerminated(ctx, db, execItem.Id, res.Message, res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		)
		if err != nil {
			log.Errorf("回写执行项为「已终止」失败: %v", err)
		}
	default:
		log.Errorf("下游返回未知结果 exec_result=%q，将按 completed 回写库", res.ExecResult)
		err = gormmodel.UpdateExecItemStatusToCompleted(ctx, db, execItem.Id, res.Message, res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		)
		if err != nil {
			log.Errorf("未知结果按「已完成」回写执行项失败: %v", err)
		}
	}

	// 插入执行日志
	if err := db.Create(execLog).Error; err != nil {
		log.Errorf("写入执行流水 plan_exec_log 失败: %v", err)
	}

	batchCount, err := gormmodel.UpdatePlanBatchFinishedTime(ctx, db, execItem.BatchPk)
	if err != nil {
		log.Errorf("更新批次 finished_time（用于收尾判断）失败: %v", err)
	}
	if batchCount > 0 {
		batchNotifyReq := streamevent.NotifyPlanEventReq{
			EventType:  streamevent.PlanEventType_BATCH_FINISHED,
			PlanId:     execItem.PlanId,
			PlanType:   plan.Type.String,
			BatchId:    execItem.BatchId,
			Attributes: map[string]string{},
		}
		scope.WithFields(logx.Field("notify_event", planscope.NotifyEventBatchFinished)).Logger(ctx).Info("下游通知：调用 NotifyPlanEvent（批次收尾）")
		s.svcCtx.StreamEventCli.NotifyPlanEvent(ctx, &batchNotifyReq)
	}
	planCount, err := gormmodel.UpdatePlanFinishedTime(ctx, db, execItem.PlanPk)
	if err != nil {
		log.Errorf("更新计划 finished_time（用于收尾判断）失败: %v", err)
	}
	if planCount > 0 {
		planPlanReq := streamevent.NotifyPlanEventReq{
			EventType:  streamevent.PlanEventType_PLAN_FINISHED,
			PlanId:     execItem.PlanId,
			PlanType:   plan.Type.String,
			Attributes: map[string]string{},
		}
		scope.WithFields(logx.Field("notify_event", planscope.NotifyEventPlanFinished)).Logger(ctx).Info("下游通知：调用 NotifyPlanEvent（计划收尾）")
		s.svcCtx.StreamEventCli.NotifyPlanEvent(ctx, &planPlanReq)
	}
	scope.WithFields(
		logx.Field("batch_notify_rows", batchCount),
		logx.Field("plan_notify_rows", planCount),
	).Logger(ctx).Info("计划执行回调：下游处理结束（batch_notify_rows、plan_notify_rows 为本次是否触发批次/计划收尾通知）")
}
