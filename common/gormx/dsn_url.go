package gormx

import (
	"net"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

// kvDSNEscaper 转义 key-value DSN 值中的特殊字符，转义规则遵循 libpq 关键字/值格式约定。
var kvDSNEscaper = strings.NewReplacer(`\`, `\\`, " ", `\ `, "'", `\'`)

// urlToKVDSN 把 <scheme>:// 前缀的 URL 形式 DSN（scheme://user:pass@host:port/dbname?k=v）
// 按标准 URL 语义解析并转成 key-value 形式 DSN，供 PostgreSQL 协议兼容数据库
// （金仓 KingbaseES 等）复用 postgres 驱动时使用；dsn 不带该前缀时原样返回。
// 解析逻辑与具体数据库无关：scheme 仅用于前缀匹配和错误信息。
// 查询参数按出现顺序输出，重复 key 保留首个。
func urlToKVDSN(dsn, scheme string) (string, error) {
	if !strings.HasPrefix(strings.ToLower(dsn), scheme+"://") {
		return dsn, nil
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return "", errors.Wrapf(err, "invalid %s dsn", scheme)
	}

	var parts []string
	appendKV := func(k, v string) {
		if v != "" {
			parts = append(parts, k+"="+kvDSNEscaper.Replace(v))
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

	seen := make(map[string]struct{})
	for _, pair := range strings.Split(u.RawQuery, "&") {
		key, value, _ := strings.Cut(pair, "=")
		if key == "" {
			continue
		}
		if decoded, unescapeErr := url.QueryUnescape(key); unescapeErr == nil {
			key = decoded
		}
		if decoded, unescapeErr := url.QueryUnescape(value); unescapeErr == nil {
			value = decoded
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		appendKV(key, value)
	}

	return strings.Join(parts, " "), nil
}
