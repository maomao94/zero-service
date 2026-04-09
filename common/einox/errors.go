package einox

import "errors"

// =============================================================================
// Agent 错误定义
// =============================================================================

// Agent 相关错误
var (
	// ErrUserIDRequired user_id 必填
	ErrUserIDRequired = errors.New("user_id is required")

	// ErrSessionIDRequired session_id 必填
	ErrSessionIDRequired = errors.New("session_id is required")

	// ErrAgentNotFound Agent 未找到
	ErrAgentNotFound = errors.New("agent not found")

	// ErrSessionNotFound 会话未找到
	ErrSessionNotFound = errors.New("session not found")

	// ErrInvalidArgument 无效参数
	ErrInvalidArgument = errors.New("invalid argument")
)
