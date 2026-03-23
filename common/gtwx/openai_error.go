package gtwx

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OpenAIError OpenAI 风格错误响应
type OpenAIError struct {
	ErrorMsg   OpenAIErrorDetail `json:"error"`
	HTTPStatus int               `json:"-"`
}

// OpenAIErrorDetail OpenAI 错误详情
type OpenAIErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

func (e *OpenAIError) Error() string {
	return e.ErrorMsg.Message
}

// openAIErrorBody 是纯 JSON 序列化用的响应结构体。
// 不实现 error 接口，确保 go-zero 的 doHandleError 走 writeJson 分支而非 http.Error 纯文本分支。
type openAIErrorBody struct {
	ErrorMsg OpenAIErrorDetail `json:"error"`
}

// NewModelNotFoundError 模型未找到错误
func NewModelNotFoundError(model string, available []string) *OpenAIError {
	return &OpenAIError{
		HTTPStatus: http.StatusNotFound,
		ErrorMsg: OpenAIErrorDetail{
			Type:    "invalid_request_error",
			Message: fmt.Sprintf("model '%s' not found, available: %v", model, available),
			Code:    "model_not_found",
		},
	}
}

// NewInvalidRequestError 无效请求错误
func NewInvalidRequestError(msg string) *OpenAIError {
	return &OpenAIError{
		HTTPStatus: http.StatusBadRequest,
		ErrorMsg: OpenAIErrorDetail{
			Type:    "invalid_request_error",
			Message: msg,
			Code:    "invalid_request",
		},
	}
}

// NewInternalError 内部错误
func NewInternalError(msg string) *OpenAIError {
	return &OpenAIError{
		HTTPStatus: http.StatusInternalServerError,
		ErrorMsg: OpenAIErrorDetail{
			Type:    "internal_error",
			Message: msg,
		},
	}
}

// SetOpenAIErrorHandler sets an httpx error handler that converts all errors
// (including gRPC errors) to OpenAI-style JSON responses.
func SetOpenAIErrorHandler() {
	httpx.SetErrorHandlerCtx(func(ctx context.Context, err error) (int, any) {
		// 已经是 OpenAIError，直接返回
		var openAIErr *OpenAIError
		if errors.As(err, &openAIErr) {
			return openAIErr.HTTPStatus, &openAIErrorBody{ErrorMsg: openAIErr.ErrorMsg}
		}

		// gRPC status error -> OpenAI error
		if st, ok := status.FromError(err); ok {
			httpStatus := GrpcCodeToHTTPStatus(st.Code())
			return httpStatus, &openAIErrorBody{
				ErrorMsg: OpenAIErrorDetail{
					Type:    grpcCodeToOpenAIType(st.Code()),
					Message: st.Message(),
					Code:    grpcCodeToOpenAICode(st.Code()),
				},
			}
		}

		// 其他错误 -> internal_error
		return http.StatusInternalServerError, &openAIErrorBody{
			ErrorMsg: OpenAIErrorDetail{
				Type:    "internal_error",
				Message: err.Error(),
			},
		}
	})
}

// grpcCodeToOpenAIType 将 gRPC status code 映射为 OpenAI error type
func grpcCodeToOpenAIType(code codes.Code) string {
	switch code {
	case codes.NotFound:
		return "invalid_request_error"
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return "invalid_request_error"
	case codes.Unauthenticated:
		return "authentication_error"
	case codes.PermissionDenied:
		return "permission_error"
	case codes.ResourceExhausted:
		return "rate_limit_error"
	case codes.DeadlineExceeded, codes.Canceled:
		return "timeout_error"
	case codes.Unavailable:
		return "upstream_error"
	case codes.AlreadyExists, codes.Aborted:
		return "conflict_error"
	default:
		return "internal_error"
	}
}

// grpcCodeToOpenAICode 将 gRPC status code 映射为 OpenAI error code
func grpcCodeToOpenAICode(code codes.Code) string {
	switch code {
	case codes.NotFound:
		return "model_not_found"
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return "invalid_request"
	case codes.Unauthenticated:
		return "invalid_api_key"
	case codes.PermissionDenied:
		return "permission_denied"
	case codes.ResourceExhausted:
		return "rate_limit_exceeded"
	case codes.DeadlineExceeded, codes.Canceled:
		return "timeout"
	case codes.Unavailable:
		return "upstream_unavailable"
	case codes.AlreadyExists, codes.Aborted:
		return "conflict"
	default:
		return ""
	}
}
