package logic

import (
	"context"

	"zero-service/app/ispagent/internal/ispclient"
	"zero-service/app/ispagent/internal/svc"
	"zero-service/app/ispagent/ispagent"

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

func (l *SendPatrolDeviceRunDataLogic) SendPatrolDeviceRunData(in *ispagent.SendPatrolDeviceRunDataReq) (*ispagent.SendPatrolDeviceRunDataRes, error) {
	if err := l.svcCtx.IspClient.CacheReport(l.ctx, ispclient.ReportCategoryPatrolDeviceRunData, in.GetCode(), patrolDeviceRunDataToItems(in.GetItems())); err != nil {
		return nil, err
	}
	return &ispagent.SendPatrolDeviceRunDataRes{}, nil
}
