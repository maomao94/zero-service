package logic

import (
	"context"

	"zero-service/app/ispagent/internal/svc"
	"zero-service/app/ispagent/ispagent"
	"zero-service/common/isp"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendPatrolDeviceRunDataLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendPatrolDeviceRunDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendPatrolDeviceRunDataLogic {
	return &SendPatrolDeviceRunDataLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendPatrolDeviceRunDataLogic) SendPatrolDeviceRunData(in *ispagent.SendPatrolDeviceRunDataReq) (*ispagent.CommandRes, error) {
	msg, err := l.svcCtx.IspClient.Execute(l.ctx, isp.TypePatrolDeviceRunData, isp.CommandReport, in.GetCode(), patrolDeviceRunDataToItems(in.GetItems()))
	if err != nil {
		return nil, err
	}
	return commandResponse(msg), nil
}
