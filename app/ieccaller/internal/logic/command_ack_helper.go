package logic

import (
	"context"
	"errors"
	"fmt"

	"zero-service/common/antsx"
	"zero-service/common/iec104/client"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/wendy512/go-iecp5/asdu"
)

func wrapCommandAckError(err error, fallbackMsg string) error {
	switch {
	case errors.Is(err, antsx.ErrReplyExpired), errors.Is(err, context.DeadlineExceeded):
		return tool.NewErrorByPbCodeWrap(extproto.Code__1_00_TIMEOUT, err, "IEC控制命令ACK超时: %v", err)
	case errors.Is(err, antsx.ErrDuplicateID):
		return tool.NewErrorByPbCodeWrap(extproto.Code__1_05_BIZ_REPEAT, err, "IEC控制命令重复下发: %v", err)
	case isCommandRejectedError(err):
		return tool.NewErrorByPbCodeWrap(extproto.Code__1_05_BIZ_STATE, err, "IEC命令被设备拒绝: %v", err)
	default:
		return tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "%s: %v", fallbackMsg, err)
	}
}

func isCommandRejectedError(err error) bool {
	var rejected *client.CommandRejectedError
	return errors.As(err, &rejected)
}

func ackBoolValue(ackValue any) (bool, error) {
	value, ok := ackValue.(bool)
	if !ok {
		return false, unexpectedAckValueError("bool", ackValue)
	}
	return value, nil
}

func ackDoubleCommandValue(ackValue any) (asdu.DoubleCommand, error) {
	value, ok := ackValue.(asdu.DoubleCommand)
	if !ok {
		return 0, unexpectedAckValueError("asdu.DoubleCommand", ackValue)
	}
	return value, nil
}

func ackStepCommandValue(ackValue any) (asdu.StepCommand, error) {
	value, ok := ackValue.(asdu.StepCommand)
	if !ok {
		return 0, unexpectedAckValueError("asdu.StepCommand", ackValue)
	}
	return value, nil
}

func ackSetpointNormalizedValue(ackValue any) (asdu.Normalize, error) {
	value, ok := ackValue.(asdu.Normalize)
	if !ok {
		return 0, unexpectedAckValueError("asdu.Normalize", ackValue)
	}
	return value, nil
}

func ackInt16Value(ackValue any) (int16, error) {
	value, ok := ackValue.(int16)
	if !ok {
		return 0, unexpectedAckValueError("int16", ackValue)
	}
	return value, nil
}

func ackFloat32Value(ackValue any) (float32, error) {
	value, ok := ackValue.(float32)
	if !ok {
		return 0, unexpectedAckValueError("float32", ackValue)
	}
	return value, nil
}

func ackUint32Value(ackValue any) (uint32, error) {
	value, ok := ackValue.(uint32)
	if !ok {
		return 0, unexpectedAckValueError("uint32", ackValue)
	}
	return value, nil
}

func unexpectedAckValueError(expected string, actual any) error {
	return fmt.Errorf("unexpected IEC command ACK value type: expected %s, got %T", expected, actual)
}
