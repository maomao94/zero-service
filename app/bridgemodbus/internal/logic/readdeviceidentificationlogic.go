package logic

import (
	"context"
	"fmt"
	"zero-service/common/modbusx"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/grid-x/modbus"
	"github.com/zeromicro/go-zero/core/logx"
)

type ReadDeviceIdentificationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReadDeviceIdentificationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadDeviceIdentificationLogic {
	return &ReadDeviceIdentificationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 读取设备标识 (Function Code 0x2B / 0x0E)
func (l *ReadDeviceIdentificationLogic) ReadDeviceIdentification(in *bridgemodbus.ReadDeviceIdentificationReq) (*bridgemodbus.ReadDeviceIdentificationRes, error) {
	mbCli := l.svcCtx.ModbusClientPool.Get()
	defer l.svcCtx.ModbusClientPool.Put(mbCli)
	results, err := mbCli.ReadDeviceIdentification(l.ctx, modbus.ReadDeviceIDCode(in.ReadDeviceIdCode))
	if err != nil {
		return nil, err
	}
	resultsDec := make(map[uint32]string)      // 十进制
	resultsHex := make(map[string]string)      // 十六进制
	resultsSemantic := make(map[string]string) // 语义化
	for id, raw := range results {
		val := string(raw)

		// 1. 十进制
		resultsDec[uint32(id)] = val

		// 2. 十六进制
		hexKey := fmt.Sprintf("0x%02X", id)
		resultsHex[hexKey] = val

		// 3. 语义化
		if name, ok := modbusx.DeviceIDObjectNames[id]; ok {
			resultsSemantic[name] = val
		} else {
			resultsSemantic["Object_"+hexKey] = val // fallback
		}
	}
	return &bridgemodbus.ReadDeviceIdentificationRes{
		Results:         resultsDec,
		HexResults:      resultsHex,
		SemanticResults: resultsSemantic,
	}, nil
}
