package logic

import (
	"context"
	"zero-service/common/bytex"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type WriteMultipleCoilsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewWriteMultipleCoilsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WriteMultipleCoilsLogic {
	return &WriteMultipleCoilsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 写多个线圈 (Function Code 0x0F)
func (l *WriteMultipleCoilsLogic) WriteMultipleCoils(in *bridgemodbus.WriteMultipleCoilsReq) (*bridgemodbus.WriteMultipleCoilsRes, error) {
	if int(in.Quantity) != len(in.Values) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "数量与值数量不一致")
	}

	mdCliPool, err := l.svcCtx.GetModbusClientPool(l.ctx, in.ModbusCode)
	if err != nil {
		return nil, err
	}
	mbCli := mdCliPool.Get()
	defer mdCliPool.Put(mbCli)

	binaryValues := bytex.BoolsToBitValues(in.Values)
	l.Infof("写多个线圈: 0x%X, bools=%v, binary=%v", binaryValues.Bytes, binaryValues.Bools, binaryValues.Binary)
	results, err := mbCli.WriteMultipleCoils(l.ctx, uint16(in.Address), uint16(in.Quantity), binaryValues.Bytes)
	if err != nil {
		return nil, err
	}
	return &bridgemodbus.WriteMultipleCoilsRes{
		Results: results,
	}, nil
}
