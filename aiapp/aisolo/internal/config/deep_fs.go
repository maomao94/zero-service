package config

import (
	"fmt"
	"os"
	"path/filepath"

	"zero-service/common/einox/fsrestrict"
)

// ResolvedDeepFilesystemRoots 将配置的相对路径转为绝对路径并校验为目录。
func ResolvedDeepFilesystemRoots(relRoots []string) ([]string, error) {
	if len(relRoots) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(relRoots))
	for _, r := range relRoots {
		if r == "" {
			continue
		}
		abs, err := filepath.Abs(r)
		if err != nil {
			return nil, fmt.Errorf("agent.deep.filesystemAllowedRoots abs(%q): %w", r, err)
		}
		abs = filepath.Clean(abs)
		fi, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf("agent.deep.filesystemAllowedRoots stat %s: %w", abs, err)
		}
		if !fi.IsDir() {
			return nil, fmt.Errorf("agent.deep.filesystemAllowedRoots not a directory: %s", abs)
		}
		out = append(out, abs)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("agent.deep.filesystemAllowedRoots: no valid entries after resolving")
	}
	return out, nil
}

// ResolvedSessionBaseDir 解析会话工作区父目录为绝对路径并校验为已存在目录。
func ResolvedSessionBaseDir(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("agent.deep.filesystemSessionBaseDir abs(%q): %w", path, err)
	}
	abs = filepath.Clean(abs)
	fi, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("agent.deep.filesystemSessionBaseDir stat %s: %w", abs, err)
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("agent.deep.filesystemSessionBaseDir not a directory: %s", abs)
	}
	return abs, nil
}

func validateDeepFilesystemRoots(c *Config) error {
	if !c.Agent.Deep.EnableLocalFilesystem {
		return nil
	}
	if c.Agent.Deep.FilesystemSessionBaseDir != "" {
		if _, err := ResolvedSessionBaseDir(c.Agent.Deep.FilesystemSessionBaseDir); err != nil {
			return err
		}
	}
	if len(c.Agent.Deep.FilesystemAllowedRoots) > 0 {
		if _, err := ResolvedDeepFilesystemRoots(c.Agent.Deep.FilesystemAllowedRoots); err != nil {
			return err
		}
	}
	return nil
}

// ToFSPolicy 将 yaml 配置映射为 fsrestrict.Policy（各 bool 原样使用，依赖 go-zero default tag 填充）。
func (p DeepFilesystemPolicy) ToFSPolicy() fsrestrict.Policy {
	return fsrestrict.Policy{
		ReadUser: p.ReadUser, WriteUser: p.WriteUser, EditUser: p.EditUser,
		ReadSession: p.ReadSession, WriteSession: p.WriteSession, EditSession: p.EditSession,
	}
}
