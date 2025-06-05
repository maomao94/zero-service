package iec104

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
)

type LogProvider struct {
	logx.Logger
}

func NewLogProvider(context context.Context) *LogProvider {
	return &LogProvider{
		Logger: logx.WithContext(context),
	}
}

func (l *LogProvider) Critical(format string, v ...interface{}) {
	if v == nil {
		l.Logger.Error(format)
	} else {
		l.Logger.Errorf(format, v)
	}
}

func (l *LogProvider) Error(format string, v ...interface{}) {
	if v == nil {
		l.Logger.Error(format)
	} else {
		l.Logger.Errorf(format, v)
	}
}

func (l *LogProvider) Warn(format string, v ...interface{}) {
	if v == nil {
		l.Logger.Error(format)
	} else {
		l.Logger.Errorf(format, v)
	}
}

func (l *LogProvider) Debug(format string, v ...interface{}) {
	if v == nil {
		l.Logger.Debug(format)
	} else {
		l.Logger.Debugf(format, v)
	}
}
