package errx

import (
	"errors"
	"strconv"

	"zero-service/third_party/extproto"

	xerrors "github.com/zeromicro/x/errors"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToGRPCStatus 是 gRPC 服务端出口的最后一道归一化。
//
// 它只兜底“没有走统一错误入口”的错误，不重写调用方显式构造的 status：
//   - context 取消/超时交给 status.FromContextError，保留 gRPC 语义；
//   - errx 错误直接取内层 status，避免 fmt.Errorf 包装后 grpc-go 用
//     err.Error() 覆盖 message 把 cause 带出去；
//   - 没有 ErrorInfo 的 status 补一条 extproto 兜底 detail，保证调用方始终
//     能按六位码处理；
//   - 其余裸错误统一包装为平台错误，对外返回 100999 + 系统繁忙。
func ToGRPCStatus(err error) error {
	if err == nil {
		return nil
	}

	// 平台错误优先于 context：业务码包装了 context 取消/超时时，
	// 必须保留业务码，不能被 context 语义覆盖。
	var platformErr *Error
	if errors.As(err, &platformErr) {
		return withExtprotoFallbackDetail(platformErr.GRPCStatus()).Err()
	}

	// CodeMsg 不是 gRPC status，直接返回会让 gRPC 按 Unknown 透出原始文案；
	// 这里转成 errx.Error，把业务码放进 detail。
	var codeMsg *xerrors.CodeMsg
	if errors.As(err, &codeMsg) && codeMsg != nil {
		if IsDefinedCode(codeMsg.Code) && codeMsg.Code != SuccessCode {
			return &Error{code: codeMsg.Code, message: codeMsg.Msg}
		}
		return &Error{
			code:    int(extproto.Code__1_00_UNKNOWN),
			message: SystemBusyMessage,
			cause:   err,
		}
	}

	// 已实现 GRPCStatus 的错误取内层 status，避免 fmt.Errorf 包装后
	// grpc-go 用 err.Error() 覆盖 message 把包装文本带出去。
	var gs interface{ GRPCStatus() *status.Status }
	if errors.As(err, &gs) && gs.GRPCStatus() != nil {
		return withExtprotoFallbackDetail(gs.GRPCStatus()).Err()
	}

	if st := status.FromContextError(err); st.Code() != codes.Unknown {
		return withExtprotoFallbackDetail(st).Err()
	}

	if st, ok := status.FromError(err); ok {
		return withExtprotoFallbackDetail(st).Err()
	}

	// 未转换的裸错误统一走平台错误构造：对外只保留系统繁忙文案，
	// cause 仅用于服务端日志和 errors.Is/As。
	return NewErrorByPbCodeWrap(extproto.Code__1_00_UNKNOWN, err, SystemBusyMessage)
}

// SystemBusyError 返回带 100999 extproto detail 的兜底平台错误。
// 供 panic 恢复等无法构造业务错误的出口使用，保证调用方仍能读到六位码。
func SystemBusyError() error {
	return NewErrorByPbCode(extproto.Code__1_00_UNKNOWN, SystemBusyMessage)
}

// withExtprotoFallbackDetail 给缺少 ErrorInfo 的 status 补一条 extproto detail。
//
// 没有 detail 的 status 一律补码，保证每种 error 最终都能解析到六位码；
// 已有 detail 时原样保留，非法码或厂商码仍由 CodeFromStatus 统一降级处理。
func withExtprotoFallbackDetail(st *status.Status) *status.Status {
	if st == nil || st.Code() == codes.OK || hasErrorInfoDetail(st) {
		return st
	}
	return newExtprotoStatus(st.Code(), st.Message(), codeFromGRPCStatus(st.Code()))
}

func hasErrorInfoDetail(st *status.Status) bool {
	for _, detail := range st.Details() {
		if _, ok := detail.(*errdetails.ErrorInfo); ok {
			return true
		}
	}
	return false
}

func newExtprotoStatus(code codes.Code, message string, extprotoCode int) *status.Status {
	st := status.New(code, message)
	withDetails, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason: strconv.Itoa(extprotoCode),
	})
	if err != nil {
		return st
	}
	return withDetails
}
