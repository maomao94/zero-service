package solo

import (
	"context"
	"zero-service/aiapp/aisolo/aisolo"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ResumeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewResumeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResumeLogic {
	return &ResumeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ResumeLogic) Resume(req *types.SoloInterruptRequest) (resp *types.SoloInterruptResponse, err error) {
	var action aisolo.ResumeAction
	if req.Approved {
		action = aisolo.ResumeAction_RESUME_ACTION_APPROVE
	} else {
		action = aisolo.ResumeAction_RESUME_ACTION_DENY
	}
	protoReq := &aisolo.ResumeReq{
		InterruptId: req.InterruptId,
		Action:      action,
	}

	_, err = l.svcCtx.EinoCli.Resume(l.ctx, protoReq)
	if err != nil {
		l.Logger.Errorf("resume failed: %v", err)
		return nil, err
	}

	return &types.SoloInterruptResponse{
		Success:     true,
		Message:     "Resume request processed",
		InterruptId: req.InterruptId,
	}, nil
}
