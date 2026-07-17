package gormx

import (
	"gorm.io/gorm/logger"
	"time"
)

type Config struct {
	// 数据库连接地址，支持 MySQL/PostgreSQL/SQLite/GaussDB 自动识别。
	// MySQL:      user:pass@tcp(host:port)/db?charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai
	// PostgreSQL: postgres://user:pass@host:port/db?sslmode=disable&TimeZone=Asia/Shanghai
	// GaussDB:    use PostgreSQL-compatible DSN: postgres://user:pass@host:port/db?sslmode=disable&TimeZone=Asia/Shanghai
	// SQLite:     file:./data.db?cache=shared
	DataSource string `json:",optional"`
	// 最大空闲连接数，默认 100。建议与 MaxOpenConns 一致，避免连接抖动。
	MaxIdleConns int `json:",optional,default=100"`
	// 最大打开连接数，默认 100。根据 DB 最大连接数和实例数调整，公式: (db_max / 实例数) * 0.8。
	MaxOpenConns int `json:",optional,default=100"`
	// 连接最大生命周期，默认 1h。有负载均衡时建议缩短到 5-30min。
	ConnMaxLifetime time.Duration `json:",optional,default=1h"`
	// 空闲连接最大存活时间，默认 5min。低流量时自动清理闲置连接，防止被 DB 服务端断开。
	ConnMaxIdleTime time.Duration `json:",optional,default=5m"`
	// 慢 SQL 阈值，默认 200ms。超过此时间的查询会被记录为慢查询。
	SlowThreshold time.Duration `json:",optional,default=200ms"`
	// 日志级别，默认 error。可选: silent / error / warn / info。
	// 生产建议 error 或 warn（warn 会额外记录慢查询）。
	LogLevel string `json:",optional,default=error,options=[silent,error,warn,info]"`
	// 是否脱敏 SQL 参数，默认 true。开启后日志中不打印查询参数值，防止泄露敏感数据（手机号、密码等）。
	ParameterizedQueries bool `json:",optional,default=true"`
	// 是否忽略 record not found 错误日志，默认 false。
	IgnoreRecordNotFoundError bool `json:",optional,default=false"`
	// 是否按字段名显式查询（SELECT col1, col2 而非 SELECT *），默认 false。
	QueryFields bool `json:",optional,default=false"`
	// 是否跳过默认事务包裹，默认 true。单条写操作不再自动 BEGIN/COMMIT，性能提升约 10-30%。
	// 需要多条操作原子性时，使用 db.Transact() 手动包裹。
	SkipDefaultTransaction bool `json:",optional,default=true"`
	// 是否缓存预编译语句，默认 false。开启后重复执行相同 SQL 时跳过解析阶段，降低延迟。
	// 注意：连接池切换或数据库重启后缓存会失效，某些驱动可能存在兼容性问题。
	PrepareStmt bool `json:",optional,default=false"`
	// OpenTelemetry 链路追踪配置。
	Trace TraceConfig `json:",optional"`
}

func parseLogLevel(level string) logger.LogLevel {
	switch level {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return logger.Error
	}
}
