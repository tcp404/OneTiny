package updater

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestBuildInstallPlanForBinary(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "onetiny-cli")
	replacement := filepath.Join(dir, "onetiny-cli-new")
	writeTestFile(t, current, "old")
	writeTestFile(t, replacement, "new")

	got, err := BuildInstallPlan(InstallOptions{
		Channel:         ChannelCLI,
		CurrentPath:     current,
		ReplacementPath: replacement,
		PID:             1234,
	})
	if err != nil {
		t.Fatalf("BuildInstallPlan returned error: %v", err)
	}

	if got.WaitPID != 1234 {
		t.Fatalf("WaitPID = %d, want 1234", got.WaitPID)
	}
	if got.TargetPath != current {
		t.Fatalf("TargetPath = %q, want %q", got.TargetPath, current)
	}
	if got.ReplacementPath != replacement {
		t.Fatalf("ReplacementPath = %q, want %q", got.ReplacementPath, replacement)
	}
	if got.BackupPath == "" {
		t.Fatal("BackupPath is empty")
	}
	if got.BackupPath != current+".bak" {
		t.Fatalf("BackupPath = %q, want %q", got.BackupPath, current+".bak")
	}
	if got.LogPath == "" {
		t.Fatal("LogPath is empty")
	}
}

func TestBuildInstallPlanForMacGUIAppBundle(t *testing.T) {
	current := filepath.FromSlash("/Applications/OneTiny.app/Contents/MacOS/OneTiny")
	replacement := filepath.FromSlash("/tmp/OneTiny.app")

	got, err := BuildInstallPlan(InstallOptions{
		Channel:         ChannelGUI,
		CurrentPath:     current,
		ReplacementPath: replacement,
		PID:             42,
	})
	if err != nil {
		t.Fatalf("BuildInstallPlan returned error: %v", err)
	}

	want := filepath.FromSlash("/Applications/OneTiny.app")
	if got.TargetPath != want {
		t.Fatalf("TargetPath = %q, want %q", got.TargetPath, want)
	}
}

func TestApplyInstallReplacesBinary(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "onetiny-cli")
	replacement := filepath.Join(dir, "onetiny-cli-new")
	backup := target + ".bak"
	writeTestFile(t, target, "old")
	writeTestFile(t, replacement, "new")

	if err := ApplyInstall(InstallPlan{
		TargetPath:      target,
		ReplacementPath: replacement,
		BackupPath:      backup,
	}); err != nil {
		t.Fatalf("ApplyInstall returned error: %v", err)
	}

	if got := readTestFile(t, target); got != "new" {
		t.Fatalf("target content = %q, want new", got)
	}
	assertPathMissing(t, backup)
	assertPathMissing(t, replacement)
}

func TestApplyInstallReplacesDirectoryBundle(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "Applications", "OneTiny.app")
	replacement := filepath.Join(dir, "staging", "OneTiny.app")
	backup := target + ".bak"
	writeTestFile(t, filepath.Join(target, "Contents", "MacOS", "OneTiny"), "old executable")
	writeTestFile(t, filepath.Join(target, "Contents", "Resources", "old.txt"), "old resource")
	writeTestFile(t, filepath.Join(replacement, "Contents", "MacOS", "OneTiny"), "new executable")
	writeTestFile(t, filepath.Join(replacement, "Contents", "Info.plist"), "new plist")

	if err := ApplyInstall(InstallPlan{
		TargetPath:      target,
		ReplacementPath: replacement,
		BackupPath:      backup,
	}); err != nil {
		t.Fatalf("ApplyInstall returned error: %v", err)
	}

	if got := readTestFile(t, filepath.Join(target, "Contents", "MacOS", "OneTiny")); got != "new executable" {
		t.Fatalf("target executable = %q, want new executable", got)
	}
	if got := readTestFile(t, filepath.Join(target, "Contents", "Info.plist")); got != "new plist" {
		t.Fatalf("target plist = %q, want new plist", got)
	}
	assertPathMissing(t, filepath.Join(target, "Contents", "Resources", "old.txt"))
	assertPathMissing(t, backup)
	assertPathMissing(t, replacement)
}

