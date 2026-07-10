package logic

import (
	"context"

	ispclient "zero-service/app/ispagent/internal/isp"
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
	if err := l.svcCtx.IspClient.CacheReport(l.ctx, ispclient.ReportCategoryPatrolDeviceRunData, in.GetCode(), patrolDeviceRunDataToItems(in.GetItems())); err != nil {
		return nil, err
	}
	return &ispagent.CommandRes{Success: true, Code: isp.StatusSuccess}, nil
}
