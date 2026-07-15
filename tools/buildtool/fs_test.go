package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCopyFilePreservesSourcePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows does not use POSIX executable bits")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "source")
	dst := filepath.Join(dir, "nested", "target")

	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.Chmod(src, 0o755); err != nil {
		t.Fatalf("chmod source: %v", err)
	}

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("copy file: %v", err)
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("stat destination: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Fatalf("destination mode = %o, want 755", got)
	}
}
