package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tcp404/OneTiny/internal/config"
	"github.com/tcp404/OneTiny/internal/runtime"
	"github.com/tcp404/OneTiny/internal/server"
	"github.com/tcp404/OneTiny/internal/updater"
	"github.com/tcp404/OneTiny/internal/version"
)

type fakeUpdateBackend struct {
	checkResult updater.CheckResult
	checkErr    error
	checkCalls  int
	checkOpts   updater.CheckOptions

	downloadResult updater.DownloadResult
	stageResult    updater.StageResult
	downloadErr    error
	downloadCalls  int
	downloadCheck  updater.CheckResult
	downloadDir    string
}

func (f *fakeUpdateBackend) CheckLatest(ctx context.Context, opts updater.CheckOptions) (updater.CheckResult, error) {
	if ctx == nil {
		return updater.CheckResult{}, errors.New("context is nil")
	}
	f.checkCalls++
	f.checkOpts = opts
	return f.checkResult, f.checkErr
}

func (f *fakeUpdateBackend) DownloadAndStage(ctx context.Context, result updater.CheckResult, dir string) (updater.DownloadResult, updater.StageResult, error) {
	if ctx == nil {
		return updater.DownloadResult{}, updater.StageResult{}, errors.New("context is nil")
	}
	f.downloadCalls++
	f.downloadCheck = result
	f.downloadDir = dir
	return f.downloadResult, f.stageResult, f.downloadErr
}

func newUpdateTestService(t *testing.T, backend updateBackend) *Service {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(`
server:
  road: `+root+`
  port: 0
  allow_upload: false
  max_level: 1
account:
  secure: false
`), 0o600); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}
	store := config.NewStore(path)
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	process := runtime.Process{IP: "127.0.0.1", Pwd: root, SessionVal: "session"}
	rt := runtime.New(runtime.SnapshotFromConfig(runtimeConfigFromConfig(cfg), process))
	return NewService(Dependencies{
		ConfigStore: store,
		Runtime:     rt,
		Manager:     server.NewManager(rt),
		Updater:     backend,
	})
}

