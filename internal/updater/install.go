package updater

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

const helperApplyFlag = "--onetiny-apply-update"

var (
	ErrInstallRollbackFailed = errors.New("install rollback failed")

	helperWaitTimeout                       = 30 * time.Second
	helperApplyRetryTimeout                 = 5 * time.Second
	helperApplyRetryInterval                = 200 * time.Millisecond
	installOps               installFileOps = osInstallFileOps{}
	processRunning                          = isProcessRunning
)

type installFileOps interface {
	RemoveAll(path string) error
	Rename(oldPath, newPath string) error
}

type osInstallFileOps struct{}

func (osInstallFileOps) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (osInstallFileOps) Rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

type InstallOptions struct {
	Channel         Channel
	CurrentPath     string
	ReplacementPath string
	PID             int
	RestartCommand  []string
}

type InstallPlan struct {
	WaitPID         int
	TargetPath      string
	ReplacementPath string
	BackupPath      string
	LogPath         string
	RestartCommand  []string
}

func BuildInstallPlan(options InstallOptions) (InstallPlan, error) {
	if options.CurrentPath == "" {
		return InstallPlan{}, fmt.Errorf("current path is required")
	}
	if options.ReplacementPath == "" {
		return InstallPlan{}, fmt.Errorf("replacement path is required")
	}

	targetPath := filepath.Clean(options.CurrentPath)
	if options.Channel == ChannelGUI {
		if bundlePath, ok := macAppBundlePath(targetPath); ok {
			targetPath = bundlePath
		}
	}

	return InstallPlan{
		WaitPID:         options.PID,
		TargetPath:      targetPath,
		ReplacementPath: filepath.Clean(options.ReplacementPath),
		BackupPath:      targetPath + ".bak",
		LogPath:         filepath.Join(os.TempDir(), "onetiny-update-"+strconv.Itoa(os.Getpid())+".log"),
		RestartCommand:  append([]string(nil), options.RestartCommand...),
	}, nil
}

func HelperArgs(plan InstallPlan) []string {
	args := []string{
		helperApplyFlag,
		"--wait-pid", strconv.Itoa(plan.WaitPID),
		"--target", plan.TargetPath,
		"--replacement", plan.ReplacementPath,
		"--backup", plan.BackupPath,
		"--log", plan.LogPath,
	}
	if len(plan.RestartCommand) != 0 {
		args = append(args, "--restart-command", encodeRestartCommand(plan.RestartCommand))
	}
	return args
}

