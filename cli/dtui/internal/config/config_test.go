package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := Config{
		ComposeDirs: []ComposeDir{{Name: "test", Path: "/tmp/test"}},
		DeployTargets: []DeployTarget{
			{Name: "prod", Container: "web", HtmlPath: "/usr/share/nginx/html", BackupDir: "/tmp/backups"},
		},
		DeployPackages: []DeployPackage{{Name: "v1", Path: "/tmp/pkg.zip"}},
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	loaded := Load(path)
	if len(loaded.ComposeDirs) != 1 {
		t.Errorf("expected 1 compose dir, got %d", len(loaded.ComposeDirs))
	}
	if loaded.ComposeDirs[0].Name != "test" {
		t.Errorf("expected compose name 'test', got %q", loaded.ComposeDirs[0].Name)
	}
	if len(loaded.DeployTargets) != 1 {
		t.Errorf("expected 1 deploy target, got %d", len(loaded.DeployTargets))
	}
	if loaded.DeployTargets[0].Container != "web" {
		t.Errorf("expected container 'web', got %q", loaded.DeployTargets[0].Container)
	}
	if len(loaded.DeployPackages) != 1 {
		t.Errorf("expected 1 deploy package, got %d", len(loaded.DeployPackages))
	}
}

func TestLoadNonexistent(t *testing.T) {
	cfg := Load("/nonexistent/path/config.json")
	if len(cfg.ComposeDirs) != 0 {
		t.Error("expected empty config for nonexistent file")
	}
}

func TestAddRemoveComposeDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	Save(path, Config{})

	if err := AddComposeDir(path, "c1", "/tmp/c1"); err != nil {
		t.Fatalf("AddComposeDir() failed: %v", err)
	}
	if err := AddComposeDir(path, "c2", "/tmp/c2"); err != nil {
		t.Fatalf("AddComposeDir() failed: %v", err)
	}

	cfg := Load(path)
	if len(cfg.ComposeDirs) != 2 {
		t.Fatalf("expected 2 compose dirs, got %d", len(cfg.ComposeDirs))
	}
	if cfg.ComposeDirs[0].Name != "c1" || cfg.ComposeDirs[1].Name != "c2" {
		t.Error("compose dir names mismatch")
	}

	if err := RemoveComposeDir(path, 0); err != nil {
		t.Fatalf("RemoveComposeDir() failed: %v", err)
	}
	cfg = Load(path)
	if len(cfg.ComposeDirs) != 1 {
		t.Fatalf("expected 1 compose dir after remove, got %d", len(cfg.ComposeDirs))
	}
	if cfg.ComposeDirs[0].Name != "c2" {
		t.Errorf("expected remaining dir 'c2', got %q", cfg.ComposeDirs[0].Name)
	}
}

func TestRemoveComposeDirInvalidIndex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	Save(path, Config{})

	if err := RemoveComposeDir(path, 0); err == nil {
		t.Error("expected error for invalid index on empty config")
	}
	if err := RemoveComposeDir(path, -1); err == nil {
		t.Error("expected error for negative index")
	}
}

func TestAddRemoveDeployTarget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	Save(path, Config{})

	if err := AddDeployTarget(path, "t1", "container1", "/html", "/backup"); err != nil {
		t.Fatalf("AddDeployTarget() failed: %v", err)
	}

	cfg := Load(path)
	if len(cfg.DeployTargets) != 1 {
		t.Fatalf("expected 1 deploy target, got %d", len(cfg.DeployTargets))
	}
	if cfg.DeployTargets[0].Name != "t1" {
		t.Errorf("expected name 't1', got %q", cfg.DeployTargets[0].Name)
	}

	if err := RemoveDeployTarget(path, 0); err != nil {
		t.Fatalf("RemoveDeployTarget() failed: %v", err)
	}
	cfg = Load(path)
	if len(cfg.DeployTargets) != 0 {
		t.Errorf("expected 0 deploy targets after remove, got %d", len(cfg.DeployTargets))
	}
}

func TestRemoveDeployTargetInvalidIndex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	Save(path, Config{})

	if err := RemoveDeployTarget(path, 0); err == nil {
		t.Error("expected error for invalid index")
	}
}

