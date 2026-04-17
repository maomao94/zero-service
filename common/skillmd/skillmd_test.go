package skillmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanDir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "demo_skill")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	md := `---
name: Demo
description: "hello"
tags: [a, b]
launch_prompt: "go"
---
body
`
	if err := os.WriteFile(filepath.Join(sub, "SKILL.md"), []byte(md), 0644); err != nil {
		t.Fatal(err)
	}
	infos, err := ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 1 || infos[0].Name != "Demo" {
		t.Fatalf("%+v", infos)
	}
}
