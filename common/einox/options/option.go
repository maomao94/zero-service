package options

import "time"

// OptionFunc 选项函数类型
type OptionFunc[T any] func(*T)

// Apply 应用多个选项到目标
func Apply[T any](target *T, opts ...OptionFunc[T]) {
	for _, opt := range opts {
		opt(target)
	}
}

// =============================================================================
// 通用选项
// =============================================================================

// TimeoutOption 超时选项
type TimeoutOption struct {
	Timeout time.Duration
}

// WithTimeout 设置超时
func WithTimeout(d time.Duration) OptionFunc[TimeoutOption] {
	return func(o *TimeoutOption) {
		o.Timeout = d
	}
}

// APIKeyOption API Key 选项
type APIKeyOption struct {
	APIKey string
}

// WithAPIKey 设置 API Key
func WithAPIKey(key string) OptionFunc[APIKeyOption] {
	return func(o *APIKeyOption) {
		o.APIKey = key
	}
}

// BaseURLOption Base URL 选项
type BaseURLOption struct {
	BaseURL string
}

// WithBaseURL 设置 Base URL
func WithBaseURL(url string) OptionFunc[BaseURLOption] {
	return func(o *BaseURLOption) {
		o.BaseURL = url
	}
}
