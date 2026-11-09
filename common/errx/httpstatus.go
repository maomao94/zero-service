package errx

import (
	"net/http"

	"google.golang.org/grpc/codes"
)

// extproto http_code 与 gRPC Code 对照表。
//
// extproto 的 http_code 是唯一权威来源，NewErrorByPbCode 用它推导 gRPC Code。
// gRPC Code 始终是标准 gRPC code（0-16），extproto 六位业务码只写入
// ErrorInfo.reason，不写入 gRPC Code。
//
//	extproto http_code | gRPC Code         | 典型错误码
//	200                | OK                | 0
//	400                | InvalidArgument   | 100101/100102/100103/100501
//	401                | Unauthenticated   | 100401/130102/130105
//	403                | PermissionDenied  | 100402/130101
//	404                | NotFound          | 100202/100801/130103
//	405                | Unimplemented     | 100802
//	408                | Canceled          | 100996
//	409                | Aborted           | 100203/100204/100502/100503
//	429                | ResourceExhausted | 100701/130104
//	500                | Internal          | 100201/100301/100302/100998/100999
//	503                | Unavailable       | 100303/100601/100602/100603
//	504                | DeadlineExceeded  | 100997
//
// 本表是正向映射，保证每个 extproto 码的 http_code 都能推导出标准 gRPC Code。
// 反向的 grpcCodeToHTTPStatus 只用于分类，同一 HTTP 状态可能对应多个 gRPC Code
// （如 409），不能假定两者严格双向可逆。
var httpCodeToGRPCCode = map[int]codes.Code{
	http.StatusOK:                  codes.OK,
	http.StatusBadRequest:          codes.InvalidArgument,
	http.StatusUnauthorized:        codes.Unauthenticated,
	http.StatusForbidden:           codes.PermissionDenied,
	http.StatusNotFound:            codes.NotFound,
	http.StatusMethodNotAllowed:    codes.Unimplemented,
	http.StatusRequestTimeout:      codes.Canceled,
	http.StatusConflict:            codes.Aborted,
	http.StatusTooManyRequests:     codes.ResourceExhausted,
	http.StatusInternalServerError: codes.Internal,
	http.StatusServiceUnavailable:  codes.Unavailable,
	http.StatusGatewayTimeout:      codes.DeadlineExceeded,
}

// grpcCodeToHTTPStatus 是标准 gRPC Code 到语义 HTTP 状态的分类表。
//
// 两处按 extproto 口径收敛，避免 fallback detail 与业务码 http_code 错位：
//   - Unimplemented -> 405：对应 100802 请求方法不允许；
//   - FailedPrecondition -> 409：对应 100502 业务状态不允许。
//
// 同一 HTTP 状态可能对应多个 gRPC Code（如 409 对应 AlreadyExists/Aborted/
// FailedPrecondition），因此该表不是 httpCodeToGRPCCode 的严格反函数。
var grpcCodeToHTTPStatus = map[codes.Code]int{
	codes.OK:                 http.StatusOK,
	codes.InvalidArgument:    http.StatusBadRequest,
	codes.OutOfRange:         http.StatusBadRequest,
	codes.Unauthenticated:    http.StatusUnauthorized,
	codes.PermissionDenied:   http.StatusForbidden,
	codes.NotFound:           http.StatusNotFound,
	codes.Unimplemented:      http.StatusMethodNotAllowed,
	codes.Aborted:            http.StatusConflict,
	codes.AlreadyExists:      http.StatusConflict,
	codes.FailedPrecondition: http.StatusConflict,
	codes.ResourceExhausted:  http.StatusTooManyRequests,
	codes.Internal:           http.StatusInternalServerError,
	codes.Unknown:            http.StatusInternalServerError,
	codes.DataLoss:           http.StatusInternalServerError,
	codes.Unavailable:        http.StatusServiceUnavailable,
	codes.DeadlineExceeded:   http.StatusGatewayTimeout,
	codes.Canceled:           http.StatusRequestTimeout,
}

// grpcCodeFromHTTPStatus 按 extproto 的 http_code 推导 gRPC Code。
// 未登记的状态码统一按 Internal 处理，避免把未知语义透传为 codes.Unknown。
func grpcCodeFromHTTPStatus(httpCode int) codes.Code {
	if code, ok := httpCodeToGRPCCode[httpCode]; ok {
		return code
	}
	return codes.Internal
}

// GrpcCodeToHTTPStatus 把 gRPC 状态码映射为语义 HTTP 状态。
//
// 仅用于分类、日志与非信封场景（如触发器执行结果）；网关业务响应统一 HTTP 200。
// See: https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto
func GrpcCodeToHTTPStatus(code codes.Code) int {
	if httpCode, ok := grpcCodeToHTTPStatus[code]; ok {
		return httpCode
	}
	return http.StatusInternalServerError
}
