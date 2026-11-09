package modbusx

import (
	"context"
	"errors"
	"net"

	"github.com/grid-x/modbus"
)

// exceptionMessages 是 Modbus 标准异常码的中文描述。
var exceptionMessages = map[byte]string{
	modbus.ExceptionCodeIllegalFunction:                    "非法功能码",
	modbus.ExceptionCodeIllegalDataAddress:                 "非法数据地址",
	modbus.ExceptionCodeIllegalDataValue:                   "非法数据值",
	modbus.ExceptionCodeServerDeviceFailure:                "从站设备故障",
	modbus.ExceptionCodeAcknowledge:                        "从站已确认，操作处理中",
	modbus.ExceptionCodeServerDeviceBusy:                   "从站设备忙",
	modbus.ExceptionCodeMemoryParityError:                  "存储奇偶校验错误",
	modbus.ExceptionCodeGatewayPathUnavailable:             "网关路径不可用",
	modbus.ExceptionCodeGatewayTargetDeviceFailedToRespond: "网关目标设备无响应",
}

// ExceptionMessage 返回异常码的中文描述，未知码返回“未知异常”。
func ExceptionMessage(code byte) string {
	if msg, ok := exceptionMessages[code]; ok {
		return msg
	}
	return "未知异常"
}

// Exception 判断错误是否为设备返回的 Modbus 异常响应，并提取功能码与异常码。
func Exception(err error) (functionCode, exceptionCode byte, ok bool) {
	var mbErr *modbus.Error
	// mbErr 为零值指针时不能解引用：errors.As 对 typed nil 也会返回 true
	if errors.As(err, &mbErr) && mbErr != nil {
		return mbErr.FunctionCode, mbErr.ExceptionCode, true
	}
	return 0, 0, false
}

// IsTimeout 判断错误是否为读写或建连超时。
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// IsStreamDesynced 判断错误是否为响应流与请求失配（grid-x v1.5.1 起）。
// 该状态下连接已被 SDK 关闭，重试需要调用方重新发起请求。
func IsStreamDesynced(err error) bool {
	return errors.Is(err, modbus.ErrStreamDesynced)
}
