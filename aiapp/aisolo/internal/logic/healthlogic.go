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
	if l.svcCtx.Executor != nil {
		deps["executor"] = "ok"
	} else {
		deps["executor"] = "missing"
	}
	return &aisolo.HealthResp{
		Status:       "ok",
		Version:      "refactor",
		Timestamp:    time.Now().Unix(),
		Dependencies: deps,
	}, nil
}
