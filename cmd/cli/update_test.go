package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/tcp404/OneTiny/internal/updater"
	"github.com/tcp404/OneTiny/internal/version"
	"github.com/urfave/cli/v2"
)

type fakeUpdateRunner struct {
	latestResult updater.CheckResult
	tagResult    updater.CheckResult
	tags         []string

	latestErr      error
	tagErr         error
	listErr        error
	downloadErr    error
	downloadResult updater.DownloadResult
	stageResult    updater.StageResult

	latestCalls   int
	tagCalls      int
	listCalls     int
	downloadCalls int

	latestOptions  []updater.CheckOptions
	tagOptions     []updater.CheckOptions
	downloadDirs   []string
	downloadChecks []updater.CheckResult
}

func (f *fakeUpdateRunner) CheckLatest(ctx context.Context, opts updater.CheckOptions) (updater.CheckResult, error) {
	f.latestCalls++
	f.latestOptions = append(f.latestOptions, opts)
	return f.latestResult, f.latestErr
}

func (f *fakeUpdateRunner) CheckTag(ctx context.Context, opts updater.CheckOptions) (updater.CheckResult, error) {
	f.tagCalls++
	f.tagOptions = append(f.tagOptions, opts)
	return f.tagResult, f.tagErr
}

func (f *fakeUpdateRunner) ListTags(ctx context.Context) ([]string, error) {
	f.listCalls++
	return f.tags, f.listErr
}

func (f *fakeUpdateRunner) DownloadAndStage(ctx context.Context, result updater.CheckResult, dir string) (updater.DownloadResult, updater.StageResult, error) {
	f.downloadCalls++
	f.downloadDirs = append(f.downloadDirs, dir)
	f.downloadChecks = append(f.downloadChecks, result)
	return f.downloadResult, f.stageResult, f.downloadErr
}

func TestUpdateListPrintsTagsAndReturnsExitCodeZero(t *testing.T) {
	runner := &fakeUpdateRunner{tags: []string{"v1.2.0", "v1.1.0"}}

	output, err := runUpdateCommand(t, runner, map[string]string{"list": "true"})

	requireExitCode(t, err, 0)
	if runner.listCalls != 1 {
		t.Fatalf("ListTags calls = %d, want 1", runner.listCalls)
	}
	if runner.latestCalls != 0 || runner.tagCalls != 0 {
		t.Fatalf("unexpected check calls: latest=%d tag=%d", runner.latestCalls, runner.tagCalls)
	}
	if !strings.Contains(output, "v1.2.0") || !strings.Contains(output, "v1.1.0") {
		t.Fatalf("update --list output = %q, want tag list", output)
	}
}

func TestUpdateCheckPrintsAvailableStatusAndReturnsExitCodeZero(t *testing.T) {
	runner := &fakeUpdateRunner{
		latestResult: updater.CheckResult{
			Availability: updater.Availability{
				Current:   "v1.0.0",
				Latest:    "v1.2.0",
				Known:     true,
				Available: true,
			},
		},
	}

	output, err := runUpdateCommand(t, runner, map[string]string{"check": "true"})

	requireExitCode(t, err, 0)
	if runner.latestCalls != 1 {
		t.Fatalf("CheckLatest calls = %d, want 1", runner.latestCalls)
	}
	if !strings.Contains(output, "发现新版本") || !strings.Contains(output, "v1.2.0") {
		t.Fatalf("update --check output = %q, want available status", output)
	}
	if strings.Contains(output, "下载和安装将在后续任务接入") {
		t.Fatalf("update --check output included install placeholder: %q", output)
	}
}

func TestUpdateCheckNoUpdatePrintsReasonOrLatestMessage(t *testing.T) {
	tests := []struct {
		name   string
		result updater.CheckResult
		want   string
	}{
		{
			name: "reason",
			result: updater.CheckResult{
				Availability: updater.Availability{
					Current: "dev",
					Latest:  "v1.2.0",
					Reason:  "unknown version",
				},
			},
			want: "unknown version",
		},
		{
			name: "already latest",
			result: updater.CheckResult{
				Availability: updater.Availability{
					Current: "v1.2.0",
					Latest:  "v1.2.0",
					Known:   true,
				},
			},
			want: "当前已是最新版本",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &fakeUpdateRunner{latestResult: tt.result}

			output, err := runUpdateCommand(t, runner, map[string]string{"check": "true"})

			requireExitCode(t, err, 0)
			if !strings.Contains(output, tt.want) {
				t.Fatalf("update --check output = %q, want %q", output, tt.want)
			}
		})
	}
}

