package solo

import (
	"context"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/aiapp/aisolo/aisolo"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListModesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListModesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListModesLogic {
	return &ListModesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ListModes 公开接口, 无需 JWT userId.
func (l *ListModesLogic) ListModes() (*types.SoloListModesResponse, error) {
	resp, err := l.svcCtx.AiSoloCli.ListModes(l.ctx, &aisolo.ListModesReq{})
	if err != nil {
		return nil, err
	}
	out := &types.SoloListModesResponse{}
	for _, m := range resp.GetModes() {
		out.Modes = append(out.Modes, modeToType(m))
	}
	return out, nil
}