func TestAddRemoveDeployPackage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	Save(path, Config{})

	if err := AddDeployPackage(path, "pkg1", "/tmp/pkg.zip"); err != nil {
		t.Fatalf("AddDeployPackage() failed: %v", err)
	}
	if err := AddDeployPackage(path, "pkg2", "/tmp/pkg2.zip"); err != nil {
		t.Fatalf("AddDeployPackage() failed: %v", err)
	}

	cfg := Load(path)
	if len(cfg.DeployPackages) != 2 {
		t.Fatalf("expected 2 deploy packages, got %d", len(cfg.DeployPackages))
	}

	if err := RemoveDeployPackage(path, 1); err != nil {
		t.Fatalf("RemoveDeployPackage() failed: %v", err)
	}
	cfg = Load(path)
	if len(cfg.DeployPackages) != 1 {
		t.Fatalf("expected 1 deploy package after remove, got %d", len(cfg.DeployPackages))
	}
	if cfg.DeployPackages[0].Name != "pkg1" {
		t.Errorf("expected remaining package 'pkg1', got %q", cfg.DeployPackages[0].Name)
	}
}

func TestRemoveDeployPackageInvalidIndex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	Save(path, Config{})

	if err := RemoveDeployPackage(path, 5); err == nil {
		t.Error("expected error for invalid index")
	}
}

func TestDefaultPath(t *testing.T) {
	p := DefaultPath()
	if p == "" {
		t.Error("DefaultPath() should not be empty")
	}
	if filepath.Ext(p) != ".json" {
		t.Errorf("expected .json extension, got %q", filepath.Ext(p))
	}
}

func TestDefaultBackupDir(t *testing.T) {
	d := DefaultBackupDir("mycontainer")
	if d == "" {
		t.Error("DefaultBackupDir() should not be empty")
	}
	if filepath.Base(d) != "mycontainer" {
		t.Errorf("expected base 'mycontainer', got %q", filepath.Base(d))
	}
}

func TestHistoryRecordLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	entry := HistoryEntry{
		Time:    time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC),
		Action:  "deploy",
		Target:  "test-target",
		Detail:  "/tmp/pkg.zip",
		Success: true,
	}
	RecordHistory(path, entry)

	entries := LoadHistory(path)
	if len(entries) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(entries))
	}
	if entries[0].Action != "deploy" {
		t.Errorf("expected action 'deploy', got %q", entries[0].Action)
	}
	if !entries[0].Success {
		t.Error("expected success=true")
	}
}

func TestHistoryRecordFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	RecordHistory(path, HistoryEntry{
		Time:    time.Now(),
		Action:  "deploy",
		Target:  "prod",
		Detail:  "/bad/path",
		Success: false,
		Error:   "docker not available",
	})

	entries := LoadHistory(path)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Success {
		t.Error("expected success=false")
	}
	if entries[0].Error != "docker not available" {
		t.Errorf("expected error 'docker not available', got %q", entries[0].Error)
	}
}

func TestHistoryMaxEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	for i := 0; i < 210; i++ {
		RecordHistory(path, HistoryEntry{
			Time:    time.Now(),
			Action:  "test",
			Target:  "t",
			Success: true,
		})
	}

	entries := LoadHistory(path)
	if len(entries) > 200 {
		t.Errorf("expected at most 200 entries, got %d", len(entries))
	}
}

func TestLoadHistoryNonexistent(t *testing.T) {
	entries := LoadHistory("/nonexistent/history.json")
	if len(entries) != 0 {
		t.Error("expected empty history for nonexistent file")
	}
}

func TestCleanOldBackups(t *testing.T) {
	dir := t.TempDir()

	for i := 0; i < 5; i++ {
		os.WriteFile(filepath.Join(dir, "backup"+string(rune('a'+i))), []byte("data"), 0644)
		time.Sleep(10 * time.Millisecond)
	}

	CleanOldBackups(dir, 3)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(entries) > 3 {
		t.Errorf("expected at most 3 entries after cleanup, got %d", len(entries))
	}
}

func TestCleanOldBackupsEmptyDir(t *testing.T) {
	dir := t.TempDir()
	CleanOldBackups(dir, 5)
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Error("expected empty dir after cleanup of empty dir")
	}
}

func TestInitDefaultCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "config.json")

	InitDefault(path)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("InitDefault should create config file")
	}

	cfg := Load(path)
	if len(cfg.ComposeDirs) == 0 {
		t.Error("expected default compose dirs")
	}
	if len(cfg.DeployTargets) == 0 {
		t.Error("expected default deploy targets")
	}
}

func TestInitDefaultDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	custom := Config{
		ComposeDirs: []ComposeDir{{Name: "custom", Path: "/custom"}},
	}
	Save(path, custom)

	InitDefault(path)

	cfg := Load(path)
	if len(cfg.ComposeDirs) != 1 || cfg.ComposeDirs[0].Name != "custom" {
		t.Error("InitDefault should not overwrite existing config")
	}
}
