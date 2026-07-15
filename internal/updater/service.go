package updater

import "context"

type Service struct {
	Client Client
}

type CheckOptions struct {
	Channel        Channel
	CurrentVersion string
	Platform       Platform
	Tag            string
}

func (s Service) CheckLatest(ctx context.Context, opts CheckOptions) (CheckResult, error) {
	release, err := s.Client.Latest(ctx)
	if err != nil {
		return CheckResult{}, err
	}
	return checkRelease(release, opts)
}

func (s Service) CheckTag(ctx context.Context, opts CheckOptions) (CheckResult, error) {
	release, err := s.Client.ByTag(ctx, opts.Tag)
	if err != nil {
		return CheckResult{}, err
	}
	return checkRelease(release, opts)
}

func (s Service) ListTags(ctx context.Context) ([]string, error) {
	return s.Client.Tags(ctx)
}

func (s Service) DownloadAndStage(ctx context.Context, result CheckResult, dir string) (DownloadResult, StageResult, error) {
	downloadResult, err := DownloadVerified(ctx, DownloadOptions{
		Client:  s.Client.httpClient(),
		Release: result.Release,
		Asset:   result.Asset,
		Dir:     dir,
	})
	if err != nil {
		return DownloadResult{}, StageResult{}, err
	}

	stageResult, err := StageArchive(downloadResult.ZipPath, result.Channel, result.Platform)
	if err != nil {
		return downloadResult, StageResult{}, err
	}
	return downloadResult, stageResult, nil
}

func checkRelease(release Release, opts CheckOptions) (CheckResult, error) {
	platform := opts.Platform
	if platform == (Platform{}) {
		platform = CurrentPlatform()
	}

	asset, err := FindAsset(release, opts.Channel, platform)
	if err != nil {
		return CheckResult{}, err
	}

	return CheckResult{
		Release:      release,
		Asset:        asset,
		Availability: IsUpdateAvailable(opts.CurrentVersion, release.TagName),
		Channel:      opts.Channel,
		Platform:     platform,
	}, nil
}
