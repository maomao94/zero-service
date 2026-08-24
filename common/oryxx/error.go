package oryxx

import (
	"errors"
	"fmt"
)

// OryxError 表示 Oryx HTTP API 返回的业务错误（业务码 code != 0）。
type OryxError struct {
	// Oryx 业务错误码，0 表示成功
	Code int32
	// 错误信息，来自响应 data 字段（防御未来「200 + code != 0 + message」场景）
	Message string
}

func (e *OryxError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("oryx error: code=%d, message=%s", e.Code, e.Message)
	}
	return fmt.Sprintf("oryx error: code=%d", e.Code)
}

// IsOryxError 判断 err 是否为 Oryx 业务错误，是则返回解包后的 OryxError 指针。
func IsOryxError(err error) (*OryxError, bool) {
	var oe *OryxError
	if errors.As(err, &oe) {
		return oe, true
	}
	return nil, false
}
