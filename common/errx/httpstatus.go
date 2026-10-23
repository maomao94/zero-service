package errx

import (
	"net/http"

	"google.golang.org/grpc/codes"
)

// GrpcCodeToHTTPStatus 把 gRPC 状态码映射为语义 HTTP 状态。
//
// 仅用于分类、日志与非信封场景（如触发器执行结果）；网关业务响应统一 HTTP 200。
// See: https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto
func GrpcCodeToHTTPStatus(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.AlreadyExists, codes.Aborted:
		return http.StatusConflict
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Internal, codes.DataLoss, codes.Unknown:
		return http.StatusInternalServerError
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}
