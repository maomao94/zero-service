package logic

import (
	"context"

	"zero-service/app/ispagent/internal/ispclient"
	"zero-service/app/ispagent/internal/svc"
	"zero-service/app/ispagent/ispagent"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendPatrolDeviceStatusDataLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendPatrolDeviceStatusDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendPatrolDeviceStatusDataLogic {
	return &SendPatrolDeviceStatusDataLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendPatrolDeviceStatusDataLogic) SendPatrolDeviceStatusData(in *ispagent.SendPatrolDeviceStatusDataReq) (*ispagent.SendPatrolDeviceStatusDataRes, error) {
	if err := l.svcCtx.IspClient.CacheReport(l.ctx, ispclient.ReportCategoryPatrolDeviceStatusData, in.GetCode(), patrolDeviceStatusDataToItems(in.GetItems())); err != nil {
		return nil, err
	}
	return &ispagent.SendPatrolDeviceStatusDataRes{}, nil
}
