package solo

import (
	"context"
	"zero-service/aiapp/aisolo/aisolo"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAgentsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListAgentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAgentsLogic {
	return &ListAgentsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListAgentsLogic) ListAgents() (resp *types.SoloListAgentsResponse, err error) {
	protoReq := &aisolo.ListAgentsReq{}

	result, err := l.svcCtx.AiSoloCli.ListAgents(l.ctx, protoReq)
	if err != nil {
		l.Logger.Errorf("list agents failed: %v", err)
		return nil, err
	}

	agents := make([]*types.SoloAgentInfo, len(result.Agents))
	for i, a := range result.Agents {
		agents[i] = &types.SoloAgentInfo{
			Type:         a.Id,
			Name:         a.Name,
			Description:  a.Description,
			Capabilities: a.Capabilities,
			Available:    a.Available,
		}
	}

	return &types.SoloListAgentsResponse{Agents: agents}, nil
}
