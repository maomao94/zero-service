package task

import (
	"context"
	"encoding/json"

	"zero-service/app/oryxserver/internal/relay"
	"zero-service/app/oryxserver/internal/svc"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
)

// ReconcileHandler 补拉任务处理器：薄壳，业务逻辑全部在 relay.DistributedRelay.Reconcile。
type ReconcileHandler struct {
	svcCtx *svc.ServiceContext
}

func NewReconcileHandler(svcCtx *svc.ServiceContext) *ReconcileHandler {
	return &ReconcileHandler{svcCtx: svcCtx}
}

func (h *ReconcileHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload relay.ReconcilePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logx.WithContext(ctx).Errorf("[asynq] unmarshal reconcile payload: %v", err)
		return asynq.SkipRetry
	}
	logger := logx.WithContext(ctx).WithFields(
		logx.Field("taskType", t.Type()),
		logx.Field("taskId", t.ResultWriter().TaskID()),
		logx.Field("target", payload.Target),
		logx.Field("retryCount", payload.RetryCount),
	)
	if payload.Target == "" {
		logger.Error("[asynq] missing target in payload")
		return asynq.SkipRetry
	}
	logger.Info("[asynq] reconcile start")
	if err := h.svcCtx.DistRelay.Reconcile(ctx, payload.Target, payload.RetryCount); err != nil {
		logger.Errorf("[asynq] reconcile failed: %v", err)
		return err
	}
	logger.Info("[asynq] reconcile success")
	return nil
}
