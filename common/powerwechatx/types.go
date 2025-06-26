package powerwechatx

import (
	"context"
	"github.com/ArtisanCloud/PowerLibs/v3/logger/contract"
	"github.com/zeromicro/go-zero/core/logx"
)

type PowerWechatLogDriver struct {
	ctx context.Context
}

func (l *PowerWechatLogDriver) WithContext(ctx context.Context) contract.LoggerInterface {
	return &PowerWechatLogDriver{
		ctx: ctx,
	}
}

func (l *PowerWechatLogDriver) Debug(msg string, v ...interface{}) {
	logx.WithContext(l.ctx).Debug(msg, v)
}

func (l *PowerWechatLogDriver) Info(msg string, v ...interface{}) {
	logx.WithContext(l.ctx).Info(msg, v)
}

func (l *PowerWechatLogDriver) Warn(msg string, v ...interface{}) {
	logx.WithContext(l.ctx).Info(msg, v)
}

func (l *PowerWechatLogDriver) Error(msg string, v ...interface{}) {
	logx.WithContext(l.ctx).Error(msg, v)
}

func (l *PowerWechatLogDriver) Panic(msg string, v ...interface{}) {
	logx.WithContext(l.ctx).Error(msg, v)
}

func (l *PowerWechatLogDriver) Fatal(msg string, v ...interface{}) {
	logx.WithContext(l.ctx).Error(msg, v)
}

func (l *PowerWechatLogDriver) DebugF(format string, args ...interface{}) {
	logx.WithContext(l.ctx).Debugf(format, args)
}

func (l *PowerWechatLogDriver) InfoF(format string, args ...interface{}) {
	logx.WithContext(l.ctx).Infof(format, args)
}

func (l *PowerWechatLogDriver) WarnF(format string, args ...interface{}) {
	logx.WithContext(l.ctx).Infof(format, args)
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