func TestPrepareReplacementForInstallCopiesIntoTargetDir(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "install", "onetiny-cli")
	replacement := filepath.Join(dir, "staging", "onetiny-cli")
	writeTestFile(t, target, "old")
	writeTestFile(t, replacement, "new")

	prepared, cleanup, err := prepareReplacementForInstall(replacement, target)
	if err != nil {
		t.Fatalf("prepareReplacementForInstall returned error: %v", err)
	}
	defer cleanup()

	if filepath.Dir(prepared) == filepath.Dir(replacement) {
		t.Fatalf("prepared replacement dir = %q, want target-side temp dir", filepath.Dir(prepared))
	}
	if got := readTestFile(t, prepared); got != "new" {
		t.Fatalf("prepared content = %q, want new", got)
	}
	if got := readTestFile(t, replacement); got != "new" {
		t.Fatalf("source replacement content = %q, want preserved source", got)
	}
}

func TestApplyInstallRollsBackWhenReplacementMissing(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "onetiny-cli")
	replacement := filepath.Join(dir, "missing-new")
	backup := target + ".bak"
	writeTestFile(t, target, "old")

	err := ApplyInstall(InstallPlan{
		TargetPath:      target,
		ReplacementPath: replacement,
		BackupPath:      backup,
	})
	if err == nil {
		t.Fatal("ApplyInstall returned nil error, want missing replacement error")
	}
	if got := readTestFile(t, target); got != "old" {
		t.Fatalf("target content after rollback = %q, want old", got)
	}
	assertPathMissing(t, backup)
}

func TestApplyInstallIgnoresBackupCleanupErrorAfterReplacement(t *testing.T) {
	originalOps := installOps
	ops := &backupCleanupFailureInstallOps{}
	installOps = ops
	t.Cleanup(func() {
		installOps = originalOps
	})

	err := ApplyInstall(InstallPlan{
		TargetPath:      "target",
		ReplacementPath: "replacement",
		BackupPath:      "backup",
	})
	if err != nil {
		t.Fatalf("ApplyInstall returned error after replacement succeeded: %v", err)
	}
	if ops.replaceAttempts != 1 {
		t.Fatalf("replacement rename attempts = %d, want 1", ops.replaceAttempts)
	}
	if ops.cleanupAttempts != 2 {
		t.Fatalf("backup cleanup attempts = %d, want 2", ops.cleanupAttempts)
	}
}

func TestApplyInstallCanRetryAfterRollback(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "onetiny-cli")
	replacement := filepath.Join(dir, "onetiny-cli-new")
	backup := target + ".bak"
	plan := InstallPlan{
		TargetPath:      target,
		ReplacementPath: replacement,
		BackupPath:      backup,
	}
	writeTestFile(t, target, "old")

	if err := ApplyInstall(plan); err == nil {
		t.Fatal("first ApplyInstall returned nil error, want missing replacement error")
	}
	if got := readTestFile(t, target); got != "old" {
		t.Fatalf("target content after rollback = %q, want old", got)
	}

	writeTestFile(t, replacement, "new")
	if err := ApplyInstall(plan); err != nil {
		t.Fatalf("second ApplyInstall returned error: %v", err)
	}
	if got := readTestFile(t, target); got != "new" {
		t.Fatalf("target content after retry = %q, want new", got)
	}
	assertPathMissing(t, backup)
}

