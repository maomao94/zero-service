package logic

import (
	"context"

	"zero-service/app/ispagent/internal/ispclient"
	"zero-service/app/ispagent/internal/svc"
	"zero-service/app/ispagent/ispagent"

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

func (l *SendPatrolDeviceCoordinatesLogic) SendPatrolDeviceCoordinates(in *ispagent.SendPatrolDeviceCoordinatesReq) (*ispagent.SendPatrolDeviceCoordinatesRes, error) {
	if err := l.svcCtx.IspClient.CacheReport(l.ctx, ispclient.ReportCategoryPatrolDeviceCoordinates, in.GetCode(), patrolDeviceCoordinatesToItems(in.GetItems())); err != nil {
		return nil, err
	}
	return &ispagent.SendPatrolDeviceCoordinatesRes{}, nil
}
