package logic

import (
	"context"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteConfigLogic {
	return &DeleteConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 删除配置（支持批量）
func (l *DeleteConfigLogic) DeleteConfig(in *bridgemodbus.DeleteConfigReq) (*bridgemodbus.DeleteConfigRes, error) {
	var deletedCount int
	for _, id := range in.Ids {
		err := l.svcCtx.ModbusSlaveConfigModel.Delete(l.ctx, nil, id)
		if err != nil {
			logx.Errorf("Delete failed, id=%d, err=%v", id, err)
			continue
		}
		deletedCount++
	}
	return &bridgemodbus.DeleteConfigRes{}, nil
}
