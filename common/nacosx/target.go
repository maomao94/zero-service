package nacosx

import (
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/mapping"
)

type target struct {
	Addr                 string        `key:",optional"`
	User                 string        `key:",optional"`
	Password             string        `key:",optional"`
	Service              string        `key:",optional"`
	GroupName            string        `key:"groupName,optional"`
	Clusters             []string      `key:",optional"` // 逗号分隔，由 parseURL 单独解析
	NamespaceID          string        `key:"namespaceid,optional"`
	Timeout              time.Duration `key:"timeout,optional"`
	AppName              string        `key:"appName,optional"`
	LogLevel             string        `key:"logLevel,optional"`
	LogDir               string        `key:"logDir,optional"`
	CacheDir             string        `key:"cacheDir,optional"`
	NotLoadCacheAtStart  string        `key:"notLoadCacheAtStart,optional"`
	UpdateCacheWhenEmpty string        `key:"updateCacheWhenEmpty,optional"`
}

// parseURL with parameters
func parseURL(rawURL url.URL) (target, error) {
	if rawURL.Scheme != schemeName ||
		len(rawURL.Host) == 0 || len(strings.TrimLeft(rawURL.Path, "/")) == 0 {
		return target{},
			errors.Errorf("Malformed URL('%s'). Must be in the next format: 'nacos://[user:passwd]@host/service?param=value'", rawURL.String())
	}

	var tgt target
	params := make(map[string]any, len(rawURL.Query()))
	for name, value := range rawURL.Query() {
		// clusters 是逗号分隔字符串，mapping 会按 JSON 解析失败，单独处理。
		if name == "clusters" {
			continue
		}
		params[name] = value[0]
	}

	err := mapping.UnmarshalKey(params, &tgt)
	if err != nil {
		return target{}, errors.Wrap(err, "Malformed URL parameters")
	}

	if clusters := rawURL.Query().Get("clusters"); clusters != "" {
		for _, cluster := range strings.Split(clusters, ",") {
			if cluster = strings.TrimSpace(cluster); cluster != "" {
				tgt.Clusters = append(tgt.Clusters, cluster)
			}
		}
	}

	if tgt.NamespaceID == "" {
		tgt.NamespaceID = "public"
	}

	tgt.User = rawURL.User.Username()
	tgt.Password, _ = rawURL.User.Password()
	tgt.Addr = rawURL.Host
	tgt.Service = strings.TrimLeft(rawURL.Path, "/")

	// 环境变量优先于 URL 参数，未设置时保留 URL 中的值。
	if logLevel, exists := os.LookupEnv(envLogLevel); exists {
		tgt.LogLevel = logLevel
	}

	if logDir, exists := os.LookupEnv(envLogDir); exists {
		tgt.LogDir = logDir
	}

	if cacheDir, exists := os.LookupEnv(envCacheDir); exists {
		tgt.CacheDir = cacheDir
	}

	if notLoadCacheAtStart, exists := os.LookupEnv(envNotLoadCacheAtStart); exists {
		tgt.NotLoadCacheAtStart = notLoadCacheAtStart
	}

	if updateCacheWhenEmpty, exists := os.LookupEnv(envUpdateCacheWhenEmpty); exists {
		tgt.UpdateCacheWhenEmpty = updateCacheWhenEmpty
	}

	return tgt, nil
}

func targetBool(value string, defaultValue bool) bool {
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
