package ai

import (
	"encoding/json"
	"net/http"
)

// OpenAIError OpenAI 标准错误响应格式
type OpenAIError struct {
	Error OpenAIErrorBody `json:"error"`
}

type OpenAIErrorBody struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// writeOpenAIError 写入 OpenAI 格式的错误响应
func writeOpenAIError(w http.ResponseWriter, statusCode int, errType, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(OpenAIError{
		Error: OpenAIErrorBody{
			Message: message,
			Type:    errType,
			Code:    code,
		},
	})
}
