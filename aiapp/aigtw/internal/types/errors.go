package types

import "fmt"

// OpenAIError OpenAI 风格错误响应
type OpenAIError struct {
	ErrorMsg   ErrorDetail `json:"error"`
	HTTPStatus int         `json:"-"`
}

type ErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

func (e *OpenAIError) Error() string {
	return e.ErrorMsg.Message
}

// NewModelNotFoundError 模型未找到错误
func NewModelNotFoundError(model string, available []string) *OpenAIError {
	return &OpenAIError{
		HTTPStatus: 404,
		ErrorMsg: ErrorDetail{
			Type:    "invalid_request_error",
			Message: fmt.Sprintf("model '%s' not found, available: %v", model, available),
			Code:    "model_not_found",
		},
	}
}

// NewInvalidRequestError 无效请求错误
func NewInvalidRequestError(msg string) *OpenAIError {
	return &OpenAIError{
		HTTPStatus: 400,
		ErrorMsg: ErrorDetail{
			Type:    "invalid_request_error",
			Message: msg,
			Code:    "invalid_request",
		},
	}
}

// NewInternalError 内部错误
func NewInternalError(msg string) *OpenAIError {
	return &OpenAIError{
		HTTPStatus: 500,
		ErrorMsg: ErrorDetail{
			Type:    "internal_error",
			Message: msg,
		},
	}
}
