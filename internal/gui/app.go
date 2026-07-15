package gui

import (
	"embed"
	"io/fs"
	stdruntime "runtime"
	"sync/atomic"

	appsvc "github.com/tcp404/OneTiny/internal/app"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const singleInstanceID = "com.tcp404.onetiny.gui"

type Dependencies struct {
	AppService *appsvc.Service
	ConfigDir  string
}

func Run(assets embed.FS, deps Dependencies) error {
	service := NewService(deps.AppService, nil)
	var quitting atomic.Bool

	var app *application.App
	var window application.Window
	openPanel := func() {
		if window == nil {
			return
		}
		window.Show().Focus()
	}
	app = application.New(newApplicationOptions(service, assets, func() bool {
		if !quitting.Load() && !service.requestQuit(func() {
			quitting.Store(true)
			app.Quit()
		}) {
			return false
		}
		quitting.Store(true)
		service.shutdown()
		return true
	}, service.shutdown, openPanel))
	app.SetIcon(appIcon)

	window = app.Window.NewWithOptions(newMainWindowOptions())
	service.setDialogAdapter(NewWailsDialogAdapter(app, window, deps.ConfigDir))
	service.setQuitForUpdate(func() {
		quitting.Store(true)
		service.shutdown()
		app.Quit()
	})

	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if quitting.Load() {
			return
		}
		event.Cancel()
		window.Hide()
	})

	quit := func() {
		if stdruntime.GOOS == "windows" {
			if status, err := service.GetStatus(); err == nil && status.Running {
				openPanel()
			}
		}
		if !service.requestQuit(func() {
			quitting.Store(true)
			app.Quit()
		}) {
			return
		}
		quitting.Store(true)
		app.Quit()
	}

	tray := app.SystemTray.New()
	tray.SetTooltip("OneTiny")
	tray.SetIcon(appIcon)
	tray.SetDarkModeIcon(appIcon)
	tray.AttachWindow(window).WindowOffset(8)
	tray.OnClick(openPanel)
	tray.OnRightClick(tray.ShowMenu)

	menu := app.NewMenu()
	menu.Add("打开面板").OnClick(func(_ *application.Context) {
		openPanel()
	})
	menu.AddSeparator()
	menu.Add("退出").OnClick(func(_ *application.Context) {
		quit()
	})
	tray.SetMenu(menu)

	return app.Run()
}

func newApplicationOptions(service *Service, assets fs.FS, shouldQuit func() bool, onShutdown func(), onSecondInstanceLaunch func()) application.Options {
	return application.Options{
		Name:        "OneTiny",
		Description: "OneTiny 桌面控制面板",
		Icon:        appIcon,
		Services: []application.Service{
			application.NewService(service),
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: true,
		},
		Linux: application.LinuxOptions{
			DisableQuitOnLastWindowClosed: true,
		},
		ShouldQuit: func() bool {
			if shouldQuit != nil {
				return shouldQuit()
			}
			return true
		},
		OnShutdown: onShutdown,
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: singleInstanceID,
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				if onSecondInstanceLaunch != nil {
					onSecondInstanceLaunch()
				}
			},
		},
	}
}

func newMainWindowOptions() application.WebviewWindowOptions {
	return application.WebviewWindowOptions{
		Title:      "OneTiny",
		Width:      960,
		Height:     680,
		MinWidth:   820,
		MinHeight:  560,
		StartState: application.WindowStateNormal,
		Linux: application.LinuxWindow{
			Icon: appIcon,
		},
	}
}
