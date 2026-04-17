package logic

import (
	"context"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/config"
	"zero-service/aiapp/aisolo/internal/svc"
	"zero-service/common/skillmd"
)

type ListSkillsLogic struct {
	svcCtx *svc.ServiceContext
}

func NewListSkillsLogic(_ context.Context, svcCtx *svc.ServiceContext) *ListSkillsLogic {
	return &ListSkillsLogic{svcCtx: svcCtx}
}

func (l *ListSkillsLogic) ListSkills(*aisolo.ListSkillsReq) (*aisolo.ListSkillsResp, error) {
	dir := config.EffectiveSkillsDir(l.svcCtx.Config.Skills)
	if dir == "" {
		return &aisolo.ListSkillsResp{}, nil
	}
	infos, err := skillmd.ScanDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]*aisolo.SkillInfo, 0, len(infos))
	for _, s := range infos {
		out = append(out, &aisolo.SkillInfo{
			Id:           s.ID,
			Name:         s.Name,
			Description:  s.Description,
			Tags:         s.Tags,
			LaunchPrompt: s.LaunchPrompt,
		})
	}
	return &aisolo.ListSkillsResp{Skills: out}, nil
}
