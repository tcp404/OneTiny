package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tcp404/OneTiny/internal/accesslog"
	"github.com/tcp404/OneTiny/internal/app/validation"
	"github.com/tcp404/OneTiny/internal/config"
	"github.com/tcp404/OneTiny/internal/runtime"
	"github.com/tcp404/OneTiny/internal/server"
	"github.com/tcp404/OneTiny/internal/updater"

	"github.com/fatih/color"
)

func main() {
	if updater.IsHelperInvocation(os.Args[1:]) {
		if err := updater.RunHelperFromArgs(os.Args[1:]); err != nil {
			log.Fatal(err)
		}
		return
	}

	var err error
	defer func() {
		if err != nil {
			log.Println(color.RedString("%v", err))
		}
	}()

	configPath, err := config.DefaultPath()
	if err != nil {
		return
	}
	store := config.NewStore(configPath)
	cfg, err := store.Load()
	if err != nil {
		return
	}
	runtimeState := runtime.New(runtime.SnapshotFromConfig(runtimeConfigFromConfig(cfg), runtime.NewProcess()))

	if err = CLI(store, runtimeState).Run(os.Args); err != nil {
		return
	}

	snapshot := runtimeState.Snapshot()
	if err = store.ValidateSecureConfigFor(snapshot.IsSecure); err != nil {
		return
	}

	if err = validation.ValidatePort(snapshot.Port); err != nil {
		return
	}
	if err = validation.ValidatePath(snapshot.RootPath); err != nil {
		return
	}

	logger := accesslog.New(accesslog.DefaultPath())
	manager := server.NewManagerWithDependencies(server.Dependencies{Runtime: runtimeState, AccessLog: logger})
	if err = manager.Start(); err != nil {
		return
	}
	printInfo(snapshot)

	q := make(chan os.Signal, 1)
	signal.Notify(q, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(q)

	select {
	case <-q:
		if stopErr := manager.Stop(); stopErr != nil {
			log.Println(color.RedString(stopErr.Error()))
		}
		fmt.Println(color.GreenString("\nbye~"))
	case err = <-manager.Done():
		return
	}
}

func runtimeConfigFromConfig(cfg config.Config) runtime.PersistentConfig {
	sizeBytes, _ := config.ParseByteSize(cfg.ScratchMaxItemSize)
	return runtime.PersistentConfig{
		RootPath:            cfg.RootPath,
		Port:                cfg.Port,
		MaxLevel:            cfg.MaxLevel,
		IsAllowUpload:       cfg.IsAllowUpload,
		IsSecure:            cfg.IsSecure,
		Username:            cfg.Username,
		PasswordHash:        cfg.PasswordHash,
		ScratchMaxItems:     cfg.ScratchMaxItems,
		ScratchMaxItemSize:  cfg.ScratchMaxItemSize,
		ScratchMaxItemBytes: sizeBytes,
	}
}

func printInfo(snapshot runtime.Snapshot) {
	log.SetOutput(color.Output)
	host := snapshot.IP
	if host == "" {
		host = "127.0.0.1"
	}
	if snapshot.IP != "" {
		log.Printf("Run on   [ %s ]", color.BlueString("http://%s:%d", snapshot.IP, snapshot.Port))
	} else {
		log.Printf("%s", color.YellowString("Warning: [ 暂时获取不到您的IP，可以打开新的命令行窗口输入 ->  ipconfig , 查看您的IP。]"))
	}

	log.Printf("Run with [ %s ]", color.BlueString("%s", snapshot.RootPath))
	log.Printf("Scratch list [ %s ]", color.BlueString("http://%s:%d/scratch/", host, snapshot.Port))
	log.Printf("Scratch items: [ %s ]", color.BlueString("%d", snapshot.ScratchMaxItems))
	log.Printf("Scratch item size: [ %s ]", color.BlueString("%s", snapshot.ScratchMaxItemSize))
	log.Printf("Allow access level: [ %s ]", color.BlueString("%d", snapshot.MaxLevel))

	status := color.RedString("%t", snapshot.IsAllowUpload)
	if snapshot.IsAllowUpload {
		status = color.GreenString("%t", snapshot.IsAllowUpload)
	}
	log.Printf("Allow upload: [ %s ]", status)

	status = color.RedString("%t", snapshot.IsSecure)
	if snapshot.IsSecure {
		status = color.GreenString("%t", snapshot.IsSecure)
	}
	log.Printf("Need Login: [ %s ]\n\n", status)
}
