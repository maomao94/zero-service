package gormx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type LoggerConfig struct {
	SlowThreshold time.Duration
	LogLevel      logger.LogLevel
}

type gormLogger struct {
	cfg LoggerConfig
}

func NewGormLogger(cfg LoggerConfig) logger.Interface {
	return &gormLogger{cfg: cfg}
}

func DefaultGormLogger() logger.Interface {
	return NewGormLogger(LoggerConfig{
		LogLevel:      logger.Error,
		SlowThreshold: 200 * time.Millisecond,
	})
}

func QuietGormLogger() logger.Interface {
	return NewGormLogger(LoggerConfig{
		LogLevel: logger.Silent,
	})
}

func (c *gormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &gormLogger{LoggerConfig{
		LogLevel:      level,
		SlowThreshold: c.cfg.SlowThreshold,
	}}
}

func (c *gormLogger) Info(ctx context.Context, message string, data ...any) {
	if c.cfg.LogLevel >= logger.Info {
		logx.WithContext(ctx).Infof("[gorm] "+message, data...)
	}
}

func (c *gormLogger) Warn(ctx context.Context, message string, data ...any) {
	if c.cfg.LogLevel >= logger.Warn {
		logx.WithContext(ctx).Slowf("[gorm] "+message, data...)
	}
}

func (c *gormLogger) Error(ctx context.Context, message string, data ...any) {
	if c.cfg.LogLevel >= logger.Error {
		logx.WithContext(ctx).Errorf("[gorm] "+message, data...)
	}
}

func (c *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if c.cfg.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	rowsStr := formatRows(rows)

	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && c.cfg.LogLevel >= logger.Error:
		logx.WithContext(ctx).WithDuration(elapsed).Errorf("[gorm] [rows:%s] %s error: %v", rowsStr, sql, err)

	case err != nil && errors.Is(err, gorm.ErrRecordNotFound) && c.cfg.LogLevel >= logger.Info:
		logx.WithContext(ctx).WithDuration(elapsed).Infof("[gorm] [rows:%s] %s record not found", rowsStr, sql)

	case elapsed > c.cfg.SlowThreshold && c.cfg.SlowThreshold != 0 && c.cfg.LogLevel >= logger.Warn:
		logx.WithContext(ctx).WithDuration(elapsed).Slowf("[gorm] [rows:%s] [SLOW] %s", rowsStr, sql)

	case c.cfg.LogLevel >= logger.Info:
		logx.WithContext(ctx).WithDuration(elapsed).Infof("[gorm] [rows:%s] %s", rowsStr, sql)
	}
}

func formatRows(rows int64) string {
	if rows == -1 {
		return "-"
	}
	return fmt.Sprintf("%d", rows)
}
