package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServiceCheckLatestFindsAssetAndAvailability(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/tcp404/OneTiny/releases/latest" {
			t.Fatalf("path = %s, want latest release path", r.URL.Path)
		}

		writeReleaseJSON(t, w, "v0.12.0", []Asset{
			{Name: "onetiny-cli-darwin-arm64.zip", DownloadURL: "https://example.test/cli.zip"},
		})
	}))
	defer server.Close()

	service := Service{Client: Client{BaseURL: server.URL, HTTPClient: server.Client()}}
	platform := Platform{OS: "darwin", Arch: "arm64"}

	got, err := service.CheckLatest(context.Background(), CheckOptions{
		Channel:        ChannelCLI,
		CurrentVersion: "v0.11.0",
		Platform:       platform,
	})
	if err != nil {
		t.Fatalf("CheckLatest returned error: %v", err)
	}
	if !got.Availability.Available {
		t.Fatalf("Available = false, want true: %+v", got.Availability)
	}
	if got.Availability.Current != "v0.11.0" {
		t.Fatalf("Current = %q, want v0.11.0", got.Availability.Current)
	}
	if got.Availability.Latest != "v0.12.0" {
		t.Fatalf("Latest = %q, want v0.12.0", got.Availability.Latest)
	}
	if got.Asset.Name != "onetiny-cli-darwin-arm64.zip" {
		t.Fatalf("asset name = %q, want darwin arm64 cli asset", got.Asset.Name)
	}
	if got.Channel != ChannelCLI {
		t.Fatalf("channel = %q, want %q", got.Channel, ChannelCLI)
	}
	if got.Platform != platform {
		t.Fatalf("platform = %+v, want %+v", got.Platform, platform)
	}
}

func TestServiceCheckTagRequestsTaggedRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/tcp404/OneTiny/releases/tags/v0.12.0" {
			t.Fatalf("path = %s, want tagged release path", r.URL.Path)
		}

		writeReleaseJSON(t, w, "v0.12.0", []Asset{
			{Name: "onetiny-cli-darwin-arm64.zip", DownloadURL: "https://example.test/cli.zip"},
		})
	}))
	defer server.Close()

	service := Service{Client: Client{BaseURL: server.URL, HTTPClient: server.Client()}}

	got, err := service.CheckTag(context.Background(), CheckOptions{
		Channel:        ChannelCLI,
		CurrentVersion: "v0.11.0",
		Platform:       Platform{OS: "darwin", Arch: "arm64"},
		Tag:            "v0.12.0",
	})
	if err != nil {
		t.Fatalf("CheckTag returned error: %v", err)
	}
	if got.Release.TagName != "v0.12.0" {
		t.Fatalf("release tag = %q, want v0.12.0", got.Release.TagName)
	}
	if !got.Availability.Available {
		t.Fatalf("Available = false, want true: %+v", got.Availability)
	}
}

func TestServiceListTagsForwardsTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/tcp404/OneTiny/tags" {
			t.Fatalf("path = %s, want tags path", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]map[string]string{
			{"name": "v0.12.0"},
			{"name": "v0.11.0"},
		}); err != nil {
			t.Fatalf("encode tags: %v", err)
		}
	}))
	defer server.Close()

	service := Service{Client: Client{BaseURL: server.URL, HTTPClient: server.Client()}}

	got, err := service.ListTags(context.Background())
	if err != nil {
		t.Fatalf("ListTags returned error: %v", err)
	}
	want := []string{"v0.12.0", "v0.11.0"}
	if len(got) != len(want) {
		t.Fatalf("tags len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tags[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestServiceCheckLatestReturnsAssetSelectionErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeReleaseJSON(t, w, "v0.12.0", []Asset{
			{Name: "onetiny-cli-linux-x64.zip", DownloadURL: "https://example.test/linux.zip"},
		})
	}))
	defer server.Close()

	service := Service{Client: Client{BaseURL: server.URL, HTTPClient: server.Client()}}

	_, err := service.CheckLatest(context.Background(), CheckOptions{
		Channel:        ChannelCLI,
		CurrentVersion: "v0.11.0",
		Platform:       Platform{OS: "darwin", Arch: "arm64"},
	})
	if !errors.Is(err, ErrAssetNotFound) {
		t.Fatalf("CheckLatest error = %v, want %v", err, ErrAssetNotFound)
	}
}

func TestServiceDownloadAndStageDownloadsVerifiedArchive(t *testing.T) {
	zipPath := createTestZip(t, []zipEntry{
		{name: "onetiny-cli", body: "cli", mode: 0o755},
	})
	zipBytes, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatalf("read test zip: %v", err)
	}
	checksum := sha256Hex(zipBytes)
	assetName := "onetiny-cli-darwin-arm64.zip"

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/tcp404/OneTiny/releases/latest":
			writeReleaseJSON(t, w, "v0.12.0", []Asset{
				{Name: assetName, DownloadURL: server.URL + "/downloads/" + assetName},
				{Name: checksumsAssetName, DownloadURL: server.URL + "/downloads/" + checksumsAssetName},
			})
		case "/downloads/" + assetName:
			if _, err := w.Write(zipBytes); err != nil {
				t.Fatalf("write zip: %v", err)
			}
		case "/downloads/" + checksumsAssetName:
			fmt.Fprintf(w, "%s  %s\n", checksum, assetName)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := Service{Client: Client{BaseURL: server.URL, HTTPClient: server.Client()}}
	result, err := service.CheckLatest(context.Background(), CheckOptions{
		Channel:        ChannelCLI,
		CurrentVersion: "v0.11.0",
		Platform:       Platform{OS: "darwin", Arch: "arm64"},
	})
	if err != nil {
		t.Fatalf("CheckLatest returned error: %v", err)
	}

	downloadResult, stageResult, err := service.DownloadAndStage(context.Background(), result, t.TempDir())
	if err != nil {
		t.Fatalf("DownloadAndStage returned error: %v", err)
	}
	defer removeStagingDir(t, stageResult.StagingDir)

	if _, err := os.Stat(downloadResult.ZipPath); err != nil {
		t.Fatalf("downloaded zip stat failed: %v", err)
	}
	if filepath.Base(stageResult.CandidatePath) != "onetiny-cli" {
		t.Fatalf("candidate basename = %q, want onetiny-cli", filepath.Base(stageResult.CandidatePath))
	}
}

func writeReleaseJSON(t *testing.T, w http.ResponseWriter, tag string, assets []Asset) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(Release{
		TagName: tag,
		Name:    "OneTiny " + tag,
		Assets:  assets,
	}); err != nil {
		t.Fatalf("encode release: %v", err)
	}
}