func ParseHelperArgs(args []string) (InstallPlan, error) {
	flags := flag.NewFlagSet("onetiny-update-helper", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var apply bool
	var restartCommand string
	plan := InstallPlan{}
	flags.BoolVar(&apply, strings.TrimPrefix(helperApplyFlag, "--"), false, "apply a staged OneTiny update")
	flags.IntVar(&plan.WaitPID, "wait-pid", 0, "process ID to wait for before applying the update")
	flags.StringVar(&plan.TargetPath, "target", "", "current install target path")
	flags.StringVar(&plan.ReplacementPath, "replacement", "", "staged replacement path")
	flags.StringVar(&plan.BackupPath, "backup", "", "temporary backup path")
	flags.StringVar(&plan.LogPath, "log", "", "helper log path")
	flags.StringVar(&restartCommand, "restart-command", "", "base64url JSON encoded restart command")

	if err := flags.Parse(args); err != nil {
		return InstallPlan{}, err
	}
	if !apply {
		return InstallPlan{}, fmt.Errorf("missing %s", helperApplyFlag)
	}
	if plan.TargetPath == "" {
		return InstallPlan{}, fmt.Errorf("target path is required")
	}
	if plan.ReplacementPath == "" {
		return InstallPlan{}, fmt.Errorf("replacement path is required")
	}
	if plan.BackupPath == "" {
		return InstallPlan{}, fmt.Errorf("backup path is required")
	}
	if restartCommand != "" {
		decoded, err := decodeRestartCommand(restartCommand)
		if err != nil {
			return InstallPlan{}, err
		}
		plan.RestartCommand = decoded
	}
	return plan, nil
}

func IsHelperInvocation(args []string) bool {
	return slices.Contains(args, helperApplyFlag)
}

func StartHelper(helperPath string, plan InstallPlan) error {
	if helperPath == "" {
		return fmt.Errorf("helper path is required")
	}
	cmd := exec.Command(helperPath, HelperArgs(plan)...)
	if err := startDetached(execDetachedCommand{cmd: cmd}); err != nil {
		return fmt.Errorf("start update helper: %w", err)
	}
	return nil
}

func RunHelperFromArgs(args []string) error {
	plan, err := ParseHelperArgs(args)
	if err != nil {
		return err
	}
	return RunHelper(plan)
}

func RunHelper(plan InstallPlan) error {
	helperLog(plan, "update helper started")
	helperLog(
		plan,
		"install plan: wait_pid=%d target=%q replacement=%q backup=%q restart=%q",
		plan.WaitPID,
		plan.TargetPath,
		plan.ReplacementPath,
		plan.BackupPath,
		strings.Join(plan.RestartCommand, " "),
	)
	if err := waitForProcessExit(plan.WaitPID, helperWaitTimeout); err != nil {
		helperLog(plan, "wait for process failed: %v", err)
		return err
	}

	var lastErr error
	deadline := time.Now().Add(helperApplyRetryTimeout)
	for {
		helperLog(plan, "applying update: target=%q replacement=%q backup=%q", plan.TargetPath, plan.ReplacementPath, plan.BackupPath)
		lastErr = ApplyInstall(plan)
		if lastErr == nil {
			helperLog(plan, "update install applied: %s", plan.TargetPath)
			break
		}
		helperLog(plan, "update install attempt failed: %v", lastErr)
		if errors.Is(lastErr, ErrInstallRollbackFailed) {
			return lastErr
		}
		if time.Now().After(deadline) {
			return lastErr
		}
		time.Sleep(helperApplyRetryInterval)
	}

	if len(plan.RestartCommand) != 0 {
		cmd := exec.Command(plan.RestartCommand[0], plan.RestartCommand[1:]...)
		if err := startDetached(execDetachedCommand{cmd: cmd}); err != nil {
			helperLog(plan, "restart updated app failed: %v", err)
			return fmt.Errorf("restart updated app: %w", err)
		}
		helperLog(plan, "restart command started: %s", strings.Join(plan.RestartCommand, " "))
	}
	return nil
}

func ApplyInstall(plan InstallPlan) error {
	if plan.TargetPath == "" {
		return fmt.Errorf("target path is required")
	}
	if plan.ReplacementPath == "" {
		return fmt.Errorf("replacement path is required")
	}
	if plan.BackupPath == "" {
		return fmt.Errorf("backup path is required")
	}

	preparedReplacement, cleanupPrepared, err := prepareReplacementForInstall(plan.ReplacementPath, plan.TargetPath)
	if err != nil {
		return err
	}
	defer cleanupPrepared()

	if err := installOps.RemoveAll(plan.BackupPath); err != nil {
		return fmt.Errorf("remove previous backup: %w", err)
	}
	if err := installOps.Rename(plan.TargetPath, plan.BackupPath); err != nil {
		return fmt.Errorf("backup current install: %w", err)
	}
	if err := installOps.Rename(preparedReplacement, plan.TargetPath); err != nil {
		if rollbackErr := installOps.Rename(plan.BackupPath, plan.TargetPath); rollbackErr != nil {
			return fmt.Errorf("%w: replace install: %v; rollback failed: %v", ErrInstallRollbackFailed, err, rollbackErr)
		}
		return fmt.Errorf("replace install: %w", err)
	}
	_ = installOps.RemoveAll(plan.BackupPath)
	if preparedReplacement != filepath.Clean(plan.ReplacementPath) {
		_ = installOps.RemoveAll(plan.ReplacementPath)
	}
	return nil
}

func prepareReplacementForInstall(replacementPath, targetPath string) (string, func(), error) {
	replacementPath = filepath.Clean(replacementPath)
	targetPath = filepath.Clean(targetPath)
	if sameDirectory(replacementPath, targetPath) {
		return replacementPath, func() {}, nil
	}

	tempRoot, err := os.MkdirTemp(filepath.Dir(targetPath), "."+filepath.Base(targetPath)+".new-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("create target-side replacement temp dir: %w", err)
	}
	cleanup := func() {
		_ = os.RemoveAll(tempRoot)
	}

	preparedPath := filepath.Join(tempRoot, filepath.Base(targetPath))
	if err := copyPath(replacementPath, preparedPath); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("prepare target-side replacement: %w", err)
	}
	return preparedPath, cleanup, nil
}

