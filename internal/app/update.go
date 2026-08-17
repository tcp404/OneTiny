package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/tcp404/OneTiny/internal/updater"
	"github.com/tcp404/OneTiny/internal/version"
)

const releaseTagURLBase = "https://github.com/tcp404/OneTiny/releases/tag/"

var (
	ErrNoAvailableUpdate           = errors.New("没有可下载的可用更新")
	ErrNoDownloadedStage           = errors.New("没有已下载并准备好的更新")
	ErrUpdateInstallAlreadyStarted = errors.New("更新安装已启动")
)

type updateBackend interface {
	CheckLatest(context.Context, updater.CheckOptions) (updater.CheckResult, error)
	DownloadAndStage(context.Context, updater.CheckResult, string) (updater.DownloadResult, updater.StageResult, error)
}

func (s *Service) GetUpdateStatus() (UpdateStatusDTO, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updateStatusLocked(), nil
}

func (s *Service) CheckUpdate() (UpdateStatusDTO, error) {
	backend := s.updateBackend()
	s.mu.Lock()
	if status := s.updateStatusLocked(); status.State == "installing" {
		err := ErrUpdateInstallAlreadyStarted
		s.mu.Unlock()
		return status, err
	}
	s.updateStatus = UpdateStatusDTO{
		CurrentVersion: version.Version,
		State:          "checking",
		Message:        "正在检查更新",
	}
	s.hasUpdateCheck = false
	s.updateCheck = updater.CheckResult{}
	s.updateDownload = updater.DownloadResult{}
	s.updateStage = updater.StageResult{}
	s.mu.Unlock()

	result, err := backend.CheckLatest(context.Background(), updater.CheckOptions{
		Channel:        updater.ChannelGUI,
		CurrentVersion: version.Version,
		Platform:       updater.CurrentPlatform(),
	})

	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		return s.setUpdateErrorLocked(err), err
	}
	s.updateCheck = result
	s.hasUpdateCheck = true
	s.updateStatus = updateStatusFromCheck(result)
	return s.updateStatusLocked(), nil
}

func (s *Service) DownloadUpdate() (UpdateStatusDTO, error) {
	backend := s.updateBackend()
	s.mu.Lock()
	if status := s.updateStatusLocked(); status.State == "installing" {
		err := ErrUpdateInstallAlreadyStarted
		s.mu.Unlock()
		return status, err
	}
	if !s.hasUpdateCheck || !s.updateCheck.Availability.Available {
		err := ErrNoAvailableUpdate
		status := s.setUpdateErrorLocked(err)
		s.mu.Unlock()
		return status, err
	}
	check := s.updateCheck
	s.updateStatus = updateStatusFromCheck(check)
	s.updateStatus.State = "downloading"
	s.updateStatus.Message = "正在下载更新"
	s.updateDownload = updater.DownloadResult{}
	s.updateStage = updater.StageResult{}
	s.mu.Unlock()

	downloadResult, stageResult, err := backend.DownloadAndStage(context.Background(), check, "")

	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		return s.setUpdateErrorLocked(err), err
	}
	s.updateDownload = downloadResult
	s.updateStage = stageResult
	s.updateStatus = updateStatusFromCheck(check)
	s.updateStatus.State = "downloaded"
	s.updateStatus.Message = "更新包已下载"
	s.updateStatus.DownloadedPath = downloadResult.ZipPath
	return s.updateStatusLocked(), nil
}

