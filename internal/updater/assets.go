package updater

import (
	"fmt"
	"runtime"
	"strings"
)

func CurrentPlatform() Platform {
	return Platform{
		OS:   runtime.GOOS,
		Arch: normalizeArch(runtime.GOARCH),
	}
}

func normalizeArch(arch string) string {
	switch strings.ToLower(strings.TrimSpace(arch)) {
	case "amd64", "x86_64", "x64":
		return "x64"
	case "arm64", "aarch64":
		return "arm64"
	default:
		return strings.ToLower(strings.TrimSpace(arch))
	}
}

func AssetName(channel Channel, platform Platform) (string, error) {
	osName := strings.ToLower(strings.TrimSpace(platform.OS))
	arch := normalizeArch(platform.Arch)
	if !isReleasedAsset(channel, osName, arch) {
		return "", fmt.Errorf("%w: %s %s/%s", ErrUnsupportedPlatform, channel, osName, arch)
	}

	return fmt.Sprintf("onetiny-%s-%s-%s.zip", channel, osName, arch), nil
}

func FindAsset(release Release, channel Channel, platform Platform) (Asset, error) {
	name, err := AssetName(channel, platform)
	if err != nil {
		return Asset{}, err
	}
	return FindAssetByName(release, name)
}

func FindAssetByName(release Release, name string) (Asset, error) {
	for _, asset := range release.Assets {
		if asset.Name == name {
			return asset, nil
		}
	}
	return Asset{}, fmt.Errorf("%w: %s", ErrAssetNotFound, name)
}

func isReleasedAsset(channel Channel, goos, arch string) bool {
	switch channel {
	case ChannelCLI:
		switch goos {
		case "linux", "windows":
			return arch == "x64"
		case "darwin":
			return arch == "x64" || arch == "arm64"
		}
	case ChannelGUI:
		switch goos {
		case "windows":
			return arch == "x64"
		case "darwin":
			return arch == "arm64"
		}
	}
	return false
}
