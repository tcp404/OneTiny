package gui

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tcp404/OneTiny/internal/app"
	"github.com/tcp404/OneTiny/internal/config"
	"github.com/tcp404/OneTiny/internal/runtime"
	"github.com/tcp404/OneTiny/internal/server"
	"github.com/tcp404/OneTiny/internal/updater"
)

func TestServiceDialogMethodsAreNoopWithoutAdapter(t *testing.T) {
	service := NewService(&app.Service{}, nil)
	if got, err := service.ChooseDirectory("/tmp"); err != nil || got != "" {
		t.Fatalf("ChooseDirectory() = %q, %v; want empty nil", got, err)
	}
	if got, err := service.ExportLogs(app.LogFilterDTO{}); err != nil || got != "" {
		t.Fatalf("ExportLogs() = %q, %v; want empty nil", got, err)
	}
	if err := service.OpenConfigDir(); err != nil {
		t.Fatalf("OpenConfigDir() error = %v", err)
	}
	if err := service.OpenShareAddress(); err != nil {
		t.Fatalf("OpenShareAddress() error = %v", err)
	}
}

func TestOpenShareAddressSkipsEmptyAddress(t *testing.T) {
	service := NewService(newTestAppService(t), &recordingDialogs{})

	if err := service.OpenShareAddress(); err != nil {
		t.Fatalf("OpenShareAddress() error = %v", err)
	}

	dialogs := service.dialogs.(*recordingDialogs)
	if len(dialogs.openedURLs) != 0 {
		t.Fatalf("opened URLs = %v, want none", dialogs.openedURLs)
	}
}

func TestOpenShareAddressOpensRunningAddress(t *testing.T) {
	appService := newTestAppService(t)
	dialogs := &recordingDialogs{}
	service := NewService(appService, dialogs)

	status, err := appService.StartSharing()
	if err != nil {
		t.Fatalf("StartSharing() error = %v", err)
	}
	t.Cleanup(func() {
		_, _ = appService.StopSharing()
	})

	if err := service.OpenShareAddress(); err != nil {
		t.Fatalf("OpenShareAddress() error = %v", err)
	}
	if len(dialogs.openedURLs) != 1 {
		t.Fatalf("opened URLs = %v, want one URL", dialogs.openedURLs)
	}
	if dialogs.openedURLs[0] != status.Address {
		t.Fatalf("opened URL = %q, want %q", dialogs.openedURLs[0], status.Address)
	}
}

func TestUpdateMethodsForwardToAppService(t *testing.T) {
	fake := &guiUpdateBackend{
		checkResult:    guiAvailableUpdateCheck(),
		downloadResult: updater.DownloadResult{ZipPath: filepath.Join(t.TempDir(), "OneTiny.zip")},
		stageResult:    updater.StageResult{CandidatePath: filepath.Join(t.TempDir(), "OneTiny.app")},
	}
	service := NewService(newTestAppServiceWithDependencies(t, app.Dependencies{
		Updater: fake,
	}), &recordingDialogs{})

	initial, err := service.GetUpdateStatus()
	if err != nil {
		t.Fatalf("GetUpdateStatus() error = %v", err)
	}
	if initial.State != "idle" {
		t.Fatalf("initial State = %q, want idle", initial.State)
	}

	checked, err := service.CheckUpdate()
	if err != nil {
		t.Fatalf("CheckUpdate() error = %v", err)
	}
	if fake.checkCalls != 1 {
		t.Fatalf("CheckLatest calls = %d, want 1", fake.checkCalls)
	}
	if checked.State != "available" || !checked.Available {
		t.Fatalf("checked status = %+v, want available", checked)
	}

	downloaded, err := service.DownloadUpdate()
	if err != nil {
		t.Fatalf("DownloadUpdate() error = %v", err)
	}
	if fake.downloadCalls != 1 {
		t.Fatalf("DownloadAndStage calls = %d, want 1", fake.downloadCalls)
	}
	if downloaded.State != "downloaded" || downloaded.DownloadedPath != fake.downloadResult.ZipPath {
		t.Fatalf("downloaded status = %+v", downloaded)
	}
}

func TestInstallUpdateWithoutDownloadedUpdateReturnsError(t *testing.T) {
	service := NewService(newTestAppService(t), &recordingDialogs{})
	quitCalled := make(chan struct{}, 1)
	service.setQuitForUpdate(func() {
		quitCalled <- struct{}{}
	})

	_, err := service.InstallUpdate()
	if !errors.Is(err, app.ErrNoDownloadedStage) {
		t.Fatalf("InstallUpdate() error = %v, want %v", err, app.ErrNoDownloadedStage)
	}
	select {
	case <-quitCalled:
		t.Fatal("quit callback was called after failed install")
	case <-time.After(updateQuitDelay + 50*time.Millisecond):
	}
}

