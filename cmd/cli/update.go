package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/tcp404/OneTiny/internal/updater"
	"github.com/tcp404/OneTiny/internal/version"
	"github.com/urfave/cli/v2"
)

type updateRunner interface {
	CheckLatest(context.Context, updater.CheckOptions) (updater.CheckResult, error)
	CheckTag(context.Context, updater.CheckOptions) (updater.CheckResult, error)
	ListTags(context.Context) ([]string, error)
	DownloadAndStage(context.Context, updater.CheckResult, string) (updater.DownloadResult, updater.StageResult, error)
}

var (
	confirmInput    io.Reader = os.Stdin
	startCLIInstall           = realStartCLIInstall
)

func updateCmd() *cli.Command {
	return updateCmdWithRunner(updater.Service{}, os.Stdout, color.Output)
}

func updateCmdWithRunner(runner updateRunner, output io.Writer, _ io.Writer) *cli.Command {
	if output == nil {
		output = io.Discard
	}

	return &cli.Command{
		Name:    "update",
		Aliases: []string{"u", "up"},
		Usage:   "更新 OneTiny 到最新版",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "list",
				Aliases:     []string{"l"},
				Usage:       "列出远程服务器上所有可用版本",
				Required:    false,
				DefaultText: "false",
			},
			&cli.BoolFlag{
				Name:  "check",
				Usage: "只检查是否有可用更新",
			},
			&cli.BoolFlag{
				Name:  "download-only",
				Usage: "仅下载安装包，不安装",
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "自动确认更新操作",
			},
			&cli.StringFlag{
				Name:     "use",
				Usage:    "指定版本号",
				Required: false,
			},
			&cli.StringFlag{
				Name:  "output",
				Usage: "指定下载安装包输出目录",
			},
		},
		Action: func(c *cli.Context) error {
			return updateAction(c, runner, output)
		},
	}
}

func updateAction(c *cli.Context, runner updateRunner, output io.Writer) error {
	if c.Bool("list") {
		tags, err := runner.ListTags(commandContext(c))
		if err != nil {
			return cli.Exit(err.Error(), 31)
		}
		for _, tag := range tags {
			fmt.Fprintln(output, color.GreenString("%s", tag))
		}
		return cli.Exit("", 0)
	}

	opts := updater.CheckOptions{
		Channel:        updater.ChannelCLI,
		CurrentVersion: version.Version,
		Platform:       updater.CurrentPlatform(),
	}

	var (
		result updater.CheckResult
		err    error
	)
	if tag := strings.TrimSpace(c.String("use")); tag != "" {
		opts.Tag = tag
		result, err = runner.CheckTag(commandContext(c), opts)
	} else {
		result, err = runner.CheckLatest(commandContext(c), opts)
	}
	if err != nil {
		return cli.Exit(err.Error(), 31)
	}

	printUpdateCheckResult(output, result)
	if c.Bool("check") || !result.Availability.Available {
		return cli.Exit("", 0)
	}

	if !c.Bool("download-only") && !c.Bool("yes") && !confirmInstall(output) {
		fmt.Fprintln(output, color.YellowString("已取消安装"))
		return cli.Exit("", 0)
	}

	downloadResult, stageResult, err := runner.DownloadAndStage(commandContext(c), result, c.String("output"))
	if err != nil {
		return cli.Exit(err.Error(), 31)
	}

	if c.Bool("download-only") {
		fmt.Fprintln(output, color.GreenString("更新包已下载: %s", downloadResult.ZipPath))
		fmt.Fprintln(output, color.GreenString("解压目录: %s", stageResult.StagingDir))
		return cli.Exit("", 0)
	}

	if err := startCLIInstall(stageResult); err != nil {
		return cli.Exit(err.Error(), 31)
	}
	fmt.Fprintln(output, color.GreenString("更新安装已启动"))
	return cli.Exit("", 0)
}

func commandContext(c *cli.Context) context.Context {
	if c.Context != nil {
		return c.Context
	}
	return context.Background()
}

func printUpdateCheckResult(output io.Writer, result updater.CheckResult) {
	availability := result.Availability
	latest := availability.Latest
	if latest == "" {
		latest = result.Release.TagName
	}

	if availability.Available {
		if availability.Current != "" {
			fmt.Fprintln(output, color.GreenString("发现新版本: %s（当前版本: %s）", latest, availability.Current))
			return
		}
		fmt.Fprintln(output, color.GreenString("发现新版本: %s", latest))
		return
	}

	if strings.TrimSpace(availability.Reason) != "" {
		fmt.Fprintln(output, color.YellowString("%s", availability.Reason))
		return
	}

	if latest != "" {
		fmt.Fprintln(output, color.GreenString("当前已是最新版本: %s", latest))
		return
	}
	fmt.Fprintln(output, color.GreenString("当前已是最新版本"))
}

func confirmInstall(output io.Writer) bool {
	fmt.Fprint(output, "是否安装并退出当前命令？[y/N] ")
	answer, err := bufio.NewReader(confirmInput).ReadString('\n')
	if err != nil && err != io.EOF {
		return false
	}
	answer = strings.TrimSpace(answer)
	return strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes")
}

func realStartCLIInstall(stage updater.StageResult) error {
	current, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve current executable: %w", err)
	}

	plan, err := updater.BuildInstallPlan(updater.InstallOptions{
		Channel:         updater.ChannelCLI,
		CurrentPath:     current,
		ReplacementPath: stage.CandidatePath,
		PID:             os.Getpid(),
	})
	if err != nil {
		return err
	}

	helperPath, err := prepareCLIHelper(current)
	if err != nil {
		return err
	}
	return updater.StartHelper(helperPath, plan)
}

func prepareCLIHelper(currentPath string) (string, error) {
	helperDir, err := os.MkdirTemp("", "onetiny-cli-updater-*")
	if err != nil {
		return "", fmt.Errorf("create update helper dir: %w", err)
	}

	helperName := "onetiny-cli-updater"
	if runtime.GOOS == "windows" {
		helperName += ".exe"
	}
	helperPath := filepath.Join(helperDir, helperName)
	if err := copyExecutable(currentPath, helperPath); err != nil {
		return "", err
	}
	return helperPath, nil
}

func copyExecutable(sourcePath, targetPath string) (err error) {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open current executable: %w", err)
	}
	defer func() {
		err = errors.Join(err, wrapCloseError("close current executable", source.Close()))
	}()

	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		return fmt.Errorf("create update helper: %w", err)
	}
	defer func() {
		err = errors.Join(err, wrapCloseError("close update helper", target.Close()))
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

func wrapCloseError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
