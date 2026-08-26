package task

import (
	"context"
	"encoding/json"

	"zero-service/app/oryxserver/internal/logic"
	"zero-service/app/oryxserver/internal/relay"
	"zero-service/app/oryxserver/internal/svc"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
)

// StopHandler 补停任务处理器：重跑 StopRelayPull 核心逻辑（本地停止 → 广播）。
type StopHandler struct {
	svcCtx *svc.ServiceContext
}

func NewStopHandler(svcCtx *svc.ServiceContext) *StopHandler {
	return &StopHandler{svcCtx: svcCtx}
}

func (h *StopHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload relay.StopPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logx.WithContext(ctx).Errorf("[asynq] unmarshal stop payload: %v", err)
		return asynq.SkipRetry
	}
	logger := logx.WithContext(ctx).WithFields(
		logx.Field("taskType", t.Type()),
		logx.Field("taskId", t.ResultWriter().TaskID()),
		logx.Field("target", payload.Target),
		logx.Field("app", payload.App),
		logx.Field("stream", payload.Stream),
	)
	if payload.App == "" || payload.Stream == "" {
		logger.Error("[asynq] missing app or stream in payload")
		return asynq.SkipRetry
	}

	logger.Info("[asynq] stop start")
	l := logic.NewStopRelayPullLogic(ctx, h.svcCtx)
	err := l.StopRelayPullFromAsynq(payload.App, payload.Stream)
	if err != nil {
		logger.Errorf("[asynq] stop failed: %v", err)
		return err
	}
	logger.Info("[asynq] stop success")
	return nil
}
