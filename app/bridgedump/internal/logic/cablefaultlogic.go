package logic

import (
	"context"
	"zero-service/app/bridgedump/bridgedump"
	"zero-service/app/bridgedump/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

var cableFaultDataFile = "cable_fault"

type CableFaultLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCableFaultLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CableFaultLogic {
	return &CableFaultLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 电缆故障结果数据接入
func (l *CableFaultLogic) CableFault(in *bridgedump.CableFaultReq) (*bridgedump.CableFaultRes, error) {
	_, err := l.svcCtx.DumpBridgeData(l.ctx, l.svcCtx.Config.DumpPath, cableFaultDataFile, in)
	if err != nil {
		return nil, err
	}
	return &bridgedump.CableFaultRes{
		Code: 200,
		Msg:  "成功",
	}, nil
}