func sameDirectory(leftPath, rightPath string) bool {
	leftDir, leftErr := filepath.Abs(filepath.Dir(leftPath))
	rightDir, rightErr := filepath.Abs(filepath.Dir(rightPath))
	if leftErr != nil || rightErr != nil {
		return filepath.Clean(filepath.Dir(leftPath)) == filepath.Clean(filepath.Dir(rightPath))
	}
	return leftDir == rightDir
}

func copyPath(sourcePath, targetPath string) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDirectory(sourcePath, targetPath)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("unsupported replacement entry %q", sourcePath)
	}
	return copyRegularFile(sourcePath, targetPath, info.Mode())
}

func copyDirectory(sourceDir, targetDir string) error {
	return filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(targetDir, rel)
		if entry.IsDir() {
			perm := info.Mode().Perm()
			if perm == 0 {
				perm = 0o755
			}
			return os.MkdirAll(target, perm)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported replacement entry %q", path)
		}
		return copyRegularFile(path, target, info.Mode())
	})
}

func copyRegularFile(sourcePath, targetPath string, mode os.FileMode) (err error) {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	perm := mode.Perm()
	if perm == 0 {
		perm = 0o644
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, source.Close())
	}()

	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, target.Close())
	}()

	if _, err := io.Copy(target, source); err != nil {
		return err
	}
	if err := target.Chmod(perm); err != nil {
		return err
	}
	return target.Sync()
}

func helperLog(plan InstallPlan, format string, args ...any) {
	logPath := strings.TrimSpace(plan.LogPath)
	if logPath == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = fmt.Fprintf(file, "%s %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
}

type detachedCommand interface {
	Start() error
	Release() error
}

type execDetachedCommand struct {
	cmd *exec.Cmd
}

func (command execDetachedCommand) Start() error {
	return command.cmd.Start()
}

func (command execDetachedCommand) Release() error {
	return command.cmd.Process.Release()
}

func startDetached(command detachedCommand) error {
	if err := command.Start(); err != nil {
		return err
	}
	return command.Release()
}

func macAppBundlePath(path string) (string, bool) {
	slashPath := filepath.ToSlash(filepath.Clean(path))
	const marker = ".app/Contents/MacOS/"
	index := strings.Index(slashPath, marker)
	if index < 0 {
		return "", false
	}
	return filepath.FromSlash(slashPath[:index+len(".app")]), true
}

func encodeRestartCommand(command []string) string {
	body, _ := json.Marshal(command)
	return base64.RawURLEncoding.EncodeToString(body)
}

func decodeRestartCommand(encoded string) ([]string, error) {
	body, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode restart command: %w", err)
	}
	var command []string
	if err := json.Unmarshal(body, &command); err != nil {
		return nil, fmt.Errorf("parse restart command: %w", err)
	}
	return command, nil
}

func waitForProcessExit(pid int, timeout time.Duration) error {
	if pid <= 0 {
		return nil
	}

	deadline := time.Now().Add(timeout)
	for {
		running, err := processRunning(pid)
		if err != nil {
			return fmt.Errorf("check process %d: %w", pid, err)
		}
		if !running {
			return nil
		}
		if timeout <= 0 || !time.Now().Before(deadline) {
			return fmt.Errorf("timed out waiting for process %d to exit", pid)
		}
		sleep := 100 * time.Millisecond
		if remaining := time.Until(deadline); remaining < sleep {
			sleep = remaining
		}
		time.Sleep(sleep)
	}
}
