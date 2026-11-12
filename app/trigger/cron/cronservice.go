package cron

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"time"
	"zero-service/app/trigger/internal/confirm"
	"zero-service/app/trigger/internal/execdelay"
	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/internal/triggerutil"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/common/carbonx"
	"zero-service/common/gormx"
	"zero-service/common/limitx"
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

const (
	// planBatchDispatchKeyPrefix 批次下发令牌桶的 Redis key 前缀。
	planBatchDispatchKeyPrefix = "trigger:rate:batch:"
	// planBatchDispatchRate 单批次下发令牌桶速率（个/秒）。
	// 批次二次确认晚于触发时间时，整批执行项会同时过期；按批次限速避免瞬时全部下发。
	planBatchDispatchRate = 10
	// planBatchDispatchBurst 单批次令牌桶容量，允许的瞬时下发条数。
	planBatchDispatchBurst = 10
	// planTriggerLockTTL 执行项抢占租约：抢占后未正常下发时，执行项最迟在这个时间后重新参与扫表。
	planTriggerLockTTL = 5 * time.Minute
	// deferTriggerItemAttempts 限速推迟的写入尝试次数，用于吸收瞬时数据库故障。
	deferTriggerItemAttempts = 3
	// deferTriggerItemRetryInterval 限速推迟失败后的重试间隔基数，按尝试次数线性退避。
	deferTriggerItemRetryInterval = 20 * time.Millisecond
)

type CronService struct {
	cancelChan chan struct{}
	svcCtx     *svc.ServiceContext
	taskRunner *threading.TaskRunner
	// batchLimiter 按批次限制执行项下发速率，多实例共享 Redis 中的令牌桶。
	batchLimiter *limitx.TokenBucket
	// lastConfirmScan 控制批次确认扫描频率，避免执行项繁忙时高频查询确认表。
	lastConfirmScan time.Time
}

func NewCronService(svcCtx *svc.ServiceContext) *CronService {
	return &CronService{
		svcCtx:     svcCtx,
		taskRunner: threading.NewTaskRunner(16),
		batchLimiter: limitx.NewTokenBucket(
			svcCtx.Redis,
			limitx.WithRate(planBatchDispatchRate),
			limitx.WithBurst(planBatchDispatchBurst),
			limitx.WithKeyPrefix(planBatchDispatchKeyPrefix),
		),
	}
}

func (s *CronService) Start() {
	if s.cancelChan != nil {
		return
	}
	// Create cancellation channel for proper goroutine termination
	s.cancelChan = make(chan struct{})
	logx.Info(planscope.CronPlanLogMessage("cron service started"))

	go s.scanLoop()
}

func (s *CronService) Stop() {
	if s.cancelChan != nil {
		// Close the cancellation channel to signal goroutine termination
		close(s.cancelChan)
		s.cancelChan = nil
		logx.Info(planscope.CronPlanLogMessage("cron service stopped"))
	}
}

