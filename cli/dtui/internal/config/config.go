package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Config dtui 配置文件结构。
type Config struct {
	ComposeDirs    []ComposeDir    `json:"compose_dirs"`
	DeployTargets  []DeployTarget  `json:"deploy_targets"`
	DeployPackages []DeployPackage `json:"deploy_packages"`
}

// ComposeDir 编排目录配置。
type ComposeDir struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// DeployTarget 前端发布目标配置。
type DeployTarget struct {
	Name      string `json:"name"`
	Container string `json:"container"`
	HtmlPath  string `json:"html_path"`
	BackupDir string `json:"backup_dir"`
}

// DeployPackage 发布包配置。
type DeployPackage struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// DefaultPath 返回默认配置文件路径 ~/.dtui/config.json。
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".dtui", "config.json")
}

// Load 加载配置文件。
func Load(path string) Config {
	cfg := Config{}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, &cfg)
	return cfg
}

// Save 保存配置文件。
func Save(path string, cfg Config) error {
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// InitDefault 初始化配置文件（如果不存在）。
func InitDefault(path string) {
	if _, err := os.Stat(path); err == nil {
		return
	}
	home, _ := os.UserHomeDir()
	composeDir := filepath.Join(home, ".dtui", "compose")
	backupDir := DefaultBackupDir("dtui-hello")
	os.MkdirAll(composeDir, 0755)
	os.MkdirAll(backupDir, 0755)
	cfg := Config{
		ComposeDirs: []ComposeDir{
			{Name: "测试项目", Path: composeDir},
		},
		DeployTargets: []DeployTarget{
			{Name: "测试部署", Container: "dtui-hello", HtmlPath: "/usr/share/nginx/html", BackupDir: backupDir},
		},
		DeployPackages: []DeployPackage{},
	}
	Save(path, cfg)
}

// AddComposeDir 添加编排目录并保存。
func AddComposeDir(path string, name, dirPath string) error {
	cfg := Load(path)
	cfg.ComposeDirs = append(cfg.ComposeDirs, ComposeDir{Name: name, Path: dirPath})
	return Save(path, cfg)
}

// RemoveComposeDir 删除编排目录并保存。
func RemoveComposeDir(path string, index int) error {
	cfg := Load(path)
	if index < 0 || index >= len(cfg.ComposeDirs) {
		return fmt.Errorf("无效索引")
	}
	cfg.ComposeDirs = append(cfg.ComposeDirs[:index], cfg.ComposeDirs[index+1:]...)
	return Save(path, cfg)
}

// AddDeployTarget 添加发布目标并保存。
func AddDeployTarget(path string, name, container, htmlPath, backupDir string) error {
	cfg := Load(path)
	cfg.DeployTargets = append(cfg.DeployTargets, DeployTarget{
		Name: name, Container: container, HtmlPath: htmlPath, BackupDir: backupDir,
	})
	return Save(path, cfg)
}

// RemoveDeployTarget 删除发布目标并保存。
func RemoveDeployTarget(path string, index int) error {
	cfg := Load(path)
	if index < 0 || index >= len(cfg.DeployTargets) {
		return fmt.Errorf("无效索引")
	}
	cfg.DeployTargets = append(cfg.DeployTargets[:index], cfg.DeployTargets[index+1:]...)
	return Save(path, cfg)
}

// AddDeployPackage 添加发布包并保存。
func AddDeployPackage(path string, name, pkgPath string) error {
	cfg := Load(path)
	cfg.DeployPackages = append(cfg.DeployPackages, DeployPackage{Name: name, Path: pkgPath})
	return Save(path, cfg)
}

// RemoveDeployPackage 删除发布包并保存。
func RemoveDeployPackage(path string, index int) error {
	cfg := Load(path)
	if index < 0 || index >= len(cfg.DeployPackages) {
		return fmt.Errorf("无效索引")
	}
	cfg.DeployPackages = append(cfg.DeployPackages[:index], cfg.DeployPackages[index+1:]...)
	return Save(path, cfg)
}

// HistoryEntry 操作历史条目。
type HistoryEntry struct {
	Time    time.Time `json:"time"`
	Action  string    `json:"action"`
	Target  string    `json:"target"`
	Detail  string    `json:"detail"`
	Success bool      `json:"success"`
	Error   string    `json:"error,omitempty"`
}

func HistoryPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".dtui", "history.json")
}

func LoadHistory(path string) []HistoryEntry {
	var entries []HistoryEntry
	data, err := os.ReadFile(path)
	if err != nil {
		return entries
	}
	json.Unmarshal(data, &entries)
	return entries
}

func SaveHistory(path string, entries []HistoryEntry) error {
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func RecordHistory(path string, entry HistoryEntry) {
	entries := LoadHistory(path)
	entries = append(entries, entry)
	if len(entries) > 200 {
		entries = entries[len(entries)-200:]
	}
	SaveHistory(path, entries)
}

func DefaultBackupDir(containerName string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".dtui", "backups", containerName)
}

func CleanOldBackups(backupDir string, keep int) {
	entries, err := os.ReadDir(backupDir)
	if err != nil || len(entries) <= keep {
		return
	}
	type entryInfo struct {
		name    string
		modTime time.Time
	}
	var infos []entryInfo
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		infos = append(infos, entryInfo{name: e.Name(), modTime: info.ModTime()})
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].modTime.Before(infos[j].modTime)
	})
	for i := 0; i < len(infos)-keep; i++ {
		os.RemoveAll(filepath.Join(backupDir, infos[i].name))
	}
}
