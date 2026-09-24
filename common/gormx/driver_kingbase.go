package gormx

import (
	"net"
	"net/url"
	"strings"

	"github.com/pkg/errors"
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

var pgCompatDSNEscaper = strings.NewReplacer(`\`, `\\`, " ", `\ `, "'", `\'`)

func kingbaseURLToKVDSN(dsn string) (string, error) {
	return pgCompatURLToKVDSN(dsn, DatabaseKingbase)
}

// pgCompatURLToKVDSN 把 <dbType>:// 前缀的 URL 形式 DSN 转成 pgx 可解析的 key-value
// DSN，供金仓 KingbaseES、H3 等 PostgreSQL 协议兼容数据库复用；非该前缀时原样返回。
func pgCompatURLToKVDSN(dsn string, dbType DatabaseType) (string, error) {
	if !strings.HasPrefix(strings.ToLower(dsn), string(dbType)+"://") {
		return dsn, nil
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return "", errors.Wrapf(err, "invalid %s dsn", dbType)
	}

	var parts []string
	appendKV := func(k, v string) {
		if v != "" {
			parts = append(parts, k+"="+pgCompatDSNEscaper.Replace(v))
		}
	}

	if u.User != nil {
		appendKV("user", u.User.Username())
		if password, ok := u.User.Password(); ok {
			appendKV("password", password)
		}
	}

	var hosts, ports []string
	for _, item := range strings.Split(u.Host, ",") {
		if item == "" {
			continue
		}
		if host, port, splitErr := net.SplitHostPort(item); splitErr == nil {
			if host != "" {
				hosts = append(hosts, host)
			}
			if port != "" {
				ports = append(ports, port)
			}
			continue
		}
		hosts = append(hosts, strings.Trim(item, "[]"))
	}
	appendKV("host", strings.Join(hosts, ","))
	appendKV("port", strings.Join(ports, ","))
	appendKV("dbname", strings.TrimPrefix(u.Path, "/"))

	for key, values := range u.Query() {
		if len(values) == 0 {
			continue
		}
		appendKV(key, values[0])
	}

	return strings.Join(parts, " "), nil
}