func (s *Service) StartUpdateInstall() (UpdateInstallDTO, error) {
	s.mu.Lock()
	stage := s.updateStage
	currentExecutable := s.currentExecutable
	startUpdateHelper := s.startUpdateHelper
	if currentExecutable == nil {
		currentExecutable = os.Executable
	}
	if startUpdateHelper == nil {
		startUpdateHelper = updater.StartHelper
	}
	if status := s.updateStatusLocked(); status.State == "installing" {
		err := ErrUpdateInstallAlreadyStarted
		s.mu.Unlock()
		return UpdateInstallDTO{LogPath: status.LogPath, Message: err.Error()}, err
	}
	if stage.CandidatePath == "" {
		err := ErrNoDownloadedStage
		s.setUpdateErrorLocked(err)
		s.mu.Unlock()
		return UpdateInstallDTO{Message: err.Error()}, err
	}
	s.updateStage = updater.StageResult{}
	s.updateStatus = s.updateStatusLocked()
	s.updateStatus.State = "installing"
	s.updateStatus.Message = "正在启动更新安装"
	s.updateStatus.LogPath = ""
	s.mu.Unlock()

	currentPath, err := currentExecutable()
	if err != nil {
		return s.restoreDownloadedUpdateAfterInstallError(stage, fmt.Errorf("resolve current executable: %w", err))
	}
	if err := validateGUIInstallTargetForOS(runtime.GOOS, currentPath, stage.CandidatePath); err != nil {
		return s.restoreDownloadedUpdateAfterInstallError(stage, err)
	}
	plan, err := updater.BuildInstallPlan(updater.InstallOptions{
		Channel:         updater.ChannelGUI,
		CurrentPath:     currentPath,
		ReplacementPath: stage.CandidatePath,
		PID:             os.Getpid(),
		RestartCommand:  restartCommandForGUI(currentPath),
	})
	if err != nil {
		return s.restoreDownloadedUpdateAfterInstallError(stage, err)
	}
	helperPath, err := prepareGUIUpdateHelper(currentPath)
	if err != nil {
		return s.restoreDownloadedUpdateAfterInstallError(stage, err)
	}
	if err := startUpdateHelper(helperPath, plan); err != nil {
		return s.restoreDownloadedUpdateAfterInstallError(stage, err)
	}

	s.mu.Lock()
	s.updateStatus = s.updateStatusLocked()
	s.updateStatus.State = "installing"
	s.updateStatus.Message = "更新安装已启动"
	s.updateStatus.LogPath = plan.LogPath
	s.mu.Unlock()
	return UpdateInstallDTO{
		Started: true,
		LogPath: plan.LogPath,
		Message: "更新安装已启动",
	}, nil
}

func (s *Service) updateBackend() updateBackend {
	if s.updater != nil {
		return s.updater
	}
	return updater.Service{}
}

func (s *Service) updateStatusLocked() UpdateStatusDTO {
	status := s.updateStatus
	if status.State == "" {
		status.State = "idle"
	}
	if status.CurrentVersion == "" {
		status.CurrentVersion = version.Version
	}
	return status
}

func (s *Service) setUpdateErrorLocked(err error) UpdateStatusDTO {
	s.updateStatus = UpdateStatusDTO{
		CurrentVersion: version.Version,
		State:          "error",
		Message:        err.Error(),
	}
	return s.updateStatusLocked()
}

func (s *Service) restoreDownloadedUpdateAfterInstallError(stage updater.StageResult, err error) (UpdateInstallDTO, error) {
	s.mu.Lock()
	s.updateStage = stage
	status := s.updateStatusLocked()
	status.State = "downloaded"
	status.Message = err.Error()
	status.DownloadedPath = s.updateDownload.ZipPath
	status.LogPath = ""
	s.updateStatus = status
	s.mu.Unlock()
	return UpdateInstallDTO{Message: err.Error()}, err
}

