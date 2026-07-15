package updater

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDownloadVerifiedDownloadsZipAndVerifiesChecksum(t *testing.T) {
	zipBytes := []byte("zip bytes")
	checksum := sha256Hex(zipBytes)
	zipAsset := Asset{
		Name:        "onetiny-cli-linux-x64.zip",
		DownloadURL: "/cli.zip",
	}
	checksumAsset := Asset{
		Name:        "onetiny-checksums.txt",
		DownloadURL: "/checksums.txt",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cli.zip":
			if _, err := w.Write(zipBytes); err != nil {
				t.Fatalf("write zip response: %v", err)
			}
		case "/checksums.txt":
			fmt.Fprintf(w, "%s  %s\n", checksum, zipAsset.Name)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	zipAsset.DownloadURL = server.URL + zipAsset.DownloadURL
	checksumAsset.DownloadURL = server.URL + checksumAsset.DownloadURL
	release := Release{
		TagName: "v1.2.3",
		Assets:  []Asset{zipAsset, checksumAsset},
	}
	dir := t.TempDir()

	got, err := DownloadVerified(context.Background(), DownloadOptions{
		Client:  server.Client(),
		Release: release,
		Asset:   zipAsset,
		Dir:     dir,
	})
	if err != nil {
		t.Fatalf("DownloadVerified returned error: %v", err)
	}
	if got.Release.TagName != release.TagName {
		t.Fatalf("result release = %q, want %q", got.Release.TagName, release.TagName)
	}
	if got.Asset.Name != zipAsset.Name {
		t.Fatalf("result asset = %q, want %q", got.Asset.Name, zipAsset.Name)
	}
	if got.Checksum != checksum {
		t.Fatalf("checksum = %q, want %q", got.Checksum, checksum)
	}
	if filepath.Dir(got.ZipPath) != dir {
		t.Fatalf("zip dir = %q, want %q", filepath.Dir(got.ZipPath), dir)
	}
	if filepath.Base(got.ZipPath) != zipAsset.Name {
		t.Fatalf("zip file = %q, want %q", filepath.Base(got.ZipPath), zipAsset.Name)
	}
	gotBytes, err := os.ReadFile(got.ZipPath)
	if err != nil {
		t.Fatalf("read zip file: %v", err)
	}
	if !bytes.Equal(gotBytes, zipBytes) {
		t.Fatalf("zip bytes = %q, want %q", gotBytes, zipBytes)
	}
}

func TestDownloadVerifiedKeepsOwnedTempDirOnSuccess(t *testing.T) {
	tempRoot := t.TempDir()
	t.Setenv("TMPDIR", tempRoot)
	zipBytes := []byte("zip bytes")
	checksum := sha256Hex(zipBytes)
	zipAsset := Asset{
		Name:        "onetiny-cli-linux-x64.zip",
		DownloadURL: "/cli.zip",
	}
	checksumAsset := Asset{
		Name:        "onetiny-checksums.txt",
		DownloadURL: "/checksums.txt",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cli.zip":
			if _, err := w.Write(zipBytes); err != nil {
				t.Fatalf("write zip response: %v", err)
			}
		case "/checksums.txt":
			fmt.Fprintf(w, "%s  %s\n", checksum, zipAsset.Name)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	zipAsset.DownloadURL = server.URL + zipAsset.DownloadURL
	checksumAsset.DownloadURL = server.URL + checksumAsset.DownloadURL

	got, err := DownloadVerified(context.Background(), DownloadOptions{
		Client:  server.Client(),
		Release: Release{TagName: "v1.2.3", Assets: []Asset{zipAsset, checksumAsset}},
		Asset:   zipAsset,
	})
	if err != nil {
		t.Fatalf("DownloadVerified returned error: %v", err)
	}

	downloadDir := filepath.Dir(got.ZipPath)
	defer func() {
		if err := os.RemoveAll(downloadDir); err != nil {
			t.Fatalf("remove owned temp dir: %v", err)
		}
	}()
	if _, err := os.Stat(downloadDir); err != nil {
		t.Fatalf("owned temp dir stat failed: %v", err)
	}
	if filepath.Dir(downloadDir) != tempRoot {
		t.Fatalf("owned temp dir parent = %q, want %q", filepath.Dir(downloadDir), tempRoot)
	}
	gotBytes, err := os.ReadFile(got.ZipPath)
	if err != nil {
		t.Fatalf("read zip file: %v", err)
	}
	if !bytes.Equal(gotBytes, zipBytes) {
		t.Fatalf("zip bytes = %q, want %q", gotBytes, zipBytes)
	}
}

func TestDownloadVerifiedReturnsChecksumMismatch(t *testing.T) {
	zipBytes := []byte("zip bytes")
	zipAsset := Asset{
		Name:        "onetiny-cli-linux-x64.zip",
		DownloadURL: "/cli.zip",
	}
	checksumAsset := Asset{
		Name:        "onetiny-checksums.txt",
		DownloadURL: "/checksums.txt",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cli.zip":
			if _, err := w.Write(zipBytes); err != nil {
				t.Fatalf("write zip response: %v", err)
			}
		case "/checksums.txt":
			fmt.Fprintf(w, "%s  %s\n", sha256Hex([]byte("different")), zipAsset.Name)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	zipAsset.DownloadURL = server.URL + zipAsset.DownloadURL
	checksumAsset.DownloadURL = server.URL + checksumAsset.DownloadURL

	dir := t.TempDir()

	_, err := DownloadVerified(context.Background(), DownloadOptions{
		Client:  server.Client(),
		Release: Release{TagName: "v1.2.3", Assets: []Asset{zipAsset, checksumAsset}},
		Asset:   zipAsset,
		Dir:     dir,
	})
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("DownloadVerified error = %v, want %v", err, ErrChecksumMismatch)
	}
	if _, statErr := os.Stat(filepath.Join(dir, zipAsset.Name)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("final zip exists after checksum mismatch: %v", statErr)
	}
	assertNoDownloadTempFiles(t, dir)
	if _, statErr := os.Stat(dir); statErr != nil {
		t.Fatalf("caller-provided dir was removed: %v", statErr)
	}
}

func TestDownloadVerifiedReturnsChecksumNotFound(t *testing.T) {
	zipBytes := []byte("zip bytes")
	zipAsset := Asset{
		Name:        "onetiny-cli-linux-x64.zip",
		DownloadURL: "/cli.zip",
	}
	checksumAsset := Asset{
		Name:        "onetiny-checksums.txt",
		DownloadURL: "/checksums.txt",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cli.zip":
			if _, err := w.Write(zipBytes); err != nil {
				t.Fatalf("write zip response: %v", err)
			}
		case "/checksums.txt":
			fmt.Fprintf(w, "%s  other.zip\n", sha256Hex(zipBytes))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	zipAsset.DownloadURL = server.URL + zipAsset.DownloadURL
	checksumAsset.DownloadURL = server.URL + checksumAsset.DownloadURL

	_, err := DownloadVerified(context.Background(), DownloadOptions{
		Client:  server.Client(),
		Release: Release{TagName: "v1.2.3", Assets: []Asset{zipAsset, checksumAsset}},
		Asset:   zipAsset,
		Dir:     t.TempDir(),
	})
	if !errors.Is(err, ErrChecksumNotFound) {
		t.Fatalf("DownloadVerified error = %v, want %v", err, ErrChecksumNotFound)
	}
}

func TestDownloadVerifiedCleansOwnedTempDirWhenChecksumsAssetMissing(t *testing.T) {
	tempRoot := t.TempDir()
	t.Setenv("TMPDIR", tempRoot)
	zipAsset := Asset{
		Name:        "onetiny-cli-linux-x64.zip",
		DownloadURL: "https://example.test/cli.zip",
	}

	_, err := DownloadVerified(context.Background(), DownloadOptions{
		Release: Release{TagName: "v1.2.3", Assets: []Asset{zipAsset}},
		Asset:   zipAsset,
	})
	if !errors.Is(err, ErrChecksumNotFound) {
		t.Fatalf("DownloadVerified error = %v, want %v", err, ErrChecksumNotFound)
	}

	matches, err := filepath.Glob(filepath.Join(tempRoot, "onetiny-update-*"))
	if err != nil {
		t.Fatalf("glob temp update dirs: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("owned temp dirs left behind: %#v", matches)
	}
}

func TestDownloadVerifiedRemovesFinalZipWhenDownloadFails(t *testing.T) {
	zipBytes := []byte("zip bytes")
	zipAsset := Asset{
		Name:        "onetiny-cli-linux-x64.zip",
		DownloadURL: "/cli.zip",
	}
	checksumAsset := Asset{
		Name:        "onetiny-checksums.txt",
		DownloadURL: "/checksums.txt",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cli.zip":
			w.Header().Set("Content-Length", "100")
			if _, err := w.Write([]byte("partial")); err != nil {
				t.Fatalf("write partial zip response: %v", err)
			}
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case "/checksums.txt":
			fmt.Fprintf(w, "%s  %s\n", sha256Hex(zipBytes), zipAsset.Name)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	zipAsset.DownloadURL = server.URL + zipAsset.DownloadURL
	checksumAsset.DownloadURL = server.URL + checksumAsset.DownloadURL
	dir := t.TempDir()

	_, err := DownloadVerified(context.Background(), DownloadOptions{
		Client:  server.Client(),
		Release: Release{TagName: "v1.2.3", Assets: []Asset{zipAsset, checksumAsset}},
		Asset:   zipAsset,
		Dir:     dir,
	})
	if err == nil {
		t.Fatal("DownloadVerified returned nil error, want failed download")
	}
	if _, statErr := os.Stat(filepath.Join(dir, zipAsset.Name)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("final zip exists after failed download: %v", statErr)
	}
	assertNoDownloadTempFiles(t, dir)
}

func TestDownloadUsesTimeoutDefaultHTTPClient(t *testing.T) {
	got := httpClient(nil)
	if got == http.DefaultClient {
		t.Fatal("default client = http.DefaultClient, want package client with timeout")
	}
	if got.Timeout != 30*time.Second {
		t.Fatalf("default timeout = %s, want 30s", got.Timeout)
	}

	custom := &http.Client{}
	if got := httpClient(custom); got != custom {
		t.Fatal("custom HTTP client was not used")
	}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func assertNoDownloadTempFiles(t *testing.T, dir string) {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(dir, ".*.tmp-*"))
	if err != nil {
		t.Fatalf("glob temp download files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("temp download files left behind: %#v", matches)
	}
}
