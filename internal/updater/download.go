package updater

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const checksumsAssetName = "onetiny-checksums.txt"

type DownloadOptions struct {
	Client  *http.Client
	Release Release
	Asset   Asset
	Dir     string
}

type DownloadResult struct {
	Release  Release
	Asset    Asset
	ZipPath  string
	Checksum string
}

func DownloadVerified(ctx context.Context, opts DownloadOptions) (result DownloadResult, err error) {
	dir := opts.Dir
	ownedDir := false
	if dir == "" {
		dir, err = os.MkdirTemp("", "onetiny-update-*")
		if err != nil {
			return DownloadResult{}, fmt.Errorf("create update download dir: %w", err)
		}
		ownedDir = true
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		return DownloadResult{}, fmt.Errorf("create update download dir: %w", err)
	}
	if ownedDir {
		defer func() {
			if err != nil {
				_ = os.RemoveAll(dir)
			}
		}()
	}

	checksumAsset, ok := findChecksumAsset(opts.Release)
	if !ok {
		return DownloadResult{}, fmt.Errorf("%w: %s", ErrChecksumNotFound, checksumsAssetName)
	}

	checksumPath := filepath.Join(dir, checksumsAssetName)
	if err := downloadFileAtomic(ctx, opts.Client, checksumAsset.DownloadURL, checksumPath); err != nil {
		return DownloadResult{}, err
	}

	checksums, err := os.ReadFile(checksumPath)
	if err != nil {
		return DownloadResult{}, fmt.Errorf("read checksums file: %w", err)
	}

	expectedChecksum, ok := checksumForAsset(string(checksums), opts.Asset.Name)
	if !ok {
		return DownloadResult{}, fmt.Errorf("%w: %s", ErrChecksumNotFound, opts.Asset.Name)
	}

	zipPath := filepath.Join(dir, filepath.Base(opts.Asset.Name))
	tmpZipPath, actualChecksum, err := downloadFileTemp(ctx, opts.Client, opts.Asset.DownloadURL, zipPath, sha256.New())
	if err != nil {
		return DownloadResult{}, err
	}
	if !strings.EqualFold(actualChecksum, expectedChecksum) {
		_ = os.Remove(tmpZipPath)
		return DownloadResult{}, fmt.Errorf("%w: %s", ErrChecksumMismatch, opts.Asset.Name)
	}
	if err := renameDownload(tmpZipPath, zipPath); err != nil {
		_ = os.Remove(tmpZipPath)
		return DownloadResult{}, err
	}

	return DownloadResult{
		Release:  opts.Release,
		Asset:    opts.Asset,
		ZipPath:  zipPath,
		Checksum: expectedChecksum,
	}, nil
}

func findChecksumAsset(release Release) (Asset, bool) {
	for _, asset := range release.Assets {
		if asset.Name == checksumsAssetName {
			return asset, true
		}
	}
	return Asset{}, false
}

func checksumForAsset(checksums, assetName string) (string, bool) {
	scanner := bufio.NewScanner(strings.NewReader(checksums))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		filename := strings.TrimPrefix(fields[1], "*")
		if filename == assetName {
			return fields[0], true
		}
	}
	return "", false
}

func downloadFileAtomic(ctx context.Context, client *http.Client, url, finalPath string) error {
	tmpPath, _, err := downloadFileTemp(ctx, client, url, finalPath, nil)
	if err != nil {
		return err
	}
	if err := renameDownload(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func downloadFileTemp(ctx context.Context, client *http.Client, url, finalPath string, hasher hash.Hash) (tmpPath, checksum string, err error) {
	resp, err := download(ctx, client, url)
	if err != nil {
		return "", "", err
	}
	body := resp.Body
	defer func() {
		if body != nil {
			err = joinCloseError(err, "close download response", body.Close())
		}
		if err != nil && tmpPath != "" {
			_ = os.Remove(tmpPath)
		}
	}()

	file, err := os.CreateTemp(filepath.Dir(finalPath), "."+filepath.Base(finalPath)+".tmp-*")
	if err != nil {
		return "", "", fmt.Errorf("create temp download file: %w", err)
	}
	tmpPath = file.Name()

	writer := io.Writer(file)
	if hasher != nil {
		writer = io.MultiWriter(file, hasher)
	}
	if _, copyErr := io.Copy(writer, body); copyErr != nil {
		err = fmt.Errorf("write download file: %w", copyErr)
	}
	if err == nil {
		err = joinCloseError(err, "flush download file", file.Sync())
	}
	err = joinCloseError(err, "close download file", file.Close())
	if err != nil {
		return
	}

	err = joinCloseError(err, "close download response", body.Close())
	body = nil
	if err != nil {
		return
	}

	if hasher != nil {
		checksum = hashHex(hasher)
	}
	return tmpPath, checksum, nil
}

func renameDownload(tmpPath, finalPath string) error {
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("rename download file: %w", err)
	}
	return nil
}

func download(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}

	resp, err := httpClient(client).Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		if len(body) > 0 {
			return nil, fmt.Errorf("download request failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}
		return nil, fmt.Errorf("download request failed: %s", resp.Status)
	}
	return resp, nil
}

func httpClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return defaultHTTPClient
}

func hashHex(hasher hash.Hash) string {
	return hex.EncodeToString(hasher.Sum(nil))
}

func joinCloseError(err error, operation string, closeErr error) error {
	if closeErr == nil {
		return err
	}
	wrapped := fmt.Errorf("%s: %w", operation, closeErr)
	if err != nil {
		return errors.Join(err, wrapped)
	}
	return wrapped
}
