package logic

import (
	"context"

	ispclient "zero-service/app/ispagent/internal/isp"
	"zero-service/app/ispagent/internal/svc"
	"zero-service/app/ispagent/ispagent"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendDroneNestRunDataLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendDroneNestRunDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendDroneNestRunDataLogic {
	return &SendDroneNestRunDataLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendDroneNestRunDataLogic) SendDroneNestRunData(in *ispagent.SendDroneNestRunDataReq) (*ispagent.SendDroneNestRunDataRes, error) {
	if err := l.svcCtx.IspClient.CacheReport(l.ctx, ispclient.ReportCategoryDroneNestRunData, in.GetCode(), droneNestRunDataToItems(in.GetItems())); err != nil {
		return nil, err
	}
	return &ispagent.SendDroneNestRunDataRes{}, nil
}
