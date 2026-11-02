package main

import (
	"embed"
	"encoding/json"
	"log"
	"openperipherals/internal/wallpaper"
	"os"
	"slices"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var trayIcon []byte

func init() {
	application.RegisterEvent[Status]("status")
	application.RegisterEvent[FanStatus]("fans")
	application.RegisterEvent[Status]("tray-settings")
	application.RegisterEvent[WallpaperStatus]("wallpaper")
}

func main() {
	args := os.Args[1:]
	if slices.Contains(args, "--wallpaper-host") {
		if err := wallpaper.RunHost(os.Stdin, os.Stdout); err != nil {
			_ = json.NewEncoder(os.Stdout).Encode(wallpaper.State{Phase: "error", Reason: err.Error()})
			os.Exit(1)
		}
		return
	}
	// Start through the elevated startup task when it exists. This keeps a
	// manual launch free of UAC prompts too.
	if prepareAutostart(args) {
		return
	}
	viaTask := slices.Contains(args, viaTaskFlag)
	hidden := slices.Contains(args, hiddenFlag)
	if consumeShowRequest(viaTask) {
		hidden = false
		// Tell an already-running elevated copy to show its window too.
		os.Args = append([]string{os.Args[0]}, withoutFlag(args, hiddenFlag)...)
	}

	fans := &FanService{}
	devices := &PeripheralService{}
	wallpapers := &WallpaperService{}
	app := application.New(application.Options{
		Name:        "OpenPeripherals",
		Description: "Lightweight RGB and DPI control for Razer, Corsair, Logitech, Redragon and Pulsar peripherals",
		Icon:        trayIcon,
		Services: []application.Service{
			application.NewService(devices),
			application.NewService(fans),
			application.NewService(&SystemService{}),
			application.NewService(wallpapers),
		},
		PostShutdown: func() {
			if relaunchElevated {
				relaunchAsAdmin(relaunchArgs...)
			}
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		// Key blocking must keep running after the window is closed.
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: true,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.openperipherals.app",
			OnSecondInstanceLaunch: func(d application.SecondInstanceData) {
				if slices.Contains(d.Args, hiddenFlag) {
					return
				}
				if w, ok := application.Get().Window.Get("main"); ok {
					w.Show().Focus()
				}
			},
		},
	})
	updatesEnabled := setupUpdater(app)

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     "OpenPeripherals",
		Width:     1200,
		Height:    900,
		MinWidth:  760,
		MinHeight: 560,
		Hidden:    hidden,
		// The frontend draws its own title bar.
		Frameless:        true,
		BackgroundColour: application.NewRGB(11, 12, 13),
		URL:              "/",
	})

	// Closing the window hides it to the tray instead of quitting.
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		go func() {
			if status := wallpapers.Status(); status.Config.Enabled {
				if _, err := wallpapers.Stop(); err != nil {
					log.Printf("stop wallpaper on window close: %v", err)
				}
			}
		}()
		e.Cancel()
	})

	tray := app.SystemTray.New()
	tray.SetIcon(trayIcon)
	tray.SetTooltip("OpenPeripherals")
	setupTrayMenu(app, tray, fans, devices, wallpapers, updatesEnabled, func() { window.Show().Focus() })
	stopTray := make(chan struct{})
	defer close(stopTray)
	go runTrayTooltip(tray, fans, devices, stopTray)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
