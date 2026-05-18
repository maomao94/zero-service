package logic

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"
)

type HealthLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewHealthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HealthLogic {
	return &HealthLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *HealthLogic) Health(_ *aisolo.HealthReq) (*aisolo.HealthResp, error) {
	deps := map[string]string{}
	if l.svcCtx.ChatModel != nil {
		deps["chat_model"] = "ok"
	} else {
		deps["chat_model"] = "missing"
	}
	if l.svcCtx.Kit != nil {
		deps["tool_kit"] = "ok"
	} else {
		deps["tool_kit"] = "missing"
		if l.svcCtx.KitInitErr != "" {
			deps["tool_kit_error"] = l.svcCtx.KitInitErr
		}
	}
	if l.svcCtx.RuntimeRunner != nil {
		deps["runtime_runner"] = "ok"
	} else {
		deps["runtime_runner"] = "missing"
	}
	if l.svcCtx.RuntimeTools != nil {
		deps["runtime_tools"] = "ok"
	} else {
		deps["runtime_tools"] = "missing"
	}
	if l.svcCtx.Executor != nil {
		deps["executor"] = "ok"
	} else {
		deps["executor"] = "missing"
	}
	if l.svcCtx.Config.Knowledge.Enabled {
		deps["knowledge_backend"] = l.svcCtx.Config.Knowledge.EffectiveBackend()
	}
	if l.svcCtx.Knowledge != nil {
		deps["knowledge"] = "ok"
	} else if l.svcCtx.Config.Knowledge.Enabled {
		deps["knowledge"] = "misconfigured"
		if l.svcCtx.KnowledgeInitErr != "" {
			deps["knowledge_error"] = l.svcCtx.KnowledgeInitErr
		}
	} else {
		deps["knowledge"] = "disabled"
	}
	return &aisolo.HealthResp{
		Status:       "ok",
		Version:      "refactor",
		Timestamp:    time.Now().Unix(),
		Dependencies: deps,
	}, nil
}
