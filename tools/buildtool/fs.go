package main

import (
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func EnsureDirs(paths []string) error {
	for _, path := range paths {
		if path == "" {
			return errors.New("directory path is empty")
		}
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func CopyFile(src string, dst string) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()

	info, err := input.Stat()
	if err != nil {
		return err
	}
	mode := info.Mode().Perm()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	output, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.Chmod(dst, mode); err != nil {
		return err
	}
	return nil
}

func VerifyPath(path string, kind string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	switch kind {
	case "any", "":
		return nil
	case "file":
		if info.IsDir() {
			return errors.Errorf("%s is a directory, want file", path)
		}
	case "dir":
		if !info.IsDir() {
			return errors.Errorf("%s is a file, want directory", path)
		}
	default:
		return errors.Errorf("unsupported verify kind %q", kind)
	}
	return nil
}

func RemovePaths(paths []string) error {
	for _, path := range paths {
		if path == "" {
			return errors.New("remove path is empty")
		}
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}
	return nil
}
