package logic

import (
	"context"

	"zero-service/app/ispagent/internal/svc"
	"zero-service/app/ispagent/ispagent"
	"zero-service/common/isp"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendPatrolDeviceCoordinatesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendPatrolDeviceCoordinatesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendPatrolDeviceCoordinatesLogic {
	return &SendPatrolDeviceCoordinatesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendPatrolDeviceCoordinatesLogic) SendPatrolDeviceCoordinates(in *ispagent.SendPatrolDeviceCoordinatesReq) (*ispagent.CommandRes, error) {
	msg, err := l.svcCtx.IspClient.Execute(l.ctx, isp.TypePatrolDeviceCoordinates, isp.CommandReport, in.GetCode(), patrolDeviceCoordinatesToItems(in.GetItems()))
	if err != nil {
		return nil, err
	}
	return commandResponse(msg), nil
}
