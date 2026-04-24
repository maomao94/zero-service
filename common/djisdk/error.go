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
