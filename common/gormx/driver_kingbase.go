package gormx

import (
	"net/url"
	"strings"

	kingbase "github.com/godoes/gorm-kingbase"
	"gorm.io/gorm"
)

// kingbaseOptionWithoutQuotingCheck 是 gormx 在 kingbase:// URL 查询参数中定义的
// 专用配置项，转换 DSN 前会被摘出映射到驱动 Config，不透传给 gokb。
const kingbaseOptionWithoutQuotingCheck = "without_quoting_check"

// newKingbaseDialector 使用金仓官方 gokb 驱动（gorm-kingbase 内置）构建 dialector。
// DriverName 固定为 kingbase：gorm-kingbase 默认走 pgx 连接（与 postgres 驱动无异），
// 设置 DriverName 后才走 gokb，获得 SM3 国密认证与 pg/oracle/mysql/sqlserver 兼容
// 模式类型 OID 自动感知能力（gokb 连接后自动执行 show database_mode 切换映射）。
// gokb 的 DSN 为 key-value 形式，kingbase:// URL 仍先转换为 KV DSN；Oracle 兼容
// 模式标识符大小写敏感时，在 URL 查询参数中加 without_quoting_check=true 关闭
// 引号转义检查。注意：gokb 使用了 darwin syscall 未定义的 TCP_KEEPCNT，上游
// v1.11.0 在 macOS 上无法编译，需通过 go.mod replace 指向带修复的 fork。
func newKingbaseDialector(dsn string) (gorm.Dialector, error) {
	dsn, opts, err := popKingbaseOptions(dsn)
	if err != nil {
		return nil, err
	}
	kv, err := kingbaseURLToKVDSN(dsn)
	if err != nil {
		return nil, err
	}
	return kingbase.New(kingbase.Config{
		DriverName:          kingbase.DriverName,
		DSN:                 kv,
		WithoutQuotingCheck: opts.withoutQuotingCheck,
	}), nil
}

func kingbaseURLToKVDSN(dsn string) (string, error) {
	return urlToKVDSN(dsn, string(DatabaseKingbase))
}

// kingbaseOptions 承载从 DSN 摘出的 gormx 专用金仓配置。
type kingbaseOptions struct {
	withoutQuotingCheck bool
}

// popKingbaseOptions 从 kingbase:// URL 的查询参数中摘出 gormx 专用配置项，
// 返回移除这些参数后的 DSN；非 URL 形式（KV DSN）原样返回，专用配置仅支持
// URL 形式传入。重复参数全部摘除，任一取值为 true（不区分大小写）即开启。
func popKingbaseOptions(dsn string) (string, kingbaseOptions, error) {
	var opts kingbaseOptions
	if !strings.HasPrefix(strings.ToLower(dsn), string(DatabaseKingbase)+"://") {
		return dsn, opts, nil
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return "", opts, err
	}
	var kept []string
	for _, pair := range strings.Split(u.RawQuery, "&") {
		key, value, _ := strings.Cut(pair, "=")
		if decoded, unescapeErr := url.QueryUnescape(key); unescapeErr == nil {
			key = decoded
		}
		if key != kingbaseOptionWithoutQuotingCheck {
			kept = append(kept, pair)
			continue
		}
		if decoded, unescapeErr := url.QueryUnescape(value); unescapeErr == nil {
			value = decoded
		}
		if strings.EqualFold(value, "true") {
			opts.withoutQuotingCheck = true
		}
	}
	u.RawQuery = strings.Join(kept, "&")
	return u.String(), opts, nil
}
