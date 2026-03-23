package gtwx

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorResponse is a generic JSON error response body.
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// SetGrpcErrorHandler sets a common httpx error handler that maps gRPC errors
// to appropriate HTTP status codes with a JSON response body.
func SetGrpcErrorHandler() {
	httpx.SetErrorHandlerCtx(func(ctx context.Context, err error) (int, any) {
		if st, ok := status.FromError(err); ok && st.Code() != codes.OK {
			httpStatus := GrpcCodeToHTTPStatus(st.Code())
			return httpStatus, &ErrorResponse{
				Code:    int(st.Code()),
				Message: st.Message(),
			}
		}

		return http.StatusBadRequest, &ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		}
	})
}

// GrpcCodeToHTTPStatus maps a gRPC status code to an HTTP status code.
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

// CodeFromGrpcError converts a gRPC error to an HTTP status code.
func CodeFromGrpcError(err error) int {
	return GrpcCodeToHTTPStatus(status.Code(err))
}

// IsGrpcError checks if the error is a gRPC error.
func IsGrpcError(err error) bool {
	if err == nil {
		return false
	}

	_, ok := err.(interface {
		GRPCStatus() *status.Status
	})

	return ok
}