func availableUpdateCheck() updater.CheckResult {
	return updater.CheckResult{
		Release: updater.Release{TagName: "v9.9.9", Name: "OneTiny v9.9.9"},
		Asset:   updater.Asset{Name: "OneTiny-darwin-arm64.zip"},
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

func TestGetUpdateStatusInitialIdle(t *testing.T) {
	svc := newUpdateTestService(t, nil)

	status, err := svc.GetUpdateStatus()
	if err != nil {
		t.Fatalf("GetUpdateStatus returned error: %v", err)
	}

	if status.State != "idle" {
		t.Fatalf("State = %q, want idle", status.State)
	}
	if status.CurrentVersion != version.Version {
		t.Fatalf("CurrentVersion = %q, want %q", status.CurrentVersion, version.Version)
	}
	if status.Available {
		t.Fatal("Available = true, want false")
	}
}

func TestUpdateStatusFromCheckStates(t *testing.T) {
	available := updateStatusFromCheck(availableUpdateCheck())
	if available.State != "available" || !available.Available {
		t.Fatalf("available status = %+v", available)
	}
	if available.LatestVersion != "v9.9.9" {
		t.Fatalf("available LatestVersion = %q, want v9.9.9", available.LatestVersion)
	}

	currentCheck := availableUpdateCheck()
	currentCheck.Availability.Available = false
	currentCheck.Availability.Latest = currentCheck.Availability.Current
	current := updateStatusFromCheck(currentCheck)
	if current.State != "current" || current.Available {
		t.Fatalf("current status = %+v", current)
	}

	unknownCheck := availableUpdateCheck()
	unknownCheck.Availability = updater.Availability{
		Current: "dev",
		Latest:  "v9.9.9",
		Reason:  updater.ErrUnknownVersion.Error(),
	}
	unknown := updateStatusFromCheck(unknownCheck)
	if unknown.State != "unknown" || unknown.Available {
		t.Fatalf("unknown status = %+v", unknown)
	}
	if !strings.Contains(unknown.Message, updater.ErrUnknownVersion.Error()) {
		t.Fatalf("unknown Message = %q, want reason", unknown.Message)
	}
}

func TestCheckUpdateUsesBackendAndStoresAvailableStatus(t *testing.T) {
	fake := &fakeUpdateBackend{checkResult: availableUpdateCheck()}
	svc := newUpdateTestService(t, fake)

	status, err := svc.CheckUpdate()
	if err != nil {
		t.Fatalf("CheckUpdate returned error: %v", err)
	}

	if fake.checkCalls != 1 {
		t.Fatalf("CheckLatest calls = %d, want 1", fake.checkCalls)
	}
	if fake.checkOpts.Channel != updater.ChannelGUI {
		t.Fatalf("CheckLatest channel = %q, want %q", fake.checkOpts.Channel, updater.ChannelGUI)
	}
	if fake.checkOpts.CurrentVersion != version.Version {
		t.Fatalf("CheckLatest current version = %q, want %q", fake.checkOpts.CurrentVersion, version.Version)
	}
	if fake.checkOpts.Platform == (updater.Platform{}) {
		t.Fatal("CheckLatest platform is empty")
	}
	if status.State != "available" || !status.Available {
		t.Fatalf("status = %+v", status)
	}
	if status.ReleaseURL != "https://github.com/tcp404/OneTiny/releases/tag/v9.9.9" {
		t.Fatalf("ReleaseURL = %q", status.ReleaseURL)
	}

	cached, err := svc.GetUpdateStatus()
	if err != nil {
		t.Fatalf("GetUpdateStatus returned error: %v", err)
	}
	if cached.ReleaseURL != status.ReleaseURL {
		t.Fatalf("cached ReleaseURL = %q, want %q", cached.ReleaseURL, status.ReleaseURL)
	}
}

func TestDownloadUpdateRequiresAvailableCheck(t *testing.T) {
	svc := newUpdateTestService(t, &fakeUpdateBackend{})

	if _, err := svc.DownloadUpdate(); err == nil {
		t.Fatal("DownloadUpdate error is nil, want missing available check error")
	}
}

func TestDownloadUpdateStoresDownloadedStatus(t *testing.T) {
	check := availableUpdateCheck()
	fake := &fakeUpdateBackend{
		checkResult:    check,
		downloadResult: updater.DownloadResult{ZipPath: filepath.Join(t.TempDir(), "OneTiny.zip")},
		stageResult:    updater.StageResult{CandidatePath: filepath.Join(t.TempDir(), "OneTiny.app")},
		downloadErr:    nil,
		downloadCalls:  0,
		downloadCheck:  updater.CheckResult{},
		downloadDir:    "unused",
	}
	svc := newUpdateTestService(t, fake)
	if _, err := svc.CheckUpdate(); err != nil {
		t.Fatalf("CheckUpdate returned error: %v", err)
	}

	status, err := svc.DownloadUpdate()
	if err != nil {
		t.Fatalf("DownloadUpdate returned error: %v", err)
	}

	if fake.downloadCalls != 1 {
		t.Fatalf("DownloadAndStage calls = %d, want 1", fake.downloadCalls)
	}
	if !reflect.DeepEqual(fake.downloadCheck, check) {
		t.Fatalf("DownloadAndStage check = %+v, want %+v", fake.downloadCheck, check)
	}
	if fake.downloadDir != "" {
		t.Fatalf("DownloadAndStage dir = %q, want empty", fake.downloadDir)
	}
	if status.State != "downloaded" {
		t.Fatalf("State = %q, want downloaded", status.State)
	}
	if status.DownloadedPath != fake.downloadResult.ZipPath {
		t.Fatalf("DownloadedPath = %q, want %q", status.DownloadedPath, fake.downloadResult.ZipPath)
	}
}

func TestStartUpdateInstallRequiresDownloadedStage(t *testing.T) {
	svc := newUpdateTestService(t, &fakeUpdateBackend{})

	if _, err := svc.StartUpdateInstall(); err == nil {
		t.Fatal("StartUpdateInstall error is nil, want missing downloaded stage error")
	}
}

func TestValidateGUIInstallTargetRejectsMacAppFromBareBinary(t *testing.T) {
	err := validateGUIInstallTargetForOS(
		"darwin",
		filepath.FromSlash("/tmp/OneTiny"),
		filepath.FromSlash("/tmp/onetiny-stage/OneTiny.app"),
	)
	if err == nil {
		t.Fatal("validateGUIInstallTargetForOS error is nil, want unsupported install target")
	}
	if !strings.Contains(err.Error(), "OneTiny.app") {
		t.Fatalf("error = %q, want OneTiny.app guidance", err)
	}
}

func TestStartUpdateInstallBuildsGUIPlanAndStartsHelper(t *testing.T) {
	check := availableUpdateCheck()
	zipPath := filepath.Join(t.TempDir(), "OneTiny.zip")
	bundlePath := filepath.Join(t.TempDir(), "OneTiny.app")
	executablePath := filepath.Join(bundlePath, "Contents", "MacOS", "OneTiny")
	if err := os.MkdirAll(filepath.Dir(executablePath), 0o755); err != nil {
		t.Fatalf("MkdirAll executable dir: %v", err)
	}
	if err := os.WriteFile(executablePath, []byte("fake executable"), 0o755); err != nil {
		t.Fatalf("WriteFile executable: %v", err)
	}
	candidatePath := filepath.Join(t.TempDir(), "StagedOneTiny.app")

	var helperPath string
	var plan updater.InstallPlan
	fake := &fakeUpdateBackend{
		checkResult:    check,
		downloadResult: updater.DownloadResult{ZipPath: zipPath},
		stageResult:    updater.StageResult{CandidatePath: candidatePath},
	}
	svc := newUpdateTestService(t, fake)
	svc.currentExecutable = func() (string, error) {
		return executablePath, nil
	}
	svc.startUpdateHelper = func(gotHelperPath string, gotPlan updater.InstallPlan) error {
		helperPath = gotHelperPath
		plan = gotPlan
		if _, err := os.Stat(gotHelperPath); err != nil {
			return err
		}
		return nil
	}

	if _, err := svc.CheckUpdate(); err != nil {
		t.Fatalf("CheckUpdate returned error: %v", err)
	}
	if _, err := svc.DownloadUpdate(); err != nil {
		t.Fatalf("DownloadUpdate returned error: %v", err)
	}

	result, err := svc.StartUpdateInstall()
	if err != nil {
		t.Fatalf("StartUpdateInstall returned error: %v", err)
	}

	if !result.Started {
		t.Fatal("Started = false, want true")
	}
	if result.LogPath == "" {
		t.Fatal("LogPath is empty")
	}
	if helperPath == "" {
		t.Fatal("helper was not started")
	}
	if helperPath == executablePath {
		t.Fatal("helper path should not be current executable")
	}
	if plan.TargetPath != bundlePath {
		t.Fatalf("TargetPath = %q, want bundle path %q", plan.TargetPath, bundlePath)
	}
	if plan.ReplacementPath != candidatePath {
		t.Fatalf("ReplacementPath = %q, want %q", plan.ReplacementPath, candidatePath)
	}
	wantRestart := restartCommandForGUI(executablePath)
	if !reflect.DeepEqual(plan.RestartCommand, wantRestart) {
		t.Fatalf("RestartCommand = %#v, want %#v", plan.RestartCommand, wantRestart)
	}
	if result.LogPath != plan.LogPath {
		t.Fatalf("result LogPath = %q, want %q", result.LogPath, plan.LogPath)
	}
}

func TestStartUpdateInstallRejectsSecondStart(t *testing.T) {
	check := availableUpdateCheck()
	zipPath := filepath.Join(t.TempDir(), "OneTiny.zip")
	bundlePath := filepath.Join(t.TempDir(), "OneTiny.app")
	executablePath := filepath.Join(bundlePath, "Contents", "MacOS", "OneTiny")
	if err := os.MkdirAll(filepath.Dir(executablePath), 0o755); err != nil {
		t.Fatalf("MkdirAll executable dir: %v", err)
	}
	if err := os.WriteFile(executablePath, []byte("fake executable"), 0o755); err != nil {
		t.Fatalf("WriteFile executable: %v", err)
	}

	calls := 0
	fake := &fakeUpdateBackend{
		checkResult:    check,
		downloadResult: updater.DownloadResult{ZipPath: zipPath},
		stageResult:    updater.StageResult{CandidatePath: filepath.Join(t.TempDir(), "StagedOneTiny.app")},
	}
	svc := newUpdateTestService(t, fake)
	svc.currentExecutable = func() (string, error) {
		return executablePath, nil
	}
	svc.startUpdateHelper = func(string, updater.InstallPlan) error {
		calls++
		return nil
	}

	if _, err := svc.CheckUpdate(); err != nil {
		t.Fatalf("CheckUpdate returned error: %v", err)
	}
	if _, err := svc.DownloadUpdate(); err != nil {
		t.Fatalf("DownloadUpdate returned error: %v", err)
	}
	if _, err := svc.StartUpdateInstall(); err != nil {
		t.Fatalf("first StartUpdateInstall returned error: %v", err)
	}
	if _, err := svc.StartUpdateInstall(); err == nil {
		t.Fatal("second StartUpdateInstall error is nil, want duplicate install error")
	}
	if calls != 1 {
		t.Fatalf("StartUpdateHelper calls = %d, want 1", calls)
	}
}

func TestStartUpdateInstallRestoresDownloadedStateWhenHelperFails(t *testing.T) {
	check := availableUpdateCheck()
	zipPath := filepath.Join(t.TempDir(), "OneTiny.zip")
	bundlePath := filepath.Join(t.TempDir(), "OneTiny.app")
	executablePath := filepath.Join(bundlePath, "Contents", "MacOS", "OneTiny")
	if err := os.MkdirAll(filepath.Dir(executablePath), 0o755); err != nil {
		t.Fatalf("MkdirAll executable dir: %v", err)
	}
	if err := os.WriteFile(executablePath, []byte("fake executable"), 0o755); err != nil {
		t.Fatalf("WriteFile executable: %v", err)
	}

	startErr := errors.New("helper start failed")
	calls := 0
	fake := &fakeUpdateBackend{
		checkResult:    check,
		downloadResult: updater.DownloadResult{ZipPath: zipPath},
		stageResult:    updater.StageResult{CandidatePath: filepath.Join(t.TempDir(), "StagedOneTiny.app")},
	}
	svc := newUpdateTestService(t, fake)
	svc.currentExecutable = func() (string, error) {
		return executablePath, nil
	}
	svc.startUpdateHelper = func(string, updater.InstallPlan) error {
		calls++
		if calls == 1 {
			return startErr
		}
		return nil
	}

	if _, err := svc.CheckUpdate(); err != nil {
		t.Fatalf("CheckUpdate returned error: %v", err)
	}
	if _, err := svc.DownloadUpdate(); err != nil {
		t.Fatalf("DownloadUpdate returned error: %v", err)
	}
	if _, err := svc.StartUpdateInstall(); !errors.Is(err, startErr) {
		t.Fatalf("first StartUpdateInstall error = %v, want %v", err, startErr)
	}
	status, err := svc.GetUpdateStatus()
	if err != nil {
		t.Fatalf("GetUpdateStatus returned error: %v", err)
	}
	if status.State != "downloaded" {
		t.Fatalf("State after helper failure = %q, want downloaded", status.State)
	}
	if !strings.Contains(status.Message, startErr.Error()) {
		t.Fatalf("Message after helper failure = %q, want helper error", status.Message)
	}

	retry, err := svc.StartUpdateInstall()
	if err != nil {
		t.Fatalf("retry StartUpdateInstall returned error: %v", err)
	}
	if !retry.Started {
		t.Fatal("retry Started = false, want true")
	}
	if calls != 2 {
		t.Fatalf("StartUpdateHelper calls = %d, want 2", calls)
	}
}

func TestDownloadUpdateRejectsInstallingState(t *testing.T) {
	check := availableUpdateCheck()
	zipPath := filepath.Join(t.TempDir(), "OneTiny.zip")
	bundlePath := filepath.Join(t.TempDir(), "OneTiny.app")
	executablePath := filepath.Join(bundlePath, "Contents", "MacOS", "OneTiny")
	if err := os.MkdirAll(filepath.Dir(executablePath), 0o755); err != nil {
		t.Fatalf("MkdirAll executable dir: %v", err)
	}
	if err := os.WriteFile(executablePath, []byte("fake executable"), 0o755); err != nil {
		t.Fatalf("WriteFile executable: %v", err)
	}

	calls := 0
	fake := &fakeUpdateBackend{
		checkResult:    check,
		downloadResult: updater.DownloadResult{ZipPath: zipPath},
		stageResult:    updater.StageResult{CandidatePath: filepath.Join(t.TempDir(), "StagedOneTiny.app")},
	}
	svc := newUpdateTestService(t, fake)
	svc.currentExecutable = func() (string, error) {
		return executablePath, nil
	}
	svc.startUpdateHelper = func(string, updater.InstallPlan) error {
		calls++
		return nil
	}

	if _, err := svc.CheckUpdate(); err != nil {
		t.Fatalf("CheckUpdate returned error: %v", err)
	}
	if _, err := svc.DownloadUpdate(); err != nil {
		t.Fatalf("DownloadUpdate returned error: %v", err)
	}
	started, err := svc.StartUpdateInstall()
	if err != nil {
		t.Fatalf("StartUpdateInstall returned error: %v", err)
	}

	if _, err := svc.DownloadUpdate(); err == nil {
		t.Fatal("DownloadUpdate during installing error is nil, want installing error")
	}
	status, err := svc.GetUpdateStatus()
	if err != nil {
		t.Fatalf("GetUpdateStatus returned error: %v", err)
	}
	if status.State != "installing" {
		t.Fatalf("State after rejected DownloadUpdate = %q, want installing", status.State)
	}
	if status.LogPath != started.LogPath {
		t.Fatalf("LogPath after rejected DownloadUpdate = %q, want %q", status.LogPath, started.LogPath)
	}
	if svc.updateStage != (updater.StageResult{}) {
		t.Fatalf("updateStage after rejected DownloadUpdate = %+v, want consumed stage", svc.updateStage)
	}
	if fake.downloadCalls != 1 {
		t.Fatalf("DownloadAndStage calls = %d, want 1", fake.downloadCalls)
	}
	if _, err := svc.StartUpdateInstall(); err == nil {
		t.Fatal("second StartUpdateInstall error is nil, want duplicate install error")
	}
	if calls != 1 {
		t.Fatalf("StartUpdateHelper calls = %d, want 1", calls)
	}
}

func TestCheckUpdateRejectsInstallingState(t *testing.T) {
	check := availableUpdateCheck()
	zipPath := filepath.Join(t.TempDir(), "OneTiny.zip")
	bundlePath := filepath.Join(t.TempDir(), "OneTiny.app")
	executablePath := filepath.Join(bundlePath, "Contents", "MacOS", "OneTiny")
	if err := os.MkdirAll(filepath.Dir(executablePath), 0o755); err != nil {
		t.Fatalf("MkdirAll executable dir: %v", err)
	}
	if err := os.WriteFile(executablePath, []byte("fake executable"), 0o755); err != nil {
		t.Fatalf("WriteFile executable: %v", err)
	}

	calls := 0
	fake := &fakeUpdateBackend{
		checkResult:    check,
		downloadResult: updater.DownloadResult{ZipPath: zipPath},
		stageResult:    updater.StageResult{CandidatePath: filepath.Join(t.TempDir(), "StagedOneTiny.app")},
	}
	svc := newUpdateTestService(t, fake)
	svc.currentExecutable = func() (string, error) {
		return executablePath, nil
	}
	svc.startUpdateHelper = func(string, updater.InstallPlan) error {
		calls++
		return nil
	}

	if _, err := svc.CheckUpdate(); err != nil {
		t.Fatalf("CheckUpdate returned error: %v", err)
	}
	if _, err := svc.DownloadUpdate(); err != nil {
		t.Fatalf("DownloadUpdate returned error: %v", err)
	}
	started, err := svc.StartUpdateInstall()
	if err != nil {
		t.Fatalf("StartUpdateInstall returned error: %v", err)
	}

	if _, err := svc.CheckUpdate(); err == nil {
		t.Fatal("CheckUpdate during installing error is nil, want installing error")
	}
	status, err := svc.GetUpdateStatus()
	if err != nil {
		t.Fatalf("GetUpdateStatus returned error: %v", err)
	}
	if status.State != "installing" {
		t.Fatalf("State after rejected CheckUpdate = %q, want installing", status.State)
	}
	if status.LogPath != started.LogPath {
		t.Fatalf("LogPath after rejected CheckUpdate = %q, want %q", status.LogPath, started.LogPath)
	}
	if svc.updateStage != (updater.StageResult{}) {
		t.Fatalf("updateStage after rejected CheckUpdate = %+v, want consumed stage", svc.updateStage)
	}
	if fake.checkCalls != 1 {
		t.Fatalf("CheckLatest calls = %d, want 1", fake.checkCalls)
	}
	if _, err := svc.StartUpdateInstall(); err == nil {
		t.Fatal("second StartUpdateInstall error is nil, want duplicate install error")
	}
	if calls != 1 {
		t.Fatalf("StartUpdateHelper calls = %d, want 1", calls)
	}
}

func TestRestartCommandForGUIMacAppPath(t *testing.T) {
	current := filepath.FromSlash("/Applications/OneTiny.app/Contents/MacOS/OneTiny")
	want := []string{"open", filepath.FromSlash("/Applications/OneTiny.app")}

	got := restartCommandForGUIForOS("darwin", current)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("restart command = %#v, want %#v", got, want)
	}
}
