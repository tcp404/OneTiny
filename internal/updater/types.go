package updater

import "errors"

type Channel string

const (
	ChannelCLI Channel = "cli"
	ChannelGUI Channel = "gui"
)

type Platform struct {
	OS   string
	Arch string
}

type Asset struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"browser_download_url"`
}

type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Assets  []Asset `json:"assets"`
}

type Availability struct {
	Current   string
	Latest    string
	Known     bool
	Available bool
	Reason    string
}

type CheckResult struct {
	Release      Release
	Asset        Asset
	Availability Availability
	Channel      Channel
	Platform     Platform
}

var (
	ErrUnknownVersion      = errors.New("unknown version")
	ErrUnsupportedPlatform = errors.New("unsupported platform")
	ErrAssetNotFound       = errors.New("asset not found")
	ErrChecksumNotFound    = errors.New("checksum not found")
	ErrChecksumMismatch    = errors.New("checksum mismatch")
	ErrInvalidArchive      = errors.New("invalid archive")
)
