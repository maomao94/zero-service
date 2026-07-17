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

type fullSQLKey struct{}
type skipSQLTraceKey struct{}

func WithFullSQL(ctx context.Context) context.Context {
	return context.WithValue(ctx, fullSQLKey{}, true)
}

func WithoutSQLTrace(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipSQLTraceKey{}, true)
}

type LoggerConfig struct {
	SlowThreshold             time.Duration
	LogLevel                  logger.LogLevel
	ParameterizedQueries      bool
	IgnoreRecordNotFoundError bool
}

type gormLogger struct {
	cfg LoggerConfig
}

func NewGormLogger(cfg LoggerConfig) logger.Interface {
	return &gormLogger{cfg: cfg}
}

func DefaultGormLogger() logger.Interface {
	return NewGormLogger(LoggerConfig{
		LogLevel:             logger.Error,
		SlowThreshold:        200 * time.Millisecond,
		ParameterizedQueries: true,
	})
}

func QuietGormLogger() logger.Interface {
	return NewGormLogger(LoggerConfig{LogLevel: logger.Silent})
}

func (c *gormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &gormLogger{cfg: LoggerConfig{
		LogLevel:                  level,
		SlowThreshold:             c.cfg.SlowThreshold,
		ParameterizedQueries:      c.cfg.ParameterizedQueries,
		IgnoreRecordNotFoundError: c.cfg.IgnoreRecordNotFoundError,
	}}
}

func (c *gormLogger) ParamsFilter(ctx context.Context, sql string, params ...any) (string, []any) {
	if c.cfg.ParameterizedQueries {
		return sql, nil
	}
	if _, ok := ctx.Value(fullSQLKey{}).(bool); ok {
		return sql, params
	}
	return sql, params
}

func (c *gormLogger) Info(ctx context.Context, msg string, data ...any) {
	if c.cfg.LogLevel >= logger.Info {
		logx.WithContext(ctx).Infof("[gorm] "+msg, data...)
	}
}

func (c *gormLogger) Warn(ctx context.Context, msg string, data ...any) {
	if c.cfg.LogLevel >= logger.Warn {
		logx.WithContext(ctx).Slowf("[gorm] "+msg, data...)
	}
}

func (c *gormLogger) Error(ctx context.Context, msg string, data ...any) {
	if c.cfg.LogLevel >= logger.Error {
		logx.WithContext(ctx).Errorf("[gorm] "+msg, data...)
	}
}

func (c *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if c.cfg.LogLevel <= logger.Silent {
		return
	}
	elapsed := time.Since(begin)
	slow := isSlowSQL(elapsed, c.cfg.SlowThreshold)
	traceDisabled := isSQLTraceDisabled(ctx)

	shouldLogError := err != nil && c.cfg.LogLevel >= logger.Error && (!errors.Is(err, gorm.ErrRecordNotFound) || !c.cfg.IgnoreRecordNotFoundError)
	shouldLogRecordNotFound := err != nil && errors.Is(err, gorm.ErrRecordNotFound) && c.cfg.LogLevel >= logger.Info
	shouldLogSlow := slow && c.cfg.LogLevel >= logger.Error
	shouldLogInfo := err == nil && c.cfg.LogLevel >= logger.Info && !traceDisabled
	if !shouldLogError && !shouldLogRecordNotFound && !shouldLogSlow && !shouldLogInfo {
		return
	}

	sql, rows := fc()
	if shouldLogError {
		logx.WithContext(ctx).WithDuration(elapsed).Errorf("[gorm] [rows:%s] %s error: %v", formatRows(rows), sql, err)
	}
	if shouldLogRecordNotFound {
		logx.WithContext(ctx).WithDuration(elapsed).Infof("[gorm] [rows:%s] %s record not found", formatRows(rows), sql)
	}
	if shouldLogSlow {
		logx.WithContext(ctx).WithDuration(elapsed).Slowf("[gorm] [rows:%s] [SLOW] %s", formatRows(rows), sql)
	}
	if shouldLogInfo {
		logx.WithContext(ctx).WithDuration(elapsed).Infof("[gorm] [rows:%s] %s", formatRows(rows), sql)
	}
}

func isSQLTraceDisabled(ctx context.Context) bool {
	skip, ok := ctx.Value(skipSQLTraceKey{}).(bool)
	return ok && skip
}

func isSlowSQL(elapsed time.Duration, threshold time.Duration) bool {
	return threshold != 0 && elapsed > threshold
}

func formatRows(rows int64) string {
	if rows == -1 {
		return "-"
	}
	return fmt.Sprintf("%d", rows)
}
