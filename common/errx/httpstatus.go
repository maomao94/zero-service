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
//	400                | InvalidArgument   | 参数、业务校验类
//	401                | Unauthenticated   | 凭据、签名、时间戳类
//	403                | PermissionDenied  | 权限、账号、租户、IP 类
//	404                | NotFound          | 记录、路由、文件、配置类
//	405                | Unimplemented     | 请求方法不允许
//	408                | Canceled          | 请求取消或请求超时
//	409                | Aborted           | 数据、版本、业务状态冲突
//	413                | ResourceExhausted | 请求体或文件过大
//	415                | InvalidArgument   | 媒体类型或文件类型不支持
//	429                | ResourceExhausted | 平台、业务、外部限流
//	500                | Internal          | 数据库、缓存、内部异常
//	501                | Unimplemented     | 功能未实现
//	502                | Unavailable       | 网关错误、第三方接口错误
//	503                | Unavailable       | 下游、第三方、消息队列不可用
//	504                | DeadlineExceeded  | 系统、下游、网关超时
//
// 本表是正向映射，保证每个 extproto 码的 http_code 都能推导出标准 gRPC Code。
// 反向的 grpcCodeToHTTPStatus 只用于分类，同一 gRPC Code 可能对应多个 HTTP 状态
// （如 413/429 都收敛到 ResourceExhausted），不能假定两者严格双向可逆。
var httpCodeToGRPCCode = map[int]codes.Code{
	http.StatusOK:                    codes.OK,
	http.StatusBadRequest:            codes.InvalidArgument,
	http.StatusUnauthorized:          codes.Unauthenticated,
	http.StatusForbidden:             codes.PermissionDenied,
	http.StatusNotFound:              codes.NotFound,
	http.StatusMethodNotAllowed:      codes.Unimplemented,
	http.StatusRequestTimeout:        codes.Canceled,
	http.StatusConflict:              codes.Aborted,
	http.StatusRequestEntityTooLarge: codes.ResourceExhausted,
	http.StatusUnsupportedMediaType:  codes.InvalidArgument,
	http.StatusTooManyRequests:       codes.ResourceExhausted,
	http.StatusInternalServerError:   codes.Internal,
	http.StatusNotImplemented:        codes.Unimplemented,
	http.StatusBadGateway:            codes.Unavailable,
	http.StatusServiceUnavailable:    codes.Unavailable,
	http.StatusGatewayTimeout:        codes.DeadlineExceeded,
}

// grpcCodeToHTTPStatus 是标准 gRPC Code 到语义 HTTP 状态的分类表。
//
// 两处按 extproto 口径收敛，避免 fallback detail 与业务码 http_code 错位：
//   - Unimplemented -> 405：对应 100068 请求方法不允许；
//   - FailedPrecondition -> 409：对应 100045 业务状态不允许。
//
// 同一 gRPC Code 可能对应多个 HTTP 状态（如 ResourceExhausted 对应 413/429，
// Unavailable 对应 502/503），同一 HTTP 状态也可能对应多个 gRPC Code（如 409
// 对应 AlreadyExists/Aborted/FailedPrecondition），因此该表不是严格反函数。
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
