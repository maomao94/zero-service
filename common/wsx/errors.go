package wsx

import "errors"

var (
	ErrNotConnected     = errors.New("[wsx] not connected to server")
	ErrNotRunning       = errors.New("[wsx] client not running")
	ErrAlreadyRunning   = errors.New("[wsx] client already running")
	ErrAuthTimeout      = errors.New("[wsx] authentication timeout")
	ErrAuthFailed       = errors.New("[wsx] authentication failed")
	ErrAuthCanceled     = errors.New("[wsx] authentication canceled")
	ErrTokenRefresh     = errors.New("[wsx] token refresh failed")
	ErrMaxReconnect     = errors.New("[wsx] reached max reconnect retries")
	ErrConnNil          = errors.New("[wsx] connection is nil")
	ErrNotAuthenticated = errors.New("[wsx] client not authenticated")
)
