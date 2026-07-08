package wsx

import "errors"

var (
	ErrNotConnected     = errors.New("[wsx] not connected to server")
	ErrNotAuthenticated = errors.New("[wsx] not authenticated")
	ErrAuthTimeout      = errors.New("[wsx] authentication timeout")
	ErrAuthFailed       = errors.New("[wsx] authentication failed")
	ErrAuthCanceled     = errors.New("[wsx] authentication canceled")
	ErrTokenRefresh     = errors.New("[wsx] token refresh failed")
)