func TestRunHelperStopsRetryAfterRollbackFailurePreservesBackup(t *testing.T) {
	originalOps := installOps
	originalRetryTimeout := helperApplyRetryTimeout
	originalRetryInterval := helperApplyRetryInterval
	ops := &rollbackFailureInstallOps{targetExists: true, replacementExists: true}
	installOps = ops
	helperApplyRetryTimeout = 10 * time.Millisecond
	helperApplyRetryInterval = time.Millisecond
	t.Cleanup(func() {
		installOps = originalOps
		helperApplyRetryTimeout = originalRetryTimeout
		helperApplyRetryInterval = originalRetryInterval
	})

	err := RunHelper(InstallPlan{
		TargetPath:      "target",
		ReplacementPath: "replacement",
		BackupPath:      "backup",
	})
	if !errors.Is(err, ErrInstallRollbackFailed) {
		t.Fatalf("RunHelper error = %v, want %v", err, ErrInstallRollbackFailed)
	}
	if ops.removeBackupCalls != 1 {
		t.Fatalf("backup RemoveAll calls = %d, want 1", ops.removeBackupCalls)
	}
	if ops.replaceAttempts != 1 {
		t.Fatalf("replacement rename attempts = %d, want 1", ops.replaceAttempts)
	}
	if ops.rollbackAttempts != 1 {
		t.Fatalf("rollback rename attempts = %d, want 1", ops.rollbackAttempts)
	}
	if !ops.backupExists {
		t.Fatal("backup was removed after rollback failure")
	}
}

func TestWaitForProcessExitReturnsErrorOnTimeout(t *testing.T) {
	err := waitForProcessExit(os.Getpid(), time.Nanosecond)
	if err == nil {
		t.Fatal("waitForProcessExit returned nil error, want timeout")
	}
	if !strings.Contains(err.Error(), "timed out waiting for process") {
		t.Fatalf("waitForProcessExit error = %q, want timeout error", err)
	}
}

func TestWaitForProcessExitPollsUntilProcessExits(t *testing.T) {
	originalProcessRunning := processRunning
	calls := 0
	processRunning = func(pid int) (bool, error) {
		calls++
		if pid != 1234 {
			t.Fatalf("pid = %d, want 1234", pid)
		}
		return calls < 3, nil
	}
	t.Cleanup(func() {
		processRunning = originalProcessRunning
	})

	if err := waitForProcessExit(1234, time.Second); err != nil {
		t.Fatalf("waitForProcessExit returned error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("processRunning calls = %d, want 3", calls)
	}
}

func TestRunHelperStopsInstallWhenWaitTimesOut(t *testing.T) {
	originalTimeout := helperWaitTimeout
	helperWaitTimeout = time.Nanosecond
	t.Cleanup(func() {
		helperWaitTimeout = originalTimeout
	})

	dir := t.TempDir()
	target := filepath.Join(dir, "onetiny-cli")
	replacement := filepath.Join(dir, "onetiny-cli-new")
	backup := target + ".bak"
	writeTestFile(t, target, "old")
	writeTestFile(t, replacement, "new")

	err := RunHelper(InstallPlan{
		WaitPID:         os.Getpid(),
		TargetPath:      target,
		ReplacementPath: replacement,
		BackupPath:      backup,
	})
	if err == nil {
		t.Fatal("RunHelper returned nil error, want wait timeout")
	}
	if !strings.Contains(err.Error(), "timed out waiting for process") {
		t.Fatalf("RunHelper error = %q, want timeout error", err)
	}
	if got := readTestFile(t, target); got != "old" {
		t.Fatalf("target content after wait timeout = %q, want old", got)
	}
	if got := readTestFile(t, replacement); got != "new" {
		t.Fatalf("replacement content after wait timeout = %q, want new", got)
	}
	assertPathMissing(t, backup)
}

func TestRunHelperWritesHelperLog(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "onetiny-cli")
	replacement := filepath.Join(dir, "onetiny-cli-new")
	backup := target + ".bak"
	logPath := filepath.Join(dir, "update.log")
	writeTestFile(t, target, "old")
	writeTestFile(t, replacement, "new")

	if err := RunHelper(InstallPlan{
		TargetPath:      target,
		ReplacementPath: replacement,
		BackupPath:      backup,
		LogPath:         logPath,
	}); err != nil {
		t.Fatalf("RunHelper returned error: %v", err)
	}

	body := readTestFile(t, logPath)
	if !strings.Contains(body, "update install applied") {
		t.Fatalf("helper log = %q, want success entry", body)
	}
	if !strings.Contains(body, "target=") || !strings.Contains(body, "replacement=") || !strings.Contains(body, "backup=") {
		t.Fatalf("helper log = %q, want install paths", body)
	}
}

