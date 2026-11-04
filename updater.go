package main

import (
	"context"
	_ "embed"
	"log"
	"runtime"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
)

// currentVersion is set from the release tag by the packaging workflow.
var currentVersion = "0.0.2"

//go:embed updater.key.pub
var updaterPublicKey []byte

func setupUpdater(app *application.App) bool {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		return false
	}

	provider, err := endpoint.New(endpoint.Config{
		URL: "https://github.com/ge0rg3e/openperipherals/releases/latest/download/manifest.json",
	})
	if err != nil {
		log.Printf("update provider: %v", err)
		return false
	}
	if err := app.Updater.Init(updater.Config{
		CurrentVersion: currentVersion,
		Providers:      []updater.Provider{provider},
		PublicKey:      updaterPublicKey,
		Window:         &updater.BuiltinWindow{Options: updater.WindowOptions{Title: "OpenPeripherals update"}},
	}); err != nil {
		log.Printf("update initialization: %v", err)
		return false
	}

	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		go checkForUpdate(app)
	})
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			checkForUpdate(app)
		}
	}()
	return true
}

func addUpdaterMenu(menu *application.Menu, app *application.App) {
	menu.Add("Check for updates").OnClick(func(*application.Context) {
		go func() {
			if err := app.Updater.CheckAndInstall(context.Background()); err != nil {
				log.Printf("update check: %v", err)
			}
		}()
	})
}

func checkForUpdate(app *application.App) {
	ctx := context.Background()
	release, err := app.Updater.Check(ctx)
	if err != nil {
		log.Printf("automatic update check: %v", err)
		return
	}
	if release == nil {
		return
	}
	if err := app.Updater.CheckAndInstall(ctx); err != nil {
		log.Printf("automatic update install: %v", err)
	}
}
