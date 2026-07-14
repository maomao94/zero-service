package logic

import (
	"context"

	ispclient "zero-service/app/ispagent/internal/isp"
	"zero-service/app/ispagent/internal/svc"
	"zero-service/app/ispagent/ispagent"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendEnvDataLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendEnvDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendEnvDataLogic {
	return &SendEnvDataLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendEnvDataLogic) SendEnvData(in *ispagent.SendEnvDataReq) (*ispagent.SendEnvDataRes, error) {
	if err := l.svcCtx.IspClient.CacheReport(l.ctx, ispclient.ReportCategoryEnvData, in.GetCode(), envDataToItems(in.GetItems())); err != nil {
		return nil, err
	}
	return &ispagent.SendEnvDataRes{}, nil
}
