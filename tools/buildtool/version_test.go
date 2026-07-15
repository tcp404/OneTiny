package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateVersionAcceptsReleaseTags(t *testing.T) {
	tests := []string{
		"v0.6.0",
		"v1.2.3",
		"v1.2.3-alpha.1",
	}

	for _, version := range tests {
		t.Run(version, func(t *testing.T) {
			if err := ValidateVersion(version); err != nil {
				t.Fatalf("ValidateVersion(%q) returned error: %v", version, err)
			}
		})
	}
}

func TestValidateVersionRejectsAmbiguousVersions(t *testing.T) {
	tests := []string{
		"",
		"0.6.0",
		"v1",
		"latest",
		"refs/tags/v0.6.0",
	}

	for _, version := range tests {
		t.Run(version, func(t *testing.T) {
			if err := ValidateVersion(version); err == nil {
				t.Fatalf("ValidateVersion(%q) returned nil error", version)
			}
		})
	}
}

func TestDefaultVersionReadsReleasePleaseManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".release-please-manifest.json"), []byte(`{".":"0.11.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	got, err := DefaultVersion()
	if err != nil {
		t.Fatalf("DefaultVersion returned error: %v", err)
	}
	if got != "v0.11.0" {
		t.Fatalf("DefaultVersion = %q, want v0.11.0", got)
	}
}
