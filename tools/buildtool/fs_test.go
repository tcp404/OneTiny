package main

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
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

func TestCopyFileReplacesExistingDestination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows file replacement semantics differ")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "source")
	dst := filepath.Join(dir, "target")

	if err := os.WriteFile(src, []byte("new"), 0o755); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(dst, []byte("old"), 0o644); err != nil {
		t.Fatalf("write destination: %v", err)
	}
	beforeInode := fileInode(t, dst)

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("copy file: %v", err)
	}

	body, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(body) != "new" {
		t.Fatalf("destination body = %q, want new", body)
	}
	if afterInode := fileInode(t, dst); afterInode == beforeInode {
		t.Fatalf("destination inode = %d before and after copy, want replacement", afterInode)
	}
}

func fileInode(t *testing.T, path string) uint64 {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %q: %v", path, err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("stat sys type = %T, want *syscall.Stat_t", info.Sys())
	}
	return stat.Ino
}
