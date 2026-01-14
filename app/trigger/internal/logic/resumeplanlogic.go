package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
)

type ResumePlanLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResumePlanLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResumePlanLogic {
	return &ResumePlanLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 恢复计划
func (l *ResumePlanLogic) ResumePlan(in *trigger.ResumePlanReq) (*trigger.ResumePlanRes, error) {
	// todo: add your logic here and delete this line

	return &trigger.ResumePlanRes{}, nil
}