func TestInstallUpdateStartsHelperAndTriggersQuitCallback(t *testing.T) {
	bundlePath := filepath.Join(t.TempDir(), "OneTiny.app")
	executablePath := filepath.Join(bundlePath, "Contents", "MacOS", "OneTiny")
	if err := os.MkdirAll(filepath.Dir(executablePath), 0o755); err != nil {
		t.Fatalf("MkdirAll executable dir: %v", err)
	}
	if err := os.WriteFile(executablePath, []byte("fake executable"), 0o755); err != nil {
		t.Fatalf("WriteFile executable: %v", err)
	}

	fake := &guiUpdateBackend{
		checkResult:    guiAvailableUpdateCheck(),
		downloadResult: updater.DownloadResult{ZipPath: filepath.Join(t.TempDir(), "OneTiny.zip")},
		stageResult:    updater.StageResult{CandidatePath: filepath.Join(t.TempDir(), "StagedOneTiny.app")},
	}
	var helperStarted atomic.Bool
	service := NewService(newTestAppServiceWithDependencies(t, app.Dependencies{
		Updater: fake,
		CurrentExecutable: func() (string, error) {
			return executablePath, nil
		},
		StartUpdateHelper: func(string, updater.InstallPlan) error {
			helperStarted.Store(true)
			return nil
		},
	}), &recordingDialogs{})
	quitCalled := make(chan struct{}, 1)
	service.setQuitForUpdate(func() {
		quitCalled <- struct{}{}
	})

	if _, err := service.CheckUpdate(); err != nil {
		t.Fatalf("CheckUpdate() error = %v", err)
	}
	if _, err := service.DownloadUpdate(); err != nil {
		t.Fatalf("DownloadUpdate() error = %v", err)
	}

	result, err := service.InstallUpdate()
	if err != nil {
		t.Fatalf("InstallUpdate() error = %v", err)
	}
	if !result.Started {
		t.Fatal("Started = false, want true")
	}
	if !helperStarted.Load() {
		t.Fatal("update helper was not started")
	}
	select {
	case <-quitCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("quit callback was not called")
	}
}

type guiUpdateBackend struct {
	checkResult    updater.CheckResult
	checkCalls     int
	downloadResult updater.DownloadResult
	stageResult    updater.StageResult
	downloadCalls  int
}

func (f *guiUpdateBackend) CheckLatest(context.Context, updater.CheckOptions) (updater.CheckResult, error) {
	f.checkCalls++
	return f.checkResult, nil
}

func (f *guiUpdateBackend) DownloadAndStage(context.Context, updater.CheckResult, string) (updater.DownloadResult, updater.StageResult, error) {
	f.downloadCalls++
	return f.downloadResult, f.stageResult, nil
}

func guiAvailableUpdateCheck() updater.CheckResult {
	return updater.CheckResult{
		Release: updater.Release{TagName: "v9.9.9"},
		Availability: updater.Availability{
			Current:   "v1.0.0",
			Latest:    "v9.9.9",
			Known:     true,
			Available: true,
		},
		Channel:  updater.ChannelGUI,
		Platform: updater.Platform{OS: "darwin", Arch: "arm64"},
	}
}

type recordingDialogs struct {
	openedURLs []string
}

func (d *recordingDialogs) ChooseDirectory(string) (string, error) {
	return "", nil
}

func (d *recordingDialogs) ChooseExportPath() (string, error) {
	return "", nil
}

func (d *recordingDialogs) OpenConfigDir() error {
	return nil
}

func (d *recordingDialogs) OpenURL(url string) error {
	d.openedURLs = append(d.openedURLs, url)
	return nil
}

func (d *recordingDialogs) ConfirmQuitWhileRunning(func()) error {
	return nil
}

func newTestAppService(t *testing.T) *app.Service {
	t.Helper()
	return newTestAppServiceWithDependencies(t, app.Dependencies{})
}

func newTestAppServiceWithDependencies(t *testing.T, overrides app.Dependencies) *app.Service {
	t.Helper()
	root := t.TempDir()
	port := freeTCPPort(t)
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(`
server:
  road: `+root+`
  port: `+strconv.Itoa(port)+`
  allow_upload: false
  max_level: 1
account:
  secure: false
scratch:
  max_items: 500
  max_item_size: 10MB
`), 0o600); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}
	store := config.NewStore(path)
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	rt := runtime.New(runtime.SnapshotFromConfig(runtime.PersistentConfig{
		RootPath:            cfg.RootPath,
		Port:                cfg.Port,
		MaxLevel:            cfg.MaxLevel,
		IsAllowUpload:       cfg.IsAllowUpload,
		IsSecure:            cfg.IsSecure,
		Username:            cfg.Username,
		PasswordHash:        cfg.PasswordHash,
		ScratchMaxItems:     cfg.ScratchMaxItems,
		ScratchMaxItemSize:  cfg.ScratchMaxItemSize,
		ScratchMaxItemBytes: 10 * 1024 * 1024,
	}, runtime.Process{IP: "127.0.0.1", Pwd: root, SessionVal: "session"}))
	deps := app.Dependencies{
		ConfigStore: store,
		Runtime:     rt,
		Manager:     server.NewManager(rt),
	}
	if overrides.Updater != nil {
		deps.Updater = overrides.Updater
	}
	if overrides.CurrentExecutable != nil {
		deps.CurrentExecutable = overrides.CurrentExecutable
	}
	if overrides.StartUpdateHelper != nil {
		deps.StartUpdateHelper = overrides.StartUpdateHelper
	}
	return app.NewService(deps)
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen tcp: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}