func TestUpdateAvailableWithoutCheckCancelsInstallByDefaultOnEOF(t *testing.T) {
	runner := &fakeUpdateRunner{
		latestResult: availableUpdateResult(),
	}
	setConfirmInput(t, "")
	installCalls := stubStartCLIInstall(t, func(stage updater.StageResult) error {
		return nil
	})

	output, err := runUpdateCommand(t, runner, nil)

	requireExitCode(t, err, 0)
	if !strings.Contains(output, "发现新版本") || !strings.Contains(output, "已取消安装") {
		t.Fatalf("update output = %q, want available status and cancelled install", output)
	}
	if installCalls() != 0 {
		t.Fatalf("install starter calls = %d, want 0", installCalls())
	}
}

func TestUpdateDownloadOnlyDownloadsStagesAndPrintsPaths(t *testing.T) {
	runner := &fakeUpdateRunner{
		latestResult:   availableUpdateResult(),
		downloadResult: updater.DownloadResult{ZipPath: "/tmp/onetiny.zip"},
		stageResult: updater.StageResult{
			StagingDir:    "/tmp/onetiny-stage",
			CandidatePath: "/tmp/onetiny-stage/onetiny-cli",
		},
	}
	installCalls := stubStartCLIInstall(t, func(stage updater.StageResult) error {
		return nil
	})

	output, err := runUpdateCommand(t, runner, map[string]string{
		"download-only": "true",
		"output":        "/tmp/downloads",
	})

	requireExitCode(t, err, 0)
	if runner.downloadCalls != 1 {
		t.Fatalf("DownloadAndStage calls = %d, want 1", runner.downloadCalls)
	}
	if got := runner.downloadDirs[0]; got != "/tmp/downloads" {
		t.Fatalf("DownloadAndStage dir = %q, want /tmp/downloads", got)
	}
	if !strings.Contains(output, "更新包已下载: /tmp/onetiny.zip") {
		t.Fatalf("update --download-only output = %q, want downloaded zip path", output)
	}
	if !strings.Contains(output, "解压目录: /tmp/onetiny-stage") {
		t.Fatalf("update --download-only output = %q, want staging dir", output)
	}
	if installCalls() != 0 {
		t.Fatalf("install starter calls = %d, want 0", installCalls())
	}
}

func TestUpdateYesDownloadsStagesAndStartsInstall(t *testing.T) {
	wantStage := updater.StageResult{
		StagingDir:    "/tmp/onetiny-stage",
		CandidatePath: "/tmp/onetiny-stage/onetiny-cli",
	}
	runner := &fakeUpdateRunner{
		latestResult:   availableUpdateResult(),
		downloadResult: updater.DownloadResult{ZipPath: "/tmp/onetiny.zip"},
		stageResult:    wantStage,
	}
	var gotStage updater.StageResult
	installCalls := stubStartCLIInstall(t, func(stage updater.StageResult) error {
		gotStage = stage
		return nil
	})

	output, err := runUpdateCommand(t, runner, map[string]string{"yes": "true"})

	requireExitCode(t, err, 0)
	if runner.downloadCalls != 1 {
		t.Fatalf("DownloadAndStage calls = %d, want 1", runner.downloadCalls)
	}
	if installCalls() != 1 {
		t.Fatalf("install starter calls = %d, want 1", installCalls())
	}
	if gotStage != wantStage {
		t.Fatalf("install starter stage = %+v, want %+v", gotStage, wantStage)
	}
	if !strings.Contains(output, "更新安装已启动") {
		t.Fatalf("update --yes output = %q, want install started message", output)
	}
}