func (s *CronService) scanLoop() {
	for {
		itemsProcessed := s.ScanPlanExecItem()
		if time.Since(s.lastConfirmScan) >= planConfirmScanInterval {
			s.ScanPlanConfirm()
			s.lastConfirmScan = time.Now()
		}
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
	execItem, err := gormmodel.LockTriggerItem(ctx, s.svcCtx.DB.WithContext(ctx).DB, dbType, planTriggerLockTTL)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return false
		}
		lockScope := planscope.CronLockScope()
		if errors.Is(err, model.ErrNoRowsUpdate) {
			// 多实例并发抢占时的高频良性竞争，降为 Debug，避免挤占关键日志。
			lockScope.Logger(ctx).Debug(lockScope.LogMessage("定时扫表：待调度执行项已被其他调度器抢占"))
			return false
		}
		lockScope.Logger(ctx).Errorf(lockScope.LogMessage("定时扫表：抢占待调度执行项失败: %v"), err)
		return false
	}
	if execItem == nil {
		return false
	}
	execScope := planscope.ExecCron(execItem)
	execLog := execScope.Logger(ctx)

	queryExecItem := &gormmodel.PlanExecItem{}
	queryErr := s.svcCtx.DB.WithContext(ctx).DB.Where("id = ?", execItem.Id).First(queryExecItem).Error
	if queryErr != nil {
		execLog.Errorf(execScope.LogMessage("扫表锁定后重新加载执行项失败: %v"), queryErr)
		return false
	}
	plan := &gormmodel.Plan{}
	planErr := s.svcCtx.DB.WithContext(ctx).DB.Where("id = ?", queryExecItem.PlanPk).First(plan).Error
	if planErr != nil {
		execLog.Errorf(execScope.LogMessage("扫表锁定后加载计划失败: %v"), planErr)
		return false
	}
	if plan.Status != model.PlanStatusEnabled || plan.FinishedTime.Valid {
		execLog.Info(execScope.LogMessage("扫表锁定后计划已非启用状态，停止调用下游"))
		return false
	}
	planBatch := &gormmodel.PlanBatch{}
	batchErr := s.svcCtx.DB.WithContext(ctx).DB.Where("id = ?", queryExecItem.BatchPk).First(planBatch).Error
	if batchErr != nil {
		execLog.Errorf(execScope.LogMessage("扫表锁定后加载计划批次失败: %v"), batchErr)
		return false
	}
	if planBatch.Status != model.PlanStatusEnabled || planBatch.FinishedTime.Valid {
		execLog.Info(execScope.LogMessage("扫表锁定后计划批次已非启用状态，停止调用下游"))
		return false
	}

	// 批次级限速：晚确认后整批执行项会同时过期，按批次令牌桶分散放行，避免下游瞬时并发。
	decision := s.allowBatchDispatch(ctx, execItem, planBatch)
	if !decision.allowed {
		nextTriggerTime := time.Now().Add(decision.wait)
		if err := s.deferTriggerItem(ctx, execItem, nextTriggerTime); err != nil {
			if errors.Is(err, model.ErrNoRowsUpdate) {
				// 执行项已被其它链路推进到终态或已推迟，本次推迟不再生效，属良性并发。
				execLog.Info(execScope.LogMessage("批次限速：执行项已不在执行中，跳过推迟"))
			} else {
				// 推迟失败时执行项停在抢占租约上，最迟在租约到期后重新参与扫表。
				execLog.WithFields(
					logx.Field("wait_ms", decision.wait.Milliseconds()),
					logx.Field("lease_retry_at", time.Now().Add(planTriggerLockTTL).Format(time.RFC3339Nano)),
				).Errorf(execScope.LogMessage("批次限速：推迟执行项失败，回退到抢占租约重试: %v"), err)
			}
		} else {
			execLog.WithFields(
				logx.Field("throttle", "batch_token_bucket"),
				logx.Field("wait_ms", decision.wait.Milliseconds()),
				logx.Field("next_trigger", nextTriggerTime.Format(time.RFC3339Nano)),
				logx.Field("confirmed_time", decision.confirmedAt.Format(time.RFC3339Nano)),
				logx.Field("plan_trigger_time", decision.planTriggerAt.Format(time.RFC3339Nano)),
				logx.Field("rate", planBatchDispatchRate),
				logx.Field("burst", planBatchDispatchBurst),
			).Info(execScope.LogMessage("批次限速：批次令牌不足，执行项已推迟"))
		}
		return true
	}

	scope := planscope.TriggerScope(queryExecItem, plan)
	scope.Logger(ctx).Infof(
		scope.LogMessage("定时扫表：执行项已到调度时间，开始调用下游执行任务（HandlerPlanTaskEvent），next_trigger=%s"),
		queryExecItem.NextTriggerTime.Format(time.RFC3339Nano),
	)

	// 更新计划 scan_flg
	if err := s.svcCtx.DB.WithContext(ctx).Model(&gormmodel.Plan{}).Where("id = ?", queryExecItem.PlanPk).Update("scan_flg", 1).Error; err != nil {
		scope.Logger(ctx).Errorf(scope.LogMessage("更新计划 scan_flg 失败: %v"), err)
	}

	// 更新批次 scan_flg
	if err := s.svcCtx.DB.WithContext(ctx).Model(&gormmodel.PlanBatch{}).Where("id = ?", queryExecItem.BatchPk).Update("scan_flg", 1).Error; err != nil {
		scope.Logger(ctx).Errorf(scope.LogMessage("更新批次 scan_flg 失败: %v"), err)
	}

	s.ExecuteCallback(ctx, queryExecItem, plan)
	return true
}

