package task

import (
	"context"
	"encoding/json"

	"zero-service/app/oryxserver/internal/relay"
	"zero-service/app/oryxserver/internal/svc"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
)

// ReconcileHandler 补拉任务处理器：薄壳，业务逻辑全部在 relay.RelayRegistry.Reconcile。
type ReconcileHandler struct {
	svcCtx *svc.ServiceContext
}

func NewReconcileHandler(svcCtx *svc.ServiceContext) *ReconcileHandler {
	return &ReconcileHandler{svcCtx: svcCtx}
}

func (h *ReconcileHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload relay.ReconcilePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logx.WithContext(ctx).Errorf("[asynq-task] 反序列化补拉 payload 失败: %v", err)
		return asynq.SkipRetry
	}
	logger := logx.WithContext(ctx).WithFields(
		logx.Field("taskType", t.Type()),
		logx.Field("taskId", t.ResultWriter().TaskID()),
		logx.Field("uid", payload.UID),
		logx.Field("retryCount", payload.RetryCount),
	)
	if payload.UID == "" {
		logger.Error("[asynq-task] payload 缺少 uid")
		return asynq.SkipRetry
	}
	logger.Info("[asynq-task] 补拉开始")
	if err := h.svcCtx.RelayRegistry.Reconcile(ctx, payload.UID, payload.Source, payload.RetryCount); err != nil {
		logger.Errorf("[asynq-task] 补拉失败: %v", err)
		return err
	}
	logger.Info("[asynq-task] 补拉成功")
	return nil
}
