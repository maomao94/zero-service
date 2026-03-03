package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetExecItemDashboardLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetExecItemDashboardLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetExecItemDashboardLogic {
	return &GetExecItemDashboardLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取执行项仪表板统计信息
func (l *GetExecItemDashboardLogic) GetExecItemDashboard(in *trigger.GetExecItemDashboardReq) (*trigger.GetExecItemDashboardRes, error) {
	// todo: add your logic here and delete this line

	return &trigger.GetExecItemDashboardRes{}, nil
}
