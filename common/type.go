package common

import (
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	ExpireTime      = 30 * 60
	PayType_Wxpay   = "wxpay"
	PayType_Alipay  = "alipay"
	TxnType_Consume = 1000
	TxnType_Refund  = 2000
)

// 定义交易结果的常量
const (
	ResultUnprocessed string = "U" // 未处理
	ResultProcessing  string = "P" // 交易处理中
	ResultFailed      string = "F" // 失败
	ResultTimedOut    string = "T" // 超时
	ResultClosed      string = "C" // 关闭
	ResultSuccessful  string = "S" // 成功
)

type PowerWechatLogDriver struct {
}

func (l *PowerWechatLogDriver) Debug(msg string, v ...interface{}) {
	logx.Debug(msg, v)
}

func (l *PowerWechatLogDriver) Info(msg string, v ...interface{}) {
	logx.Info(msg, v)
}

func (l *PowerWechatLogDriver) Warn(msg string, v ...interface{}) {
	logx.Info(msg, v)
}

func (l *PowerWechatLogDriver) Error(msg string, v ...interface{}) {
	logx.Error(msg, v)
}

func (l *PowerWechatLogDriver) Panic(msg string, v ...interface{}) {
	logx.Error(msg, v)
}

func (l *PowerWechatLogDriver) Fatal(msg string, v ...interface{}) {
	logx.Error(msg, v)
}

func (l *PowerWechatLogDriver) DebugF(format string, args ...interface{}) {
	logx.Debugf(format, args)
}

func (l *PowerWechatLogDriver) InfoF(format string, args ...interface{}) {
	logx.Infof(format, args)
}

func (l *PowerWechatLogDriver) WarnF(format string, args ...interface{}) {
	logx.Infof(format, args)
}

func (l *PowerWechatLogDriver) ErrorF(format string, args ...interface{}) {
	logx.Errorf(format, args)
}

func (l *PowerWechatLogDriver) PanicF(format string, args ...interface{}) {
	logx.Errorf(format, args)
}

func (l *PowerWechatLogDriver) FatalF(format string, args ...interface{}) {
	logx.Errorf(format, args)
}