// deferTriggerItem 把被限速拦截的执行项恢复到抢占前状态并推迟下一次触发时间。
// 瞬时数据库故障按短间隔重试；ErrNoRowsUpdate 表示执行项已被其它链路推进，直接返回不再重试。
func (s *CronService) deferTriggerItem(ctx context.Context, claimed *gormmodel.PlanExecItem, next time.Time) error {
	var err error
	for attempt := 1; attempt <= deferTriggerItemAttempts; attempt++ {
		err = gormmodel.DeferTriggerItem(ctx, s.svcCtx.DB.WithContext(ctx).DB, claimed, next)
		if err == nil || errors.Is(err, model.ErrNoRowsUpdate) {
			return err
		}
		if attempt < deferTriggerItemAttempts {
			time.Sleep(time.Duration(attempt) * deferTriggerItemRetryInterval)
		}
	}
	return err
}

// batchDispatchDecision 批次级限速的判定结果。命中限速时带上确认时间与批次触发时间，
// 便于在一条日志里说明「为什么这个批次被判为晚确认」。
type batchDispatchDecision struct {
	allowed       bool
	wait          time.Duration
	confirmedAt   time.Time
	planTriggerAt time.Time
}

// allowBatchDispatch 获取批次级下发令牌。仅对「确认时间晚于批次触发时间」的批次限速，提前确认或未配置确认的批次保持原有下发行为。
// 执行项已在 LockTriggerItem 中抢占成功，这里在锁内查询确认记录并做时间判断，不把时间比较放到 claim SQL 里。
// 查询或限流组件异常时放行，避免限流故障阻塞调度。
func (s *CronService) allowBatchDispatch(ctx context.Context, execItem *gormmodel.PlanExecItem, batch *gormmodel.PlanBatch) batchDispatchDecision {
	allow := batchDispatchDecision{allowed: true}
	if !s.batchLimiter.Enabled() || batch == nil || !batch.PlanTriggerTime.Valid {
		return allow
	}
	scope := planscope.ExecCron(execItem)
	record, err := gormmodel.FindPlanConfirmByTarget(ctx, s.svcCtx.DB.WithContext(ctx).DB, model.ConfirmTypeBatch, batch.Id)
	if err != nil {
		scope.Logger(ctx).Errorf(scope.LogMessage("批次限速：查询确认记录失败，本次放行 batch_pk=%s err=%v"), batch.Id, err)
		return allow
	}
	if record == nil || !record.ConfirmedTime.Valid || !record.ConfirmedTime.Time.After(batch.PlanTriggerTime.Time) {
		return allow
	}
	allowed, wait, err := s.batchLimiter.Allow(ctx, batch.Id)
	if err != nil {
		scope.Logger(ctx).Errorf(scope.LogMessage("批次限速：获取令牌失败，本次放行 batch_pk=%s err=%v"), batch.Id, err)
		return allow
	}
	return batchDispatchDecision{
		allowed:       allowed,
		wait:          wait,
		confirmedAt:   record.ConfirmedTime.Time,
		planTriggerAt: batch.PlanTriggerTime.Time,
	}
}

