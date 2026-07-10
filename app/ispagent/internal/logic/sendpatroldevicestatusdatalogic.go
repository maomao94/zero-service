package logic

import (
	"context"

	ispclient "zero-service/app/ispagent/internal/isp"
	"zero-service/app/ispagent/internal/svc"
	"zero-service/app/ispagent/ispagent"
	"zero-service/common/isp"

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

func (l *SendPatrolDeviceStatusDataLogic) SendPatrolDeviceStatusData(in *ispagent.SendPatrolDeviceStatusDataReq) (*ispagent.CommandRes, error) {
	if err := l.svcCtx.IspClient.CacheReport(l.ctx, ispclient.ReportCategoryPatrolDeviceStatusData, in.GetCode(), patrolDeviceStatusDataToItems(in.GetItems())); err != nil {
		return nil, err
	}
	return &ispagent.CommandRes{Success: true, Code: isp.StatusSuccess}, nil
}
