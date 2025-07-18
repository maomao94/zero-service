package asynqx

import (
	"github.com/zeromicro/go-zero/core/logx"
	"os"
)

type BaseLogger struct {
}

// Debug logs a message at Debug level.
func (l *BaseLogger) Debug(args ...interface{}) {
	logx.Debug(args...)
}

// Info logs a message at Info level.
func (l *BaseLogger) Info(args ...interface{}) {
	logx.Info(args...)
}

// Warn logs a message at Warning level.
func (l *BaseLogger) Warn(args ...interface{}) {
	logx.Info(args...)
}

// Error logs a message at Error level.
func (l *BaseLogger) Error(args ...interface{}) {
	logx.Error(args...)
}

// Fatal logs a message at Fatal level
// and process will exit with status set to 1.
func (l *BaseLogger) Fatal(args ...interface{}) {
	l.Error(args...)
	os.Exit(1)
}
