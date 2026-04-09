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

func (l *SwitchAgentLogic) SwitchAgent(req *types.SoloSwitchAgentReq) (resp *types.SoloSwitchAgentResp, err error) {
	protoReq := &aisolo.SessionRequest{
		SessionId: req.SessionId,
	}

	result, err := l.svcCtx.EinoCli.GetSession(l.ctx, protoReq)
	if err != nil {
		l.Logger.Errorf("switch agent failed: %v", err)
		return nil, err
	}

	return &types.SoloSwitchAgentResp{
		Session: &types.SoloSessionInfo{
			SessionId:    result.SessionId,
			UserId:       result.UserId,
			AgentMode:    req.AgentMode,
			Title:        result.Title,
			CreatedAt:    result.CreatedAt,
			UpdatedAt:    result.UpdatedAt,
			MessageCount: int(result.MessageCount),
			LastMessage:  result.LastMessage,
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
