package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var log *zap.SugaredLogger

func Init() error {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	logDir := filepath.Join(home, ".dtui", "log")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logPath := filepath.Join(logDir, "dtui.log")
	writer := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10, // MB
		MaxBackups: 5,
		MaxAge:     30, // days
		Compress:   false,
	}

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "lvl",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.AddSync(writer),
		zapcore.DebugLevel,
	)

	l := zap.New(core, zap.AddCaller())
	log = l.Sugar()
	return nil
}

func Sync() {
	if log != nil {
		_ = log.Sync()
	}
}

func Debugw(msg string, keysAndValues ...interface{}) {
	if log != nil {
		log.Debugw(msg, keysAndValues...)
	}
}

func Infow(msg string, keysAndValues ...interface{}) {
	if log != nil {
		log.Infow(msg, keysAndValues...)
	}
}

func Warnw(msg string, keysAndValues ...interface{}) {
	if log != nil {
		log.Warnw(msg, keysAndValues...)
	}
}

func Errorw(msg string, keysAndValues ...interface{}) {
	if log != nil {
		log.Errorw(msg, keysAndValues...)
	}
}
