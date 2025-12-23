package client

import "github.com/wendy512/go-iecp5/asdu"

// ASDUCall  is the interface of client handler
type ASDUCall interface {
	// OnInterrogation 总召唤回复
	OnInterrogation(*asdu.ASDU) error
	// OnCounterInterrogation 总计数器回复
	OnCounterInterrogation(*asdu.ASDU) error
	// OnRead 读定值回复
	OnRead(*asdu.ASDU) error
	// OnTestCommand 测试下发回复
	OnTestCommand(*asdu.ASDU) error
	// OnClockSync 时钟同步回复
	OnClockSync(*asdu.ASDU) error
	// OnResetProcess 进程重置回复
	OnResetProcess(*asdu.ASDU) error
	// OnDelayAcquisition 延迟获取回复
	OnDelayAcquisition(*asdu.ASDU) error
	// OnASDU 数据回复或控制回复
	OnASDU(*asdu.ASDU) error
}

// emptyASDUCall 是ASDUCall接口的空实现，用于默认初始化
// 这样可以避免空指针异常，同时允许用户通过WithASDUHandler覆盖

type emptyASDUCall struct{}

var _ ASDUCall = (*emptyASDUCall)(nil)

// OnInterrogation 空实现
func (e *emptyASDUCall) OnInterrogation(*asdu.ASDU) error {
	return nil
}

// OnCounterInterrogation 空实现
func (e *emptyASDUCall) OnCounterInterrogation(*asdu.ASDU) error {
	return nil
}

// OnRead 空实现
func (e *emptyASDUCall) OnRead(*asdu.ASDU) error {
	return nil
}

// OnTestCommand 空实现
func (e *emptyASDUCall) OnTestCommand(*asdu.ASDU) error {
	return nil
}

// OnClockSync 空实现
func (e *emptyASDUCall) OnClockSync(*asdu.ASDU) error {
	return nil
}

// OnResetProcess 空实现
func (e *emptyASDUCall) OnResetProcess(*asdu.ASDU) error {
	return nil
}

// OnDelayAcquisition 空实现
func (e *emptyASDUCall) OnDelayAcquisition(*asdu.ASDU) error {
	return nil
}

// OnASDU 空实现
func (e *emptyASDUCall) OnASDU(*asdu.ASDU) error {
	return nil
}