func TestUpdateUserDeclinesInstall(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "n", input: "n\n"},
		{name: "eof", input: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &fakeUpdateRunner{latestResult: availableUpdateResult()}
			setConfirmInput(t, tt.input)
			installCalls := stubStartCLIInstall(t, func(stage updater.StageResult) error {
				return nil
			})

			output, err := runUpdateCommand(t, runner, nil)

			requireExitCode(t, err, 0)
			if !strings.Contains(output, "已取消安装") {
				t.Fatalf("update output = %q, want cancelled install", output)
			}
			if installCalls() != 0 {
				t.Fatalf("install starter calls = %d, want 0", installCalls())
			}
		})
	}
}

func TestUpdateInstallStarterErrorReturnsExitCode31(t *testing.T) {
	runner := &fakeUpdateRunner{
		latestResult:   availableUpdateResult(),
		downloadResult: updater.DownloadResult{ZipPath: "/tmp/onetiny.zip"},
		stageResult: updater.StageResult{
			StagingDir:    "/tmp/onetiny-stage",
			CandidatePath: "/tmp/onetiny-stage/onetiny-cli",
		},
	}
	installCalls := stubStartCLIInstall(t, func(stage updater.StageResult) error {
		return errors.New("start failed")
	})

	_, err := runUpdateCommand(t, runner, map[string]string{"yes": "true"})

	requireExitCode(t, err, 31)
	if installCalls() != 1 {
		t.Fatalf("install starter calls = %d, want 1", installCalls())
	}
	if !strings.Contains(err.Error(), "start failed") {
		t.Fatalf("update error = %q, want starter error", err.Error())
	}
}

func TestPrepareCLIHelperCreatesPrivateTempDir(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "onetiny-cli")
	if err := os.WriteFile(sourcePath, []byte("helper-body"), 0o755); err != nil {
		t.Fatalf("write source executable: %v", err)
	}

	helperPath, err := prepareCLIHelper(sourcePath)
	if err != nil {
		t.Fatalf("prepareCLIHelper returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(filepath.Dir(helperPath))
	})

	wantName := "onetiny-cli-updater"
	if runtime.GOOS == "windows" {
		wantName += ".exe"
	}
	if got := filepath.Base(helperPath); got != wantName {
		t.Fatalf("helper filename = %q, want %q", got, wantName)
	}
	if got := filepath.Dir(helperPath); got == filepath.Clean(os.TempDir()) {
		t.Fatalf("helper path %q is directly in global temp dir", helperPath)
	}
	if got := filepath.Base(filepath.Dir(helperPath)); !strings.HasPrefix(got, "onetiny-cli-updater-") {
		t.Fatalf("helper temp dir = %q, want onetiny-cli-updater-*", got)
	}

	predictablePath := filepath.Join(os.TempDir(), fmt.Sprintf("onetiny-cli-updater-%d", os.Getpid()))
	if runtime.GOOS == "windows" {
		predictablePath += ".exe"
	}
	if helperPath == predictablePath {
		t.Fatalf("helper path = %q, want non-predictable private temp dir path", helperPath)
	}

	body, err := os.ReadFile(helperPath)
	if err != nil {
		t.Fatalf("read helper: %v", err)
	}
	if string(body) != "helper-body" {
		t.Fatalf("helper body = %q, want copied executable body", string(body))
	}
	info, err := os.Stat(filepath.Dir(helperPath))
	if err != nil {
		t.Fatalf("stat helper dir: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("helper dir mode = %o, want 700", got)
	}
}

func TestCopyExecutableDoesNotOverwriteExistingTarget(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "source")
	targetPath := filepath.Join(dir, "target")
	if err := os.WriteFile(sourcePath, []byte("new-helper"), 0o755); err != nil {
		t.Fatalf("write source executable: %v", err)
	}
	if err := os.WriteFile(targetPath, []byte("existing-helper"), 0o755); err != nil {
		t.Fatalf("write existing target: %v", err)
	}

	err := copyExecutable(sourcePath, targetPath)

	if err == nil {
		t.Fatalf("copyExecutable returned nil, want error for existing target")
	}
	body, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		t.Fatalf("read existing target: %v", readErr)
	}
	if string(body) != "existing-helper" {
		t.Fatalf("target body = %q, want existing-helper", string(body))
	}
}

func TestUpdateRunnerErrorReturnsExitCode31(t *testing.T) {
	runner := &fakeUpdateRunner{latestErr: errors.New("network down")}

	_, err := runUpdateCommand(t, runner, map[string]string{"check": "true"})

	requireExitCode(t, err, 31)
	if !strings.Contains(err.Error(), "network down") {
		t.Fatalf("update error = %q, want runner error", err.Error())
	}
}

