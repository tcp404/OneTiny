package updater

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type StageResult struct {
	StagingDir    string
	CandidatePath string
}

func StageArchive(zipPath string, channel Channel, platform Platform) (result StageResult, err error) {
	stagingDir, err := os.MkdirTemp("", "onetiny-stage-*")
	if err != nil {
		return StageResult{}, fmt.Errorf("create update staging dir: %w", err)
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(stagingDir)
		}
	}()

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return StageResult{}, fmt.Errorf("%w: open archive: %v", ErrInvalidArchive, err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		if err := extractZipFile(file, stagingDir); err != nil {
			return StageResult{}, err
		}
	}

	candidatePath, err := findCandidate(stagingDir, channel, platform)
	if err != nil {
		return StageResult{}, err
	}

	return StageResult{
		StagingDir:    stagingDir,
		CandidatePath: candidatePath,
	}, nil
}

func extractZipFile(file *zip.File, stagingDir string) error {
	target, err := zipEntryTarget(stagingDir, file.Name)
	if err != nil {
		return err
	}

	mode := file.FileInfo().Mode()
	if file.FileInfo().IsDir() {
		perm := mode.Perm()
		if perm == 0 {
			perm = 0o755
		}
		if err := os.MkdirAll(target, perm); err != nil {
			return fmt.Errorf("create archive directory: %w", err)
		}
		return os.Chmod(target, perm)
	}
	if !mode.IsRegular() {
		return fmt.Errorf("%w: unsupported archive entry %q", ErrInvalidArchive, file.Name)
	}

	perm := mode.Perm()
	if perm == 0 {
		perm = 0o644
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create archive parent directory: %w", err)
	}

	source, err := file.Open()
	if err != nil {
		return fmt.Errorf("open archive entry: %w", err)
	}

	dest, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return joinCloseError(fmt.Errorf("create archive file: %w", err), "close archive entry", source.Close())
	}

	_, copyErr := io.Copy(dest, source)
	closeSourceErr := source.Close()
	closeDestErr := dest.Close()
	var extractErr error
	if copyErr != nil {
		extractErr = fmt.Errorf("write archive file: %w", copyErr)
	}
	extractErr = joinCloseError(extractErr, "close archive entry", closeSourceErr)
	extractErr = joinCloseError(extractErr, "close archive file", closeDestErr)
	if extractErr != nil {
		return extractErr
	}
	if err := os.Chmod(target, perm); err != nil {
		return fmt.Errorf("chmod archive file: %w", err)
	}
	return nil
}

func zipEntryTarget(stagingDir, name string) (string, error) {
	normalized := strings.ReplaceAll(name, "\\", "/")
	if normalized == "" || strings.HasPrefix(normalized, "/") || strings.HasPrefix(normalized, "//") || hasWindowsVolume(normalized) {
		return "", fmt.Errorf("%w: unsafe archive path %q", ErrInvalidArchive, name)
	}

	clean := path.Clean(normalized)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("%w: unsafe archive path %q", ErrInvalidArchive, name)
	}

	target := filepath.Join(stagingDir, filepath.FromSlash(clean))
	rel, err := filepath.Rel(stagingDir, target)
	if err != nil {
		return "", fmt.Errorf("resolve archive path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: unsafe archive path %q", ErrInvalidArchive, name)
	}
	return target, nil
}

func hasWindowsVolume(name string) bool {
	if len(name) < 2 || name[1] != ':' {
		return false
	}
	drive := name[0]
	return (drive >= 'A' && drive <= 'Z') || (drive >= 'a' && drive <= 'z')
}

func findCandidate(stagingDir string, channel Channel, platform Platform) (string, error) {
	osName := strings.ToLower(strings.TrimSpace(platform.OS))
	arch := normalizeArch(platform.Arch)
	if !isReleasedAsset(channel, osName, arch) {
		return "", fmt.Errorf("%w: %s %s/%s", ErrUnsupportedPlatform, channel, osName, arch)
	}

	switch channel {
	case ChannelCLI:
		name := "onetiny-cli"
		if osName == "windows" {
			name += ".exe"
		}
		return requireRegularCandidate(filepath.Join(stagingDir, name), name)
	case ChannelGUI:
		switch osName {
		case "windows":
			return requireRegularCandidate(filepath.Join(stagingDir, "OneTiny.exe"), "OneTiny.exe")
		case "darwin":
			return requireAppBundleCandidate(filepath.Join(stagingDir, "OneTiny.app"), "OneTiny.app")
		}
	}
	return "", fmt.Errorf("%w: %s %s/%s", ErrUnsupportedPlatform, channel, osName, arch)
}

func requireRegularCandidate(candidatePath, name string) (string, error) {
	info, err := os.Stat(candidatePath)
	if err != nil {
		return "", fmt.Errorf("%w: missing candidate %s", ErrInvalidArchive, name)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%w: candidate %s is not a file", ErrInvalidArchive, name)
	}
	return candidatePath, nil
}

func requireAppBundleCandidate(candidatePath, name string) (string, error) {
	if _, err := requireDirectoryCandidate(candidatePath, name); err != nil {
		return "", err
	}
	requiredFiles := []string{
		filepath.Join("Contents", "Info.plist"),
		filepath.Join("Contents", "MacOS", "OneTiny"),
	}
	for _, requiredFile := range requiredFiles {
		if _, err := requireRegularCandidate(filepath.Join(candidatePath, requiredFile), filepath.Join(name, requiredFile)); err != nil {
			return "", err
		}
	}
	return candidatePath, nil
}

func requireDirectoryCandidate(candidatePath, name string) (string, error) {
	info, err := os.Stat(candidatePath)
	if err != nil {
		return "", fmt.Errorf("%w: missing candidate %s", ErrInvalidArchive, name)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%w: candidate %s is not a directory", ErrInvalidArchive, name)
	}
	return candidatePath, nil
}