func (s *CronService) ExecuteCallback(ctx context.Context, execItem *gormmodel.PlanExecItem, plan *gormmodel.Plan) {
	traceID := trace.TraceIDFromContext(ctx)
	if execItem.RequestTimeout == 0 {
		execItem.RequestTimeout = s.svcCtx.Config.StreamEventConf.Timeout
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(execItem.RequestTimeout)*time.Millisecond+120*time.Second)
	defer cancel()
	scope := planscope.TriggerScope(execItem, plan)
	log := scope.Logger(ctx)
	log.Debug(scope.LogMessage("计划执行回调：调用下游 HandlerPlanTaskEvent 开始"))
	db := s.svcCtx.DB.WithContext(ctx).DB

	if err := gormmodel.UpdateExecItemStatusToRunning(ctx, db, execItem.Id, ""); err != nil {
		log.Errorf(scope.LogMessage("回写执行项状态为「执行中」失败: %v"), err)
		return
	}
	callPlan := &streamevent.PlanPb{
		CreateTime:  carbonx.FormatDateTime(plan.CreateTime),
		UpdateTime:  carbonx.FormatDateTime(plan.UpdateTime),
		CreateUser:  plan.CreateUser.String,
		UpdateUser:  plan.UpdateUser.String,
		DeptCode:    plan.DeptCode.String,
		Id:          plan.Id,
		PlanId:      plan.PlanId,
		PlanName:    plan.PlanName.String,
		Type:        plan.Type.String,
		GroupId:     plan.GroupId.String,
		Description: plan.Description.String,
		StartTime:   carbonx.FormatDateTime(plan.StartTime),
		EndTime:     carbonx.FormatDateTime(plan.EndTime),
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
		PlanTriggerTime: carbonx.FormatDateTime(execItem.PlanTriggerTime),
		LastResult:      execItem.LastResult.String,
		LastMessage:     execItem.LastMessage.String,
		LastReason:      execItem.LastReason.String,
	}
	if execItem.LastTriggerTime.Valid {
		req.LastTriggerTime = carbonx.FormatDateTime(execItem.LastTriggerTime.Time)
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
			ok, releaseErr := lock.Release()
			if releaseErr != nil {
				logx.WithContext(ctx).Errorf(scope.LogMessage("执行回调 Redis 锁释放失败: %v"), releaseErr)
			} else if !ok {
				// 多见于锁 TTL 短于 RPC（默认未 SetExpire 仅 ~500ms）；已 SetExpire 后仅极端情况（key 被删/过期竞态）
				logx.WithContext(ctx).Debug(scope.LogMessage("执行回调 Redis 锁释放：未删除 key（多因锁已过期或 token 不一致）"))
			}
		}()
		res, taskErr = s.svcCtx.StreamEventCli.HandlerPlanTaskEvent(ctx, req)
		errCh <- taskErr
	})
	err = <-errCh
	if err != nil {
		log.Errorf(scope.LogMessage("调用下游 HandlerPlanTaskEvent 失败: %v"), err)
		// 请求没通，下游是否已收到未知：不改 status，仅打标记，靠 next_trigger_time 过期后重投。
		// 使用独立短超时，避免任务排队或调用链耗尽外层 deadline 时阻止失败标记落库。
		markCtx, markCancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer markCancel()
		if markErr := gormmodel.MarkExecItemUnresolved(markCtx, db, execItem.Id, model.ResultDispatchFailed, err.Error()); markErr != nil {
			log.Errorf(scope.LogMessage("标记执行项投递失败状态失败: %v"), markErr)
		}
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
		resultLog.Info(scope.LogMessage("下游返回：执行完成（completed）"))
		err = gormmodel.UpdateExecItemStatusToCompleted(ctx, db, execItem.Id, res.Message, res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		)
		if err != nil {
			log.Errorf(scope.LogMessage("回写执行项为「已完成」失败: %v"), err)
		}
	case model.ResultFailed:
		resultLog.Info(scope.LogMessage("下游返回：执行失败（failed）"))
		err = gormmodel.UpdateExecItemStatusToFail(ctx, db, execItem.Id, model.ResultFailed, res.Message, res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		)
	case model.ResultDelayed:
		resultLog.Info(scope.LogMessage("下游返回：延期重试（delayed），将按规则更新下次触发时间"))
		currentTime := carbon.Now()
		dr := execdelay.Resolve(res.DelayConfig, res.Message, res.Reason, currentTime, execdelay.ModeDelayed)
		execdelay.LogWarnings(ctx, scope, dr)
		delayReason := execdelay.FinalReason(dr.ReasonStem, dr.NextTrigger, execdelay.ModeDelayed)
		execLog.Reason = sql.NullString{String: delayReason, Valid: true}
		nextTime := carbon.Parse(dr.NextTrigger)
		if nextTime.Error != nil {
			log.Errorf(scope.LogMessage("解析延期触发时间失败: %v"), nextTime.Error)
			return
		} else {
			err = gormmodel.UpdateExecItemStatusToDelayed(ctx, db, execItem.Id, res.ExecResult, res.Message, delayReason, nextTime.StdTime(),
				[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
			)
			if err != nil {
				log.Errorf(scope.LogMessage("回写执行项为「延期等待」失败: %v"), err)
			}
		}
	case model.ResultOngoing:
		resultLog.Info(scope.LogMessage("下游返回：仍在进行（ongoing），将按规则更新下次触发时间"))
		currentTime := carbon.Now()
		or := execdelay.Resolve(res.DelayConfig, res.Message, res.Reason, currentTime, execdelay.ModeOngoing)
		execdelay.LogWarnings(ctx, scope, or)
		delayReason := execdelay.FinalReason(or.ReasonStem, or.NextTrigger, execdelay.ModeOngoing)
		execLog.Reason = sql.NullString{String: delayReason, Valid: true}
		var nextTime *time.Time
		if or.NextTrigger != "" {
			nt := carbon.Parse(or.NextTrigger)
			if nt.Error != nil {
				log.Errorf(scope.LogMessage("解析 ongoing 触发时间失败: %v"), nt.Error)
			} else {
				t := nt.StdTime()
				nextTime = &t
			}
		}
		err = gormmodel.UpdateExecItemStatusToOngoing(ctx, db, execItem.Id, res.ExecResult, delayReason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated}, nextTime, true,
		)
		if err != nil {
			log.Errorf(scope.LogMessage("回写执行项为「进行中」失败: %v"), err)
		}
	case model.ResultTerminated:
		resultLog.Info(scope.LogMessage("下游返回：已终止（terminated）"))
		err = gormmodel.UpdateExecItemStatusToTerminated(ctx, db, execItem.Id, res.Message, res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		)
		if err != nil {
			log.Errorf(scope.LogMessage("回写执行项为「已终止」失败: %v"), err)
		}
	default:
		log.Errorf(scope.LogMessage("下游返回未知结果 exec_result=%q，按失败退避重试处理"), res.ExecResult)
		err = gormmodel.UpdateExecItemStatusToFail(ctx, db, execItem.Id, model.ResultUnknown,
			fmt.Sprintf("未知 exec_result=%q; %s", res.ExecResult, res.Message), res.Reason,
			[]int{model.StatusRunning}, []int{model.StatusCompleted, model.StatusTerminated},
		)
		if err != nil {
			log.Errorf(scope.LogMessage("未知结果按「失败重试」回写执行项失败: %v"), err)
		}
	}
	if err != nil {
		log.Errorf(scope.LogMessage("执行结果条件回写失败，停止写入流水和聚合收尾: %v"), err)
		return
	}

	// 插入执行日志
	if err := db.Create(execLog).Error; err != nil {
		log.Errorf(scope.LogMessage("写入执行流水 plan_exec_log 失败: %v"), err)
	}

	batch := &gormmodel.PlanBatch{PlanPk: execItem.PlanPk, PlanId: execItem.PlanId, BatchId: execItem.BatchId}
	batch.Id = execItem.BatchPk
	batchCount, planCount, notifyErr := confirm.NotifyBatchFinishedIfNeeded(ctx, db, s.svcCtx.StreamEventCli, plan, batch, res.Reason)
	if notifyErr != nil {
		log.Errorf(scope.LogMessage("更新批次/计划 finished_time 失败: %v"), notifyErr)
	}
	scope.WithFields(
		logx.Field("batch_notify_rows", batchCount),
		logx.Field("plan_notify_rows", planCount),
	).Logger(ctx).Info(scope.LogMessage("计划执行回调：下游处理结束（batch_notify_rows、plan_notify_rows 为本次是否触发批次/计划收尾通知）"))
}
