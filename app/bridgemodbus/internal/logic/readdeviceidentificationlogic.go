package logic

import (
	"context"
	"fmt"
	"zero-service/common/modbusx"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

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
	// 只接受 0x01 基本 / 0x02 常规 / 0x03 扩展；
	// 0x04 指定对象必须调用 ReadDeviceIdentificationSpecificObject。
	code, err := validateReadDeviceIDCode(in.ReadDeviceIdCode)
	if err != nil {
		return nil, err
	}

	mbCli, err := l.svcCtx.GetModbusClient(l.ctx, in.ModbusCode)
	if err != nil {
		return nil, err
	}

	results, err := mbCli.ReadDeviceIdentification(l.ctx, code)
	if err != nil {
		return nil, wrapModbusError("读取设备标识", err)
	}
	return &bridgemodbus.ReadDeviceIdentificationRes{
		Results:         toDeviceIDDec(results),
		HexResults:      toDeviceIDHex(results),
		SemanticResults: toDeviceIDSemantic(results),
	}, nil
}

// toDeviceIDDec 把对象 ID → 原始值映射转为十进制 key 的响应 map。
func toDeviceIDDec(results map[byte][]byte) map[uint32]string {
	converted := make(map[uint32]string, len(results))
	for id, raw := range results {
		if len(raw) == 0 {
			continue
		}
		converted[uint32(id)] = string(raw)
	}
	return converted
}

// toDeviceIDHex 把对象 ID → 原始值映射转为十六进制 key 的响应 map。
func toDeviceIDHex(results map[byte][]byte) map[string]string {
	converted := make(map[string]string, len(results))
	for id, raw := range results {
		if len(raw) == 0 {
			continue
		}
		converted[fmt.Sprintf("0x%02X", id)] = string(raw)
	}
	return converted
}

// toDeviceIDSemantic 把对象 ID 映射为 Modbus 标准语义名称，未知对象回退为 Object_0xXX。
func toDeviceIDSemantic(results map[byte][]byte) map[string]string {
	converted := make(map[string]string, len(results))
	for id, raw := range results {
		if len(raw) == 0 {
			continue
		}
		if name, ok := modbusx.DeviceIDObjectName(id); ok {
			converted[name] = string(raw)
			continue
		}
		converted["Object_"+fmt.Sprintf("0x%02X", id)] = string(raw)
	}
	return converted
}
