package djisdk

import (
	"errors"
	"fmt"

	"zero-service/third_party/dji_error_code"
)

// DJIError 大疆设备返回的业务错误，包含原始错误码、枚举名称和中文描述。
type DJIError struct {
	Code    int
	Name    string
	Message string
}

func (e *DJIError) Error() string {
	return fmt.Sprintf("[dji-sdk] device error: code=%d name=%s message=%s", e.Code, e.Name, e.Message)
}

// NewDJIError 根据设备返回的 result code 构造结构化错误。
func NewDJIError(code int) *DJIError {
	name, ok := dji_error_code.DJIErrorCode_name[int32(code)]
	if !ok {
		name = fmt.Sprintf("UNKNOWN(%d)", code)
	}
	desc := djiErrorDescriptions[int32(code)]
	if desc == "" {
		desc = name
	}
	return &DJIError{
		Code:    code,
		Name:    name,
		Message: desc,
	}
}

// IsDJIError 判断 err 是否为 DJIError，是则返回解包后的 DJIError 指针。
func IsDJIError(err error) (*DJIError, bool) {
	var djiErr *DJIError
	if errors.As(err, &djiErr) {
		return djiErr, true
	}
	return nil, false
}

// PlatformError 携带 reply result 码的业务错误，供 on* handler 在 need_reply=1 时精确控制 events_reply/status_reply 的 data.result。
// Code 为 PlatformResult 枚举值；Err 为底层错误信息。
// 未包装为 PlatformError 的普通 error 默认按 PlatformResultHandlerError 回复。
type PlatformError struct {
	Code PlatformResult
	Err  error
}

func (e *PlatformError) Error() string {
	return fmt.Sprintf("[dji-sdk] platform error: code=%d err=%v", e.Code, e.Err)
}

func (e *PlatformError) Unwrap() error { return e.Err }
