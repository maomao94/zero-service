package types

import "zero-service/common/gtwx"

// OpenAIError OpenAI 风格错误响应（类型别名，实际定义在 common/gtwx）
type OpenAIError = gtwx.OpenAIError

// ErrorDetail OpenAI 错误详情（类型别名，实际定义在 common/gtwx）
type ErrorDetail = gtwx.OpenAIErrorDetail

// NewModelNotFoundError 模型未找到错误
var NewModelNotFoundError = gtwx.NewModelNotFoundError

// NewInvalidRequestError 无效请求错误
var NewInvalidRequestError = gtwx.NewInvalidRequestError

// NewInternalError 内部错误
var NewInternalError = gtwx.NewInternalError
