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
		logx.WithContext(ctx).Errorf("[asynq-task] 反序列化补停 payload 失败: %v", err)
		return asynq.SkipRetry
	}

	// 将业务字段注入 context，下游日志自动携带
	ctx = logx.ContextWithFields(ctx,
		logx.Field("app", payload.App),
		logx.Field("stream", payload.Stream),
		logx.Field("uuid", payload.UUID),
	)

	if payload.App == "" || payload.Stream == "" {
		logx.WithContext(ctx).Error("[asynq-task] payload 缺少 app 或 stream")
		return asynq.SkipRetry
	}

	// UUID 校验：读取当前 state，比较 UUID
	st, err := h.svcCtx.StateStore.GetState(ctx, payload.App, payload.Stream)
	if err != nil {
		logx.WithContext(ctx).Errorf("[asynq-task] 获取 Redis 状态失败: %v", err)
		return err
	}
	if st != nil && payload.UUID != "" && st.UUID != payload.UUID {
		logx.WithContext(ctx).Infof("[asynq-task] UUID 不匹配，跳过补停: stateUUID=%s", st.UUID)
		return nil
	}

	logx.WithContext(ctx).Info("[asynq-task] 补停开始")
	l := logic.NewStopRelayPullLogic(ctx, h.svcCtx)
	err = l.StopRelayPullFromAsynq(payload.App, payload.Stream)
	if err != nil {
		logx.WithContext(ctx).Errorf("[asynq-task] 补停失败: %v", err)
		return err
	}
	logx.WithContext(ctx).Info("[asynq-task] 补停成功")
	return nil
}
