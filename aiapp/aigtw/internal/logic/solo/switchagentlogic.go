package solo

import (
	"context"
	"zero-service/aiapp/aisolo/aisolo"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SwitchAgentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSwitchAgentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SwitchAgentLogic {
	return &SwitchAgentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SwitchAgentLogic) SwitchAgent(req *types.SoloSwitchAgentRequest) (resp *types.SoloSwitchAgentResponse, err error) {
	protoReq := &aisolo.GetSessionReq{
		SessionId: req.SessionId,
	}

	result, err := l.svcCtx.AiSoloCli.GetSession(l.ctx, protoReq)
	if err != nil {
		l.Logger.Errorf("switch agent failed: %v", err)
		return nil, err
	}

	return &types.SoloSwitchAgentResponse{
		Session: &types.SoloSessionInfo{
			SessionId:    result.Session.SessionId,
			UserId:       result.Session.UserId,
			AgentMode:    req.AgentMode,
			Title:        result.Session.Title,
			CreatedAt:    result.Session.CreatedAt,
			UpdatedAt:    result.Session.UpdatedAt,
			MessageCount: int(result.Session.MessageCount),
			LastMessage:  result.Session.LastMessage,
		},
		NewAgent: &types.SoloAgentInfo{
			Type:         req.AgentMode,
			Name:         req.AgentMode,
			Description:  "Agent mode: " + req.AgentMode,
			Capabilities: []string{},
			Available:    true,
		},
	}, nil
}
