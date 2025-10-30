package logic

import (
	"context"
	"errors"
	"fmt"
	"zero-service/common/modbusx"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadDeviceIdentificationSpecificObjectLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReadDeviceIdentificationSpecificObjectLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadDeviceIdentificationSpecificObjectLogic {
	return &ReadDeviceIdentificationSpecificObjectLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 读取特定 Object ID 的设备标识 (Function Code 0x2B / 0x0E)
func (l *ReadDeviceIdentificationSpecificObjectLogic) ReadDeviceIdentificationSpecificObject(in *bridgemodbus.ReadDeviceIdentificationSpecificObjectReq) (*bridgemodbus.ReadDeviceIdentificationSpecificObjectRes, error) {
	var mdCliPool *modbusx.ModbusClientPool
	var err error
	if len(in.ModbusCode) == 0 {
		mdCliPool = l.svcCtx.ModbusClientPool
	} else {
		var ok bool
		mdCliPool, ok = l.svcCtx.Manager.GetPool(in.ModbusCode) // 关键：用=而不是:=，避免局部变量
		if !ok {
			mdCliPool, err = l.svcCtx.AddPool(l.ctx, in.ModbusCode)
			if err != nil {
				return nil, fmt.Errorf("创建Modbus连接池失败: %w", err)
			}
		}
		if mdCliPool == nil {
			return nil, errors.New("获取的Modbus连接池为空")
		}
	}
	mbCli := mdCliPool.Get()
	defer mdCliPool.Put(mbCli)
	results, err := mbCli.ReadDeviceIdentificationSpecificObject(l.ctx, byte(in.ObjectId))
	if err != nil {
		return nil, err
	}
	resultsDec := make(map[uint32]string)      // 十进制
	resultsHex := make(map[string]string)      // 十六进制
	resultsSemantic := make(map[string]string) // 语义化
	for id, raw := range results {
		val := string(raw)
		if len(raw) == 0 {
			continue
		}

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
	return &bridgemodbus.ReadDeviceIdentificationSpecificObjectRes{
		Results:         resultsDec,
		HexResults:      resultsHex,
		SemanticResults: resultsSemantic,
	}, nil
}
