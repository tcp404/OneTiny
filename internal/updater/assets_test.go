package updater

import (
	"errors"
	"testing"
)

func TestAssetNameSupportsReleaseMatrix(t *testing.T) {
	tests := []struct {
		name     string
		channel  Channel
		platform Platform
		want     string
	}{
		{
			name:     "cli linux x64",
			channel:  ChannelCLI,
			platform: Platform{OS: "linux", Arch: "x64"},
			want:     "onetiny-cli-linux-x64.zip",
		},
		{
			name:     "cli linux amd64",
			channel:  ChannelCLI,
			platform: Platform{OS: "linux", Arch: "amd64"},
			want:     "onetiny-cli-linux-x64.zip",
		},
		{
			name:     "cli windows x64",
			channel:  ChannelCLI,
			platform: Platform{OS: "windows", Arch: "x64"},
			want:     "onetiny-cli-windows-x64.zip",
		},
		{
			name:     "cli windows amd64",
			channel:  ChannelCLI,
			platform: Platform{OS: "windows", Arch: "amd64"},
			want:     "onetiny-cli-windows-x64.zip",
		},
		{
			name:     "cli darwin x64",
			channel:  ChannelCLI,
			platform: Platform{OS: "darwin", Arch: "x64"},
			want:     "onetiny-cli-darwin-x64.zip",
		},
		{
			name:     "cli darwin amd64",
			channel:  ChannelCLI,
			platform: Platform{OS: "darwin", Arch: "amd64"},
			want:     "onetiny-cli-darwin-x64.zip",
		},
		{
			name:     "cli darwin arm64",
			channel:  ChannelCLI,
			platform: Platform{OS: "darwin", Arch: "arm64"},
			want:     "onetiny-cli-darwin-arm64.zip",
		},
		{
			name:     "gui darwin arm64",
			channel:  ChannelGUI,
			platform: Platform{OS: "darwin", Arch: "arm64"},
			want:     "onetiny-gui-darwin-arm64.zip",
		},
		{
			name:     "gui windows x64",
			channel:  ChannelGUI,
			platform: Platform{OS: "windows", Arch: "x64"},
			want:     "onetiny-gui-windows-x64.zip",
		},
		{
			name:     "gui windows amd64",
			channel:  ChannelGUI,
			platform: Platform{OS: "windows", Arch: "amd64"},
			want:     "onetiny-gui-windows-x64.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AssetName(tt.channel, tt.platform)
			if err != nil {
				t.Fatalf("AssetName returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("AssetName = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAssetNameRejectsUnreleasedMatrixEntries(t *testing.T) {
	tests := []struct {
		name     string
		channel  Channel
		platform Platform
	}{
		{
			name:     "cli linux arm64",
			channel:  ChannelCLI,
			platform: Platform{OS: "linux", Arch: "arm64"},
		},
		{
			name:     "cli windows arm64",
			channel:  ChannelCLI,
			platform: Platform{OS: "windows", Arch: "arm64"},
		},
		{
			name:     "gui darwin amd64",
			channel:  ChannelGUI,
			platform: Platform{OS: "darwin", Arch: "amd64"},
		},
		{
			name:     "gui windows arm64",
			channel:  ChannelGUI,
			platform: Platform{OS: "windows", Arch: "arm64"},
		},
		{
			name:     "gui linux amd64",
			channel:  ChannelGUI,
			platform: Platform{OS: "linux", Arch: "amd64"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := AssetName(tt.channel, tt.platform)
			if !errors.Is(err, ErrUnsupportedPlatform) {
				t.Fatalf("AssetName error = %v, want %v", err, ErrUnsupportedPlatform)
			}
		})
	}
}

func TestFindAssetReturnsMatchingReleaseAsset(t *testing.T) {
	release := Release{
		TagName: "v1.2.3",
		Assets: []Asset{
			{Name: "onetiny-cli-linux-x64.zip", DownloadURL: "https://example.com/linux.zip"},
			{Name: "onetiny-cli-darwin-arm64.zip", DownloadURL: "https://example.com/darwin.zip"},
		},
	}

	got, err := FindAsset(release, ChannelCLI, Platform{OS: "darwin", Arch: "aarch64"})
	if err != nil {
		t.Fatalf("FindAsset returned error: %v", err)
	}
	if got.Name != "onetiny-cli-darwin-arm64.zip" {
		t.Fatalf("asset name = %q, want onetiny-cli-darwin-arm64.zip", got.Name)
	}
	if got.DownloadURL != "https://example.com/darwin.zip" {
		t.Fatalf("asset URL = %q, want darwin URL", got.DownloadURL)
	}
}
