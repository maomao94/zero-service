package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendTriggerLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendTriggerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendTriggerLogic {
	return &SendTriggerLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendTriggerLogic) SendTrigger(in *trigger.SendTriggerReq) (*trigger.SendTriggerRes, error) {
	// todo: add your logic here and delete this line

	return &trigger.SendTriggerRes{}, nil
}