func TestStartDetachedReleasesProcessAfterStart(t *testing.T) {
	command := &fakeDetachedCommand{}

	if err := startDetached(command); err != nil {
		t.Fatalf("startDetached returned error: %v", err)
	}
	if !command.started {
		t.Fatal("command was not started")
	}
	if !command.released {
		t.Fatal("command process was not released")
	}
}

func TestStartDetachedReturnsReleaseError(t *testing.T) {
	releaseErr := errors.New("release failed")
	command := &fakeDetachedCommand{releaseErr: releaseErr}

	err := startDetached(command)
	if !errors.Is(err, releaseErr) {
		t.Fatalf("startDetached error = %v, want %v", err, releaseErr)
	}
	if !command.started {
		t.Fatal("command was not started")
	}
	if !command.released {
		t.Fatal("command process was not released")
	}
}

func TestHelperArgsParseRoundTrip(t *testing.T) {
	plan := InstallPlan{
		WaitPID:         5678,
		TargetPath:      filepath.Join("target", "OneTiny.app"),
		ReplacementPath: filepath.Join("staging", "OneTiny.app"),
		BackupPath:      filepath.Join("target", "OneTiny.app.bak"),
		LogPath:         filepath.Join("logs", "update.log"),
		RestartCommand:  []string{"open", "-a", "OneTiny"},
	}

	args := HelperArgs(plan)
	if !IsHelperInvocation(args) {
		t.Fatalf("IsHelperInvocation(%#v) = false, want true", args)
	}
	if IsHelperInvocation([]string{"--target", plan.TargetPath}) {
		t.Fatal("IsHelperInvocation without helper flag = true, want false")
	}

	got, err := ParseHelperArgs(args)
	if err != nil {
		t.Fatalf("ParseHelperArgs returned error: %v", err)
	}
	if !reflect.DeepEqual(got, plan) {
		t.Fatalf("parsed plan = %#v, want %#v", got, plan)
	}
}

func writeTestFile(t *testing.T, path, body string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent dir for %q: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}
	return string(body)
}

func assertPathMissing(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("path %q exists or stat failed with unexpected error: %v", path, err)
	}
}

type rollbackFailureInstallOps struct {
	targetExists      bool
	replacementExists bool
	backupExists      bool
	removeBackupCalls int
	replaceAttempts   int
	rollbackAttempts  int
}

type backupCleanupFailureInstallOps struct {
	replaceAttempts int
	cleanupAttempts int
}

func (ops *backupCleanupFailureInstallOps) RemoveAll(path string) error {
	if path == "backup" {
		ops.cleanupAttempts++
		if ops.cleanupAttempts == 2 {
			return errors.New("backup cleanup failed")
		}
	}
	return nil
}

func (ops *backupCleanupFailureInstallOps) Rename(oldPath, newPath string) error {
	switch {
	case oldPath == "target" && newPath == "backup":
		return nil
	case oldPath == "replacement" && newPath == "target":
		ops.replaceAttempts++
		return nil
	default:
		return errors.New("unexpected rename")
	}
}

func (ops *rollbackFailureInstallOps) RemoveAll(path string) error {
	if path == "backup" {
		ops.removeBackupCalls++
		ops.backupExists = false
	}
	return nil
}

func (ops *rollbackFailureInstallOps) Rename(oldPath, newPath string) error {
	switch {
	case oldPath == "target" && newPath == "backup":
		if !ops.targetExists {
			return errors.New("target missing")
		}
		ops.targetExists = false
		ops.backupExists = true
		return nil
	case oldPath == "replacement" && newPath == "target":
		ops.replaceAttempts++
		if !ops.replacementExists {
			return errors.New("replacement missing")
		}
		return errors.New("replacement rename failed")
	case oldPath == "backup" && newPath == "target":
		ops.rollbackAttempts++
		return errors.New("rollback rename failed")
	default:
		return errors.New("unexpected rename")
	}
}

type fakeDetachedCommand struct {
	startErr   error
	releaseErr error
	started    bool
	released   bool
}

func (command *fakeDetachedCommand) Start() error {
	command.started = true
	return command.startErr
}

func (command *fakeDetachedCommand) Release() error {
	command.released = true
	return command.releaseErr
}
