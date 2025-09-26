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

type ReadInputRegistersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReadInputRegistersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadInputRegistersLogic {
	return &ReadInputRegistersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 读取输入寄存器 (Function Code 0x04)
func (l *ReadInputRegistersLogic) ReadInputRegisters(in *bridgemodbus.ReadInputRegistersReq) (*bridgemodbus.ReadInputRegistersRes, error) {
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
	results, err := mbCli.ReadInputRegisters(l.ctx, uint16(in.Address), uint16(in.Quantity))
	if err != nil {
		return nil, err
	}

	// 每个寄存器占 2 个字节，所以寄存器数量 = bytes 长度 / 2
	n := len(results) / 2

	// 创建切片用于存放每个寄存器的 16 进制字符串表示
	//hexValues := make([]string, n)

	// 循环处理每个寄存器
	//for i := 0; i < n; i++ {
	//	hi := results[2*i]   // 当前寄存器的高字节 (第 2*i 个字节)
	//	lo := results[2*i+1] // 当前寄存器的低字节 (第 2*i+1 个字节)
	//
	//	// 把高字节左移 8 位，再和低字节按位或，得到完整的 16 位寄存器值
	//	val := uint16(hi)<<8 | uint16(lo)
	//
	//	// 格式化为 16 进制字符串，例如 "0x1234"
	//	hexValues[i] = fmt.Sprintf("0x%04X", val)
	//}

	// 第二种解析 方法
	hexValues := make([]string, 0, n)
	for i := 0; i < len(results); i += 2 {
		hi := results[i]   // 高字节
		lo := results[i+1] // 低字节

		val := uint16(hi)<<8 | uint16(lo)                         // 拼成 16 位寄存器值
		hexValues = append(hexValues, fmt.Sprintf("0x%04X", val)) // 转 16 进制字符串
	}
	return &bridgemodbus.ReadInputRegistersRes{
		Results: results,
		Values:  hexValues,
	}, nil
}
