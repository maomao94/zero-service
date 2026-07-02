package gnetx

import (
	"fmt"

	"github.com/panjf2000/gnet/v2/pkg/logging"
	"github.com/zeromicro/go-zero/core/logx"
)

// logxLogger 把 go-zero logx 适配为 gnet 的 logging.Logger 接口。
// 通过 gnet.WithLogger 注入后，gnet 内部所有日志走 logx，与项目其他 common/*x 包统一。
//
// gnet logging.Logger 接口：Debugf/Infof/Warnf/Errorf/Fatalf。
// logx 无独立 Warnf，用 Errorf 承载 Warn；Fatalf 用 Errorf 记录后 panic。
type logxLogger struct{}

// logxAdapter 是 logxLogger 单例，供 Server/Client 注入 gnet.WithLogger 复用。
var logxAdapter logging.Logger = logxLogger{}

func (logxLogger) Debugf(format string, args ...any) { logx.Debugf(format, args...) }
func (logxLogger) Infof(format string, args ...any)  { logx.Infof(format, args...) }
func (logxLogger) Warnf(format string, args ...any)  { logx.Errorf(format, args...) }
func (logxLogger) Errorf(format string, args ...any) { logx.Errorf(format, args...) }
func (logxLogger) Fatalf(format string, args ...any) {
	logx.Must(fmt.Errorf(format, args...))
}