func updateStatusFromCheck(result updater.CheckResult) UpdateStatusDTO {
	availability := result.Availability
	current := strings.TrimSpace(availability.Current)
	if current == "" {
		current = version.Version
	}
	latest := strings.TrimSpace(availability.Latest)
	if latest == "" {
		latest = strings.TrimSpace(result.Release.TagName)
	}

	status := UpdateStatusDTO{
		CurrentVersion: current,
		LatestVersion:  latest,
		Available:      availability.Available,
		ReleaseURL:     releaseURLFromCheck(result),
	}
	switch {
	case availability.Available:
		status.State = "available"
		if latest != "" {
			status.Message = fmt.Sprintf("发现新版本: %s", latest)
		} else {
			status.Message = "发现新版本"
		}
	case !availability.Known:
		status.State = "unknown"
		reason := strings.TrimSpace(availability.Reason)
		if reason != "" {
			status.Message = fmt.Sprintf("无法判断更新状态: %s", reason)
		} else {
			status.Message = "无法判断更新状态"
		}
	default:
		status.State = "current"
		if latest != "" {
			status.Message = fmt.Sprintf("当前已是最新版本: %s", latest)
		} else {
			status.Message = "当前已是最新版本"
		}
	}
	return status
}

func releaseURLFromCheck(result updater.CheckResult) string {
	tag := strings.TrimSpace(result.Release.TagName)
	if tag == "" {
		return ""
	}
	return releaseTagURLBase + url.PathEscape(tag)
}

func prepareGUIUpdateHelper(currentPath string) (string, error) {
	helperDir, err := os.MkdirTemp("", "onetiny-gui-updater-*")
	if err != nil {
		return "", fmt.Errorf("create update helper dir: %w", err)
	}
	if err := os.Chmod(helperDir, 0o700); err != nil {
		_ = os.RemoveAll(helperDir)
		return "", fmt.Errorf("secure update helper dir: %w", err)
	}

	helperName := "onetiny-gui-updater"
	if runtime.GOOS == "windows" {
		helperName += ".exe"
	}
	helperPath := filepath.Join(helperDir, helperName)
	if err := copyUpdateHelperExecutable(currentPath, helperPath); err != nil {
		_ = os.RemoveAll(helperDir)
		return "", err
	}
	return helperPath, nil
}

func copyUpdateHelperExecutable(sourcePath, targetPath string) (err error) {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open current executable: %w", err)
	}
	defer func() {
		err = errors.Join(err, wrapUpdateCloseError("close current executable", source.Close()))
	}()

	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		return fmt.Errorf("create update helper: %w", err)
	}
	defer func() {
		err = errors.Join(err, wrapUpdateCloseError("close update helper", target.Close()))
	}()

	if _, err := io.Copy(target, source); err != nil {
		return fmt.Errorf("copy update helper: %w", err)
	}
	if err := target.Chmod(0o755); err != nil {
		return fmt.Errorf("chmod update helper: %w", err)
	}
	if err := target.Sync(); err != nil {
		return fmt.Errorf("flush update helper: %w", err)
	}
	return nil
}

func wrapUpdateCloseError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func restartCommandForGUI(currentPath string) []string {
	return restartCommandForGUIForOS(runtime.GOOS, currentPath)
}

func restartCommandForGUIForOS(goos, currentPath string) []string {
	if goos == "darwin" {
		if bundlePath, ok := appBundlePathForExecutable(currentPath); ok {
			return []string{"open", bundlePath}
		}
	}
	return []string{currentPath}
}

func validateGUIInstallTargetForOS(goos, currentPath, replacementPath string) error {
	if goos != "darwin" || !isMacAppBundlePath(replacementPath) {
		return nil
	}
	if _, ok := appBundlePathForExecutable(currentPath); ok {
		return nil
	}
	return fmt.Errorf("macOS GUI 更新需要从 OneTiny.app 启动；当前运行的是未打包的二进制: %s", currentPath)
}

func isMacAppBundlePath(path string) bool {
	return strings.HasSuffix(filepath.ToSlash(filepath.Clean(path)), ".app")
}

func appBundlePathForExecutable(path string) (string, bool) {
	slashPath := filepath.ToSlash(filepath.Clean(path))
	const marker = ".app/Contents/MacOS/"
	index := strings.Index(slashPath, marker)
	if index < 0 {
		return "", false
	}
	return filepath.FromSlash(slashPath[:index+len(".app")]), true
}