func TestUpdateUseChecksTagInsteadOfLatest(t *testing.T) {
	runner := &fakeUpdateRunner{
		tagResult: updater.CheckResult{
			Availability: updater.Availability{
				Current: "v1.2.3",
				Latest:  "v1.2.3",
				Known:   true,
			},
		},
	}

	output, err := runUpdateCommand(t, runner, map[string]string{"check": "true", "use": "v1.2.3"})

	requireExitCode(t, err, 0)
	if runner.tagCalls != 1 {
		t.Fatalf("CheckTag calls = %d, want 1", runner.tagCalls)
	}
	if runner.latestCalls != 0 {
		t.Fatalf("CheckLatest calls = %d, want 0", runner.latestCalls)
	}
	if len(runner.tagOptions) != 1 {
		t.Fatalf("tag options count = %d, want 1", len(runner.tagOptions))
	}
	opts := runner.tagOptions[0]
	if opts.Tag != "v1.2.3" {
		t.Fatalf("CheckTag tag = %q, want v1.2.3", opts.Tag)
	}
	if opts.Channel != updater.ChannelCLI {
		t.Fatalf("CheckTag channel = %q, want %q", opts.Channel, updater.ChannelCLI)
	}
	if opts.CurrentVersion != version.Version {
		t.Fatalf("CheckTag current version = %q, want %q", opts.CurrentVersion, version.Version)
	}
	if opts.Platform != updater.CurrentPlatform() {
		t.Fatalf("CheckTag platform = %+v, want %+v", opts.Platform, updater.CurrentPlatform())
	}
	if !strings.Contains(output, "v1.2.3") {
		t.Fatalf("update --use output = %q, want requested tag", output)
	}
}

func availableUpdateResult() updater.CheckResult {
	return updater.CheckResult{
		Release: updater.Release{TagName: "v1.2.0"},
		Availability: updater.Availability{
			Current:   "v1.0.0",
			Latest:    "v1.2.0",
			Known:     true,
			Available: true,
		},
		Channel:  updater.ChannelCLI,
		Platform: updater.CurrentPlatform(),
	}
}

func stubStartCLIInstall(t *testing.T, fn func(updater.StageResult) error) func() int {
	t.Helper()
	oldStart := startCLIInstall
	calls := 0
	startCLIInstall = func(stage updater.StageResult) error {
		calls++
		return fn(stage)
	}
	t.Cleanup(func() {
		startCLIInstall = oldStart
	})
	return func() int {
		return calls
	}
}

func setConfirmInput(t *testing.T, input string) {
	t.Helper()
	oldInput := confirmInput
	confirmInput = strings.NewReader(input)
	t.Cleanup(func() {
		confirmInput = oldInput
	})
}

func runUpdateCommand(t *testing.T, runner *fakeUpdateRunner, values map[string]string) (string, error) {
	t.Helper()

	var output bytes.Buffer
	oldNoColor := color.NoColor
	color.NoColor = true
	t.Cleanup(func() {
		color.NoColor = oldNoColor
	})

	cmd := updateCmdWithRunner(runner, &output, &output)
	set := flag.NewFlagSet("update", flag.ContinueOnError)
	set.SetOutput(&output)
	for _, cliFlag := range cmd.Flags {
		if err := cliFlag.Apply(set); err != nil {
			t.Fatalf("apply flag: %v", err)
		}
	}
	for name, value := range values {
		if err := set.Set(name, value); err != nil {
			t.Fatalf("set flag %s: %v", name, err)
		}
	}

	ctx := cli.NewContext(cli.NewApp(), set, nil)
	err := cmd.Action(ctx)
	return output.String(), err
}

func requireExitCode(t *testing.T, err error, want int) {
	t.Helper()
	if err == nil {
		t.Fatalf("command returned nil, want cli exit code %d", want)
	}
	exitErr, ok := err.(cli.ExitCoder)
	if !ok {
		t.Fatalf("command returned %T %v, want cli.ExitCoder", err, err)
	}
	if got := exitErr.ExitCode(); got != want {
		t.Fatalf("exit code = %d, want %d", got, want)
	}
}
