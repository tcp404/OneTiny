package updater

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestClientLatestRequestsGitHubLatestRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/repos/tcp404/OneTiny/releases/latest" {
			t.Fatalf("path = %s, want latest release path", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "application/vnd.github.v3+json" {
			t.Fatalf("Accept = %q, want GitHub v3 media type", got)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v1.2.3",
			"name":     "OneTiny v1.2.3",
			"html_url": "https://github.com/tcp404/OneTiny/releases/tag/v1.2.3",
			"assets": []map[string]any{
				{
					"id":                   123,
					"name":                 "onetiny-cli-linux-x64.zip",
					"url":                  "https://api.github.com/assets/123",
					"content_type":         "application/zip",
					"size":                 456,
					"browser_download_url": "https://example.test/cli.zip",
				},
			},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := Client{BaseURL: server.URL, HTTPClient: server.Client()}

	got, err := client.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest returned error: %v", err)
	}
	if got.TagName != "v1.2.3" {
		t.Fatalf("TagName = %q, want v1.2.3", got.TagName)
	}
	if got.Name != "OneTiny v1.2.3" {
		t.Fatalf("Name = %q, want release name", got.Name)
	}
	if len(got.Assets) != 1 {
		t.Fatalf("assets len = %d, want 1", len(got.Assets))
	}
	asset := got.Assets[0]
	if asset.ID != 123 {
		t.Fatalf("asset ID = %d, want 123", asset.ID)
	}
	if asset.Name != "onetiny-cli-linux-x64.zip" {
		t.Fatalf("asset name = %q, want cli zip", asset.Name)
	}
	if asset.DownloadURL != "https://example.test/cli.zip" {
		t.Fatalf("asset download URL = %q, want browser download URL", asset.DownloadURL)
	}
}

func TestClientByTagRequestsGitHubTaggedRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/tcp404/OneTiny/releases/tags/v1.2.3" {
			t.Fatalf("path = %s, want tagged release path", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "application/vnd.github.v3+json" {
			t.Fatalf("Accept = %q, want GitHub v3 media type", got)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v1.2.3",
			"html_url": "https://github.com/tcp404/OneTiny/releases/tag/v1.2.3",
			"assets":   []map[string]any{},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := Client{BaseURL: server.URL, HTTPClient: server.Client()}

	got, err := client.ByTag(context.Background(), "v1.2.3")
	if err != nil {
		t.Fatalf("ByTag returned error: %v", err)
	}
	if got.TagName != "v1.2.3" {
		t.Fatalf("TagName = %q, want v1.2.3", got.TagName)
	}
}

func TestClientTagsReturnsTagNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/repos/tcp404/OneTiny/tags" {
			t.Fatalf("path = %s, want tags path", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "application/vnd.github.v3+json" {
			t.Fatalf("Accept = %q, want GitHub v3 media type", got)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]map[string]string{
			{"name": "v1.2.3"},
			{"name": "v1.2.2"},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := Client{BaseURL: server.URL, HTTPClient: server.Client()}

	got, err := client.Tags(context.Background())
	if err != nil {
		t.Fatalf("Tags returned error: %v", err)
	}
	want := []string{"v1.2.3", "v1.2.2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Tags = %#v, want %#v", got, want)
	}
}

func TestClientLatestReturnsErrorForNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusForbidden)
	}))
	defer server.Close()

	client := Client{BaseURL: server.URL, HTTPClient: server.Client()}

	_, err := client.Latest(context.Background())
	if err == nil {
		t.Fatal("Latest returned nil error, want non-2xx error")
	}
}

func TestClientUsesTimeoutDefaultHTTPClient(t *testing.T) {
	got := (Client{}).httpClient()
	if got == http.DefaultClient {
		t.Fatal("default client = http.DefaultClient, want package client with timeout")
	}
	if got.Timeout != 30*time.Second {
		t.Fatalf("default timeout = %s, want 30s", got.Timeout)
	}

	custom := &http.Client{}
	if got := (Client{HTTPClient: custom}).httpClient(); got != custom {
		t.Fatal("custom HTTPClient was not used")
	}
}
