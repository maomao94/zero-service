package fsrestrict

import (
	"path/filepath"

	"github.com/cloudwego/eino/adk/filesystem"
)

// Config 限制 Deep 本地文件 Backend 的可访问路径与操作类型。
type Config struct {
	UserRoots      []string
	SessionBaseDir string
	Policy         Policy
}

// WrapConfigured 根据 Config 包装 inner：无用户根且无会话父目录时返回 inner（不限制）。
func WrapConfigured(inner filesystem.Backend, cfg Config) filesystem.Backend {
	if inner == nil {
		return nil
	}
	if len(cfg.UserRoots) == 0 && cfg.SessionBaseDir == "" {
		return inner
	}
	if cfg.Policy == (Policy{}) {
		cfg.Policy = DefaultPolicy()
	}
	roots := make([]string, 0, len(cfg.UserRoots))
	for _, r := range cfg.UserRoots {
		if r == "" {
			continue
		}
		roots = append(roots, filepath.Clean(r))
	}
	cfg.UserRoots = roots
	if cfg.SessionBaseDir != "" {
		cfg.SessionBaseDir = filepath.Clean(cfg.SessionBaseDir)
	}
	return &policyBackend{inner: inner, cfg: cfg}
}
