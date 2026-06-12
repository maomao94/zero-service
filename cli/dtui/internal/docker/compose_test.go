package docker

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestPathTypeFolder(t *testing.T) {
	dir := t.TempDir()
	if got := PathType(dir); got != "folder" {
		t.Errorf("PathType(%q) = %q, want %q", dir, got, "folder")
	}
}

func TestPathTypeZip(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "test.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if got := PathType(zipPath); got != "zip" {
		t.Errorf("PathType(%q) = %q, want %q", zipPath, got, "zip")
	}
}

func TestPathTypeZipCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "test.ZIP")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if got := PathType(zipPath); got != "zip" {
		t.Errorf("PathType(%q) = %q, want %q", zipPath, got, "zip")
	}
}

func TestPathTypeUnknown(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := PathType(filePath); got != "unknown" {
		t.Errorf("PathType(%q) = %q, want %q", filePath, got, "unknown")
	}
}

func TestPathTypeInvalid(t *testing.T) {
	if got := PathType("/nonexistent/path"); got != "invalid" {
		t.Errorf("PathType(nonexistent) = %q, want %q", got, "invalid")
	}
}

func TestUnzipToDir(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "test.zip")
	destDir := filepath.Join(dir, "extracted")

	// Create a zip with a file and a subdirectory.
	zf, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(zf)
	fw, err := w.Create("hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte("world")); err != nil {
		t.Fatal(err)
	}
	dw, err := w.Create("subdir/nested.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dw.Write([]byte("deep")); err != nil {
		t.Fatal(err)
	}
	w.Close()
	zf.Close()

	if err := UnzipToDir(zipPath, destDir); err != nil {
		t.Fatalf("UnzipToDir() failed: %v", err)
	}

	// Verify extracted content.
	data, err := os.ReadFile(filepath.Join(destDir, "hello.txt"))
	if err != nil {
		t.Fatalf("ReadFile hello.txt: %v", err)
	}
	if string(data) != "world" {
		t.Errorf("hello.txt content = %q, want %q", string(data), "world")
	}

	data, err = os.ReadFile(filepath.Join(destDir, "subdir", "nested.txt"))
	if err != nil {
		t.Fatalf("ReadFile subdir/nested.txt: %v", err)
	}
	if string(data) != "deep" {
		t.Errorf("nested.txt content = %q, want %q", string(data), "deep")
	}
}

func TestUnzipToDirInvalidZip(t *testing.T) {
	dir := t.TempDir()
	badZip := filepath.Join(dir, "bad.zip")
	if err := os.WriteFile(badZip, []byte("not a zip"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := UnzipToDir(badZip, filepath.Join(dir, "out")); err == nil {
		t.Error("expected error for invalid zip file")
	}
}

func TestUnzipToDirNonexistentZip(t *testing.T) {
	dir := t.TempDir()
	if err := UnzipToDir("/nonexistent/file.zip", dir); err == nil {
		t.Error("expected error for nonexistent zip file")
	}
}

func TestUnzipToDirPreservesDirectoryStructure(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "nested.zip")
	destDir := filepath.Join(dir, "out")

	zf, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(zf)
	// Create deeply nested file.
	fw, err := w.Create("a/b/c/deep.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte("deep content")); err != nil {
		t.Fatal(err)
	}
	w.Close()
	zf.Close()

	if err := UnzipToDir(zipPath, destDir); err != nil {
		t.Fatalf("UnzipToDir() failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(destDir, "a", "b", "c", "deep.txt"))
	if err != nil {
		t.Fatalf("ReadFile a/b/c/deep.txt: %v", err)
	}
	if string(data) != "deep content" {
		t.Errorf("deep.txt content = %q, want %q", string(data), "deep content")
	}
}
