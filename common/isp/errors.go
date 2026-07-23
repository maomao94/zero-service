package isp

import "errors"

// IspError 表示写入 ISP 通用应答 Code 字段的协议错误。
type IspError struct {
	Code string
	Msg  string
}

func (e *IspError) Error() string { return e.Msg }

var (
	// ISP 通用应答错误。
	ErrRetry         = &IspError{Code: StatusRetry, Msg: "需重发"}
	ErrReject        = &IspError{Code: StatusReject, Msg: "拒绝"}
	ErrInternal      = &IspError{Code: StatusError, Msg: "内部错误"}
	ErrUnimplemented = &IspError{Code: StatusError, Msg: "该指令暂未实现"}

	// ISP 客户端本地运行错误，不携带任何 gRPC 语义。
	ErrInvalidMessageType  = errors.New("isp: invalid message type")
	ErrClientNotRegistered = errors.New("isp: client not registered")
	ErrSessionUnavailable  = errors.New("isp: session unavailable")
	ErrRequestFailed       = errors.New("isp: request failed")
	ErrUnexpectedResponse  = errors.New("isp: unexpected response type")
)

func IsUnimplemented(err error) bool {
	return errors.Is(err, ErrUnimplemented)
}

func NewIspError(code, msg string) *IspError {
	return &IspError{Code: code, Msg: msg}
}

// ResponseCode 根据 error 提取 ISP 响应状态码。
func ResponseCode(err error) string {
	var ispErr *IspError
	if errors.As(err, &ispErr) {
		return ispErr.Code
	}
	if err != nil {
		return StatusError
	}
	return StatusSuccess
}
