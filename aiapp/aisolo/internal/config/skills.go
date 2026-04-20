package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EffectiveSkillsDir 根据 Skills 配置解析 skill 根目录的绝对路径。
// 若设置环境变量 EINO_EXT_SKILLS_DIR（与 eino-examples chatwitheino 一致）且为有效目录，优先使用。
// 未启用、未配置目录或路径不存在时返回空字符串（不视为错误）。
func EffectiveSkillsDir(sk SkillsConfig) string {
	if env := strings.TrimSpace(os.Getenv("EINO_EXT_SKILLS_DIR")); env != "" {
		if abs, err := filepath.Abs(env); err == nil {
			if fi, err := os.Stat(abs); err == nil && fi.IsDir() {
				return abs
			}
		}
	}
	if !sk.Enabled || sk.Dir == "" {
		return ""
	}
	abs, err := filepath.Abs(sk.Dir)
	if err != nil {
		return ""
	}
	fi, err := os.Stat(abs)
	if err != nil || !fi.IsDir() {
		return ""
	}
	return abs
}

func (c *Config) Validate() error {
	if err := validateSkillsStrict(&c.Skills); err != nil {
		return err
	}
	if err := validateDeepFilesystemRoots(c); err != nil {
		return err
	}
	return nil
}

func validateSkillsStrict(sk *SkillsConfig) error {
	if sk == nil || !sk.Strict {
		return nil
	}
	if !sk.Enabled {
		return nil
	}
	if sk.Dir == "" {
		return fmt.Errorf("skills: strict is true but skills.dir is empty")
	}
	abs, err := filepath.Abs(sk.Dir)
	if err != nil {
		return fmt.Errorf("skills.dir abs(%q): %w", sk.Dir, err)
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("skills.dir stat %s: %w", abs, err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("skills.dir is not a directory: %s", abs)
	}
	return nil
}
