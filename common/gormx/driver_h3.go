package gormx

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newH3Dialector 复用 postgres 驱动连接 H3 数据库。
// H3 采用 PostgreSQL 协议兼容模式，h3:// 前缀的 URL 形式 DSN 会先转换成
// pgx 可解析的 key-value 形式，与金仓 KingbaseES 的处理保持一致。
// 该路径不支持 H3 专有驱动才提供的认证或扩展能力。
func newH3Dialector(dsn string) (gorm.Dialector, error) {
	kv, err := pgCompatURLToKVDSN(dsn, DatabaseH3)
	if err != nil {
		return nil, err
	}
	return postgres.Open(kv), nil
}
