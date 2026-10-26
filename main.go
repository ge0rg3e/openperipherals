package main

import (
	"embed"
	"log"
	"os"
	"slices"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var trayIcon []byte

// hiddenFlag starts the app in the tray (used by start with Windows).
const hiddenFlag = "--hidden"

func init() {
	application.RegisterEvent[Status]("status")
	application.RegisterEvent[FanStatus]("fans")
}

func main() {
	fans := &FanService{}
	devices := &PeripheralService{}
	app := application.New(application.Options{
		Name:        "OpenPeripherals",
		Description: "Lightweight RGB and DPI control for Razer, Corsair, Logitech, Redragon and Pulsar peripherals",
		Icon:        trayIcon,
		Services: []application.Service{
			application.NewService(devices),
			application.NewService(fans),
			application.NewService(&SystemService{}),
		},
		PostShutdown: func() {
			if relaunchElevated {
				relaunchAsAdmin()
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

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     "OpenPeripherals",
		Width:     1200,
		Height:    900,
		MinWidth:  760,
		MinHeight: 560,
		Hidden:    slices.Contains(os.Args[1:], hiddenFlag),
		// The frontend draws its own title bar.
		Frameless:        true,
		BackgroundColour: application.NewRGB(11, 12, 13),
		URL:              "/",
	})

	// Closing the window hides it to the tray instead of quitting.
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	menu := application.NewMenu()
	menu.Add("Open").OnClick(func(*application.Context) { window.Show().Focus() })
	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(*application.Context) { app.Quit() })

	tray := app.SystemTray.New()
	tray.SetIcon(trayIcon)
	tray.SetTooltip("OpenPeripherals")
	tray.SetMenu(menu)
	tray.OnClick(func() { window.Show().Focus() })
	stopTray := make(chan struct{})
	defer close(stopTray)
	go runTrayTooltip(tray, fans, devices, stopTray)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
