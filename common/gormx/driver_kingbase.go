package gormx

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newKingbaseDialector 复用 postgres 驱动连接金仓 PG 兼容模式。
// 金仓官方 Go 驱动 gokb（gorm-kingbase 内置）使用了 darwin syscall 未定义的
// TCP_KEEPCNT，在 macOS 上无法编译，故不引入；pgx 不识别 kingbase:// 前缀，
// 因此先把 URL 转成 key-value DSN。该路径不支持 SM3/SM4 国密认证。
func newKingbaseDialector(dsn string) (gorm.Dialector, error) {
	kv, err := kingbaseURLToKVDSN(dsn)
	if err != nil {
		return nil, err
	}
	return postgres.Open(kv), nil
}

func kingbaseURLToKVDSN(dsn string) (string, error) {
	return urlToKVDSN(dsn, string(DatabaseKingbase))
}
