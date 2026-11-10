package errx

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	xerrors "github.com/zeromicro/x/errors"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewErrorByPbCode 用 extproto 平台码构造业务错误。
//
// 错误对象满足 gRPC status 接口：六位码写入 ErrorInfo.reason，gRPC Code 由
// extproto 的 http_code 推导，message 为可安全展示的文案。
// 0 是唯一成功码，传入 0 或未登记的码时统一降级为 100001，避免把失败当成功。
func NewErrorByPbCode(code extproto.Code, args ...any) error {
	if code == extproto.Code__0_OK {
		return &Error{
			code:    int(extproto.Code__1_00_INTERNAL),
			message: "成功码不能用于构造错误",
		}
	}
	if !IsDefinedCode(int(code)) {
		logx.Errorf("非法错误码 %d 不能用于构造错误，已降级为 %d",
			int(code), int(extproto.Code__1_00_INTERNAL))
		return &Error{
			code:    int(extproto.Code__1_00_INTERNAL),
			message: "非法错误码不能用于构造错误",
		}
	}
	return &Error{
		code:    int(code),
		message: formatErrorMessage(ErrorMessage(int(code)), args),
	}
}

// NewErrorByPbCodeWrap 在平台错误上附加 cause。
//
// cause 用于服务端日志与 errors.Is/As 判断，不进入 gRPC status message，
// 避免数据库、连接串等内部细节泄漏给调用方。
//
// 若 cause 已经携带已登记的 extproto 码，则保留其码值：gRPC status 原样返回，
// CodeMsg 转成 errx.Error，禁止中间层重包覆盖，保证 RPC → RPC 链路码值全程保真。
func NewErrorByPbCodeWrap(code extproto.Code, cause error, args ...any) error {
	if cause == nil {
		return NewErrorByPbCode(code, args...)
	}

	// 已经实现 gRPC status 且携带合法平台码：原样透传，避免中间层覆盖码值。
	if _, ok := status.FromError(cause); ok {
		if _, ok := platformCodeOf(cause); ok {
			return cause
		}
	}

	// CodeMsg 不是 gRPC status；转成 errx.Error，保证 detail 和 GRPCStatus 都不丢。
	var codeMsg *xerrors.CodeMsg
	if errors.As(cause, &codeMsg) && codeMsg != nil &&
		IsDefinedCode(codeMsg.Code) && codeMsg.Code != SuccessCode {
		return &Error{
			code:    codeMsg.Code,
			message: codeMsg.Msg,
		}
	}

	base := NewErrorByPbCode(code, args...).(*Error)
	return &Error{
		code:    base.code,
		message: base.message,
		cause:   cause,
	}
}

// IsErrorByPbCode 判断错误是否为指定平台码。
//
// 同时支持 errx 构造的错误、gRPC status（含 ErrorInfo.reason）与
// zeromicro/x/errors.CodeMsg，包装链上的错误也能识别。
func IsErrorByPbCode(err error, code extproto.Code) bool {
	if err == nil {
		return false
	}
	if got, ok := platformCodeOf(err); ok {
		return got == int(code)
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() == codes.OK {
		// 普通错误没有平台码，不能按 100002 兜底匹配。
		return false
	}
	return codeFromGRPCStatus(st.Code()) == int(code)
}

// platformCodeOf 返回错误显式携带的 extproto 码。
//
// 只认 errx.Error、zeromicro/x/errors.CodeMsg 与 ErrorInfo.reason，
// 不按 gRPC Code 兜底，避免把普通 status 误判成“已带平台码”。
func platformCodeOf(err error) (int, bool) {
	if err == nil {
		return 0, false
	}

	// 先看 errx 构造的错误，避免被 cause 上的码反向覆盖。
	var platformErr *Error
	if errors.As(err, &platformErr) {
		if IsDefinedCode(platformErr.code) && platformErr.code != SuccessCode {
			return platformErr.code, true
		}
		return 0, false
	}

	var codeMsg *xerrors.CodeMsg
	if errors.As(err, &codeMsg) && codeMsg != nil {
		if IsDefinedCode(codeMsg.Code) && codeMsg.Code != SuccessCode {
			return codeMsg.Code, true
		}
		return 0, false
	}

	st, ok := status.FromError(err)
	if !ok {
		return 0, false
	}
	for _, detail := range st.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok {
			continue
		}
		reason, convErr := strconv.Atoi(strings.TrimSpace(info.GetReason()))
		if convErr != nil || !IsDefinedCode(reason) || reason == SuccessCode {
			continue
		}
		return reason, true
	}
	return 0, false
}

// Error 是 errx 构造的平台业务错误。
type Error struct {
	code    int
	message string
	cause   error
}

// Error 返回便于日志排查的描述；cause 只在这里出现，不进入 gRPC 响应。
func (e *Error) Error() string {
	if e.cause == nil {
		return e.message
	}
	return fmt.Sprintf("%s: %v", e.message, e.cause)
}

// Unwrap 暴露 cause，供 errors.Is/errors.As 使用。
func (e *Error) Unwrap() error {
	return e.cause
}

// Code 返回 extproto 六位错误码。
func (e *Error) Code() int {
	return e.code
}

// Message 返回可对外展示的错误文案。
func (e *Error) Message() string {
	return e.message
}

// GRPCStatus 把平台错误转换为 gRPC status。
func (e *Error) GRPCStatus() *status.Status {
	return newExtprotoStatus(
		grpcCodeFromHTTPStatus(semanticHTTPCode(e.code)),
		e.message,
		e.code,
	)
}

// formatErrorMessage 保持历史实现的参数格式化语义：
// 优先使用码表默认文案的占位符，其次按首个参数决定自定义文案。
func formatErrorMessage(message string, args []any) string {
	if len(args) == 0 {
		return message
	}
	if hasFormatPlaceholder(message) {
		return fmt.Sprintf(message, args...)
	}

	switch v := args[0].(type) {
	case string:
		if len(args) > 1 && hasFormatPlaceholder(v) {
			return fmt.Sprintf(v, args[1:]...)
		}
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func hasFormatPlaceholder(message string) bool {
	for i := 0; i < len(message); i++ {
		if message[i] == '%' && i+1 < len(message) {
			return true
		}
	}
	return false
}
