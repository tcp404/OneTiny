package main

import (
	"log"
	"os"

	"github.com/tcp404/OneTiny/internal/accesslog"
	"github.com/tcp404/OneTiny/internal/app"
	"github.com/tcp404/OneTiny/internal/config"
	"github.com/tcp404/OneTiny/internal/gui"
	"github.com/tcp404/OneTiny/internal/gui/webassets"
	"github.com/tcp404/OneTiny/internal/runtime"
	"github.com/tcp404/OneTiny/internal/server"
	"github.com/tcp404/OneTiny/internal/updater"
)

func main() {
	if updater.IsHelperInvocation(os.Args[1:]) {
		if err := updater.RunHelperFromArgs(os.Args[1:]); err != nil {
			log.Fatal(err)
		}
		return
	}

	configPath, err := config.DefaultPath()
	if err != nil {
		log.Fatal(err)
	}
	configDir, err := config.DefaultDir()
	if err != nil {
		log.Fatal(err)
	}
	store := config.NewStore(configPath)
	cfg, err := store.Load()
	if err != nil {
		log.Fatal(err)
	}
	runtimeState := runtime.New(runtime.SnapshotFromConfig(runtimeConfigFromConfig(cfg), runtime.NewProcess()))
	logger := accesslog.New(accesslog.DefaultPath())
	appService := app.NewService(app.Dependencies{
		ConfigStore: store,
		Runtime:     runtimeState,
		Manager:     server.NewManagerWithDependencies(server.Dependencies{Runtime: runtimeState, AccessLog: logger}),
		Logger:      logger,
	})
	if err := gui.Run(webassets.Assets, gui.Dependencies{AppService: appService, ConfigDir: configDir}); err != nil {
		log.Fatal(err)
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
