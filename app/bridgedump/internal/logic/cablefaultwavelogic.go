package logic

import (
	"context"
	"zero-service/app/bridgedump/bridgedump"
	"zero-service/app/bridgedump/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

var cableFaultWaveDataFile = "cable_fault_wave"

type CableFaultWaveLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCableFaultWaveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CableFaultWaveLogic {
	return &CableFaultWaveLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 电缆故障波形数据接入
func (l *CableFaultWaveLogic) CableFaultWave(in *bridgedump.CableFaultWaveReq) (*bridgedump.CableFaultWaveRes, error) {
	_, err := l.svcCtx.DumpBridgeData(l.ctx, l.svcCtx.Config.DumpPath, cableFaultWaveDataFile, in)
	if err != nil {
		return nil, err
	}
	return &bridgedump.CableFaultWaveRes{
		Code: 200,
		Msg:  "成功",
	}, nil
}
