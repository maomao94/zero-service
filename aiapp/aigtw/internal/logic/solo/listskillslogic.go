package solo

import (
	"context"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/aiapp/aisolo/aisolo"
)

type ListSkillsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListSkillsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSkillsLogic {
	return &ListSkillsLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *ListSkillsLogic) ListSkills() (*types.SoloListSkillsResponse, error) {
	resp, err := l.svcCtx.AiSoloCli.ListSkills(l.ctx, &aisolo.ListSkillsReq{})
	if err != nil {
		return nil, err
	}
	out := &types.SoloListSkillsResponse{Skills: make([]*types.SoloSkillInfo, 0, len(resp.Skills))}
	for _, s := range resp.Skills {
		out.Skills = append(out.Skills, &types.SoloSkillInfo{
			Id:           s.GetId(),
			Name:         s.GetName(),
			Description:  s.GetDescription(),
			Tags:         s.GetTags(),
			LaunchPrompt: s.GetLaunchPrompt(),
		})
	}
	return out, nil
}
