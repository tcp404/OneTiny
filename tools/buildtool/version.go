package main

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var releaseVersionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`)

const releasePleaseManifestPath = ".release-please-manifest.json"

func DefaultVersion() (string, error) {
	data, err := os.ReadFile(releasePleaseManifestPath)
	if err != nil {
		return "", errors.Wrap(err, "read release-please manifest")
	}

	var manifest map[string]string
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", errors.Wrap(err, "parse release-please manifest")
	}

	version := strings.TrimSpace(manifest["."])
	if version == "" {
		return "", errors.Errorf("%s does not contain root package version", releasePleaseManifestPath)
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if err := ValidateVersion(version); err != nil {
		return "", err
	}
	return version, nil
}

func ValidateVersion(version string) error {
	if !releaseVersionPattern.MatchString(version) {
		return errors.Errorf("version %q must be a release tag like v0.6.0", version)
	}
	return nil
}
