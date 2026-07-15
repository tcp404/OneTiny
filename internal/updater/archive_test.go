package updater

import (
	"archive/zip"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestStageArchiveFindsCLIDarwinCandidate(t *testing.T) {
	zipPath := createTestZip(t, []zipEntry{
		{name: "onetiny-cli", body: "cli", mode: 0o755},
	})

	got, err := StageArchive(zipPath, ChannelCLI, Platform{OS: "darwin", Arch: "arm64"})
	if err != nil {
		t.Fatalf("StageArchive returned error: %v", err)
	}
	defer removeStagingDir(t, got.StagingDir)

	if filepath.Base(got.CandidatePath) != "onetiny-cli" {
		t.Fatalf("candidate basename = %q, want onetiny-cli", filepath.Base(got.CandidatePath))
	}
	if filepath.Dir(got.CandidatePath) != got.StagingDir {
		t.Fatalf("candidate dir = %q, want staging dir %q", filepath.Dir(got.CandidatePath), got.StagingDir)
	}
}

func TestStageArchiveFindsCLIWindowsCandidate(t *testing.T) {
	zipPath := createTestZip(t, []zipEntry{
		{name: "onetiny-cli.exe", body: "cli", mode: 0o755},
	})

	got, err := StageArchive(zipPath, ChannelCLI, Platform{OS: "windows", Arch: "x64"})
	if err != nil {
		t.Fatalf("StageArchive returned error: %v", err)
	}
	defer removeStagingDir(t, got.StagingDir)

	if filepath.Base(got.CandidatePath) != "onetiny-cli.exe" {
		t.Fatalf("candidate basename = %q, want onetiny-cli.exe", filepath.Base(got.CandidatePath))
	}
}

func TestStageArchiveFindsGUIDarwinAppCandidate(t *testing.T) {
	zipPath := createTestZip(t, []zipEntry{
		{name: "OneTiny.app/Contents/MacOS/OneTiny", body: "gui", mode: 0o755},
		{name: "OneTiny.app/Contents/Info.plist", body: "plist", mode: 0o644},
	})

	got, err := StageArchive(zipPath, ChannelGUI, Platform{OS: "darwin", Arch: "arm64"})
	if err != nil {
		t.Fatalf("StageArchive returned error: %v", err)
	}
	defer removeStagingDir(t, got.StagingDir)

	if filepath.Base(got.CandidatePath) != "OneTiny.app" {
		t.Fatalf("candidate basename = %q, want OneTiny.app", filepath.Base(got.CandidatePath))
	}
	if filepath.Dir(got.CandidatePath) != got.StagingDir {
		t.Fatalf("candidate dir = %q, want staging dir %q", filepath.Dir(got.CandidatePath), got.StagingDir)
	}
}

func TestStageArchiveRejectsInvalidGUIDarwinAppBundle(t *testing.T) {
	tests := []struct {
		name    string
		entries []zipEntry
	}{
		{
			name: "empty app directory",
			entries: []zipEntry{
				{name: "OneTiny.app/", mode: fs.ModeDir | 0o755},
			},
		},
		{
			name: "missing executable",
			entries: []zipEntry{
				{name: "OneTiny.app/Contents/Info.plist", body: "plist", mode: 0o644},
			},
		},
		{
			name: "missing info plist",
			entries: []zipEntry{
				{name: "OneTiny.app/Contents/MacOS/OneTiny", body: "gui", mode: 0o755},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempRoot := t.TempDir()
			t.Setenv("TMPDIR", tempRoot)
			zipPath := createTestZip(t, tt.entries)

			_, err := StageArchive(zipPath, ChannelGUI, Platform{OS: "darwin", Arch: "arm64"})
			if !errors.Is(err, ErrInvalidArchive) {
				t.Fatalf("StageArchive error = %v, want %v", err, ErrInvalidArchive)
			}
			assertNoStagingDirs(t, tempRoot)
		})
	}
}

func TestStageArchiveFindsGUIWindowsCandidate(t *testing.T) {
	zipPath := createTestZip(t, []zipEntry{
		{name: "OneTiny.exe", body: "gui", mode: 0o755},
	})

	got, err := StageArchive(zipPath, ChannelGUI, Platform{OS: "windows", Arch: "x64"})
	if err != nil {
		t.Fatalf("StageArchive returned error: %v", err)
	}
	defer removeStagingDir(t, got.StagingDir)

	if filepath.Base(got.CandidatePath) != "OneTiny.exe" {
		t.Fatalf("candidate basename = %q, want OneTiny.exe", filepath.Base(got.CandidatePath))
	}
}

func TestStageArchiveRejectsZipSlipWithoutWritingOutsideStaging(t *testing.T) {
	tempRoot := t.TempDir()
	t.Setenv("TMPDIR", tempRoot)
	zipPath := createTestZip(t, []zipEntry{
		{name: "../evil", body: "bad", mode: 0o644},
	})

	_, err := StageArchive(zipPath, ChannelCLI, Platform{OS: "darwin", Arch: "arm64"})
	if !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("StageArchive error = %v, want %v", err, ErrInvalidArchive)
	}
	if _, statErr := os.Stat(filepath.Join(tempRoot, "evil")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("zip slip wrote outside staging: %v", statErr)
	}
	assertNoStagingDirs(t, tempRoot)
}

func TestStageArchiveReturnsInvalidArchiveWhenCandidateMissing(t *testing.T) {
	tempRoot := t.TempDir()
	t.Setenv("TMPDIR", tempRoot)
	zipPath := createTestZip(t, []zipEntry{
		{name: "README.txt", body: "not a release artifact", mode: 0o644},
	})

	_, err := StageArchive(zipPath, ChannelCLI, Platform{OS: "darwin", Arch: "arm64"})
	if !errors.Is(err, ErrInvalidArchive) {
		t.Fatalf("StageArchive error = %v, want %v", err, ErrInvalidArchive)
	}
	assertNoStagingDirs(t, tempRoot)
}

type zipEntry struct {
	name string
	body string
	mode fs.FileMode
}

func createTestZip(t *testing.T, entries []zipEntry) string {
	t.Helper()

	zipPath := filepath.Join(t.TempDir(), "release.zip")
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	writer := zip.NewWriter(file)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name}
		if entry.mode != 0 {
			header.SetMode(entry.mode)
		}
		w, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatalf("create zip entry %q: %v", entry.name, err)
		}
		if entry.mode.IsDir() {
			continue
		}
		if _, err := w.Write([]byte(entry.body)); err != nil {
			t.Fatalf("write zip entry %q: %v", entry.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}
	return zipPath
}

func removeStagingDir(t *testing.T, dir string) {
	t.Helper()

	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("remove staging dir: %v", err)
	}
}

func assertNoStagingDirs(t *testing.T, root string) {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(root, "onetiny-stage-*"))
	if err != nil {
		t.Fatalf("glob staging dirs: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("staging dirs left behind: %#v", matches)
	}
}
