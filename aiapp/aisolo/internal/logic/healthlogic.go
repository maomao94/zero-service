package logic

import (
	"context"
	"time"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type HealthLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewHealthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HealthLogic {
	return &HealthLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *HealthLogic) Health(in *aisolo.HealthReq) (resp *aisolo.HealthResp, err error) {
	dependencies := make(map[string]string)

	// 检查模型连接状态
	if l.svcCtx.ChatModel != nil {
		dependencies["model"] = "ok"
	} else {
		dependencies["model"] = "error"
	}

	// 检查记忆存储状态
	if l.svcCtx.MemoryStorage != nil {
		dependencies["memory"] = "ok"
	} else {
		dependencies["memory"] = "error"
	}

	// 检查工具管理器状态
	if l.svcCtx.ToolManager != nil {
		dependencies["tools"] = "ok"
	} else {
		dependencies["tools"] = "error"
	}

	// 检查Agent池状态
	if l.svcCtx.AgentPool != nil {
		dependencies["agent_pool"] = "ok"
	} else {
		dependencies["agent_pool"] = "error"
	}

	// 整体状态判断
	status := "ok"
	for _, depStatus := range dependencies {
		if depStatus == "error" {
			status = "error"
			break
		}
	}

	return &aisolo.HealthResp{
		Status:       status,
		Version:      "1.0.0",
		Timestamp:    time.Now().Unix(),
		Dependencies: dependencies,
	}, nil
}
