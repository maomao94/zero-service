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

	// 将业务字段注入 context，下游日志自动携带
	ctx = logx.ContextWithFields(ctx,
		logx.Field("app", payload.App),
		logx.Field("stream", payload.Stream),
		logx.Field("uuid", payload.UUID),
		logx.Field("retryCount", payload.RetryCount),
	)

	if payload.App == "" || payload.Stream == "" || payload.UUID == "" {
		logx.WithContext(ctx).Error("[asynq-task] payload 缺少 app/stream/uuid")
		return asynq.SkipRetry
	}

	logx.WithContext(ctx).Info("[asynq-task] 补拉开始")
	if err := h.svcCtx.RelayRegistry.Reconcile(ctx, payload.App, payload.Stream, payload.UUID, payload.RetryCount); err != nil {
		logx.WithContext(ctx).Errorf("[asynq-task] 补拉失败: %v", err)
		return err
	}
	logx.WithContext(ctx).Info("[asynq-task] 补拉成功")
	return nil
}
