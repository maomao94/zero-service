package asynqx

import (
	"fmt"
	"os"

	"github.com/zeromicro/go-zero/core/logx"
)

// BaseLogger 实现 asynq.Logger 接口，桥接 asynq 内部日志到 logx
type BaseLogger struct {
}

func (l *BaseLogger) Debug(args ...any) {
	logx.Debug("[asynq] " + fmt.Sprint(args...))
}

func (l *BaseLogger) Info(args ...any) {
	logx.Info("[asynq] " + fmt.Sprint(args...))
}

func (l *BaseLogger) Warn(args ...any) {
	logx.Error("[asynq] " + fmt.Sprint(args...))
}

func (l *BaseLogger) Error(args ...any) {
	logx.Error("[asynq] " + fmt.Sprint(args...))
}

func (l *BaseLogger) Fatal(args ...any) {
	logx.Error("[asynq] " + fmt.Sprint(args...))
	os.Exit(1)
}
