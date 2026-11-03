package main

import (
	"encoding/json"
	"openperipherals/internal/wallpaper"
	"os"
	"slices"
)

const windowsDesktop = true

func launchOptions(args []string) (exit, hidden bool) {
	if slices.Contains(args, "--wallpaper-host") {
		if err := wallpaper.RunHost(os.Stdin, os.Stdout); err != nil {
			_ = json.NewEncoder(os.Stdout).Encode(wallpaper.State{Phase: "error", Reason: err.Error()})
			os.Exit(1)
		}
		return true, false
	}
	if prepareAutostart(args) {
		return true, false
	}
	hidden = slices.Contains(args, hiddenFlag)
	if consumeShowRequest(slices.Contains(args, viaTaskFlag)) {
		hidden = false
		os.Args = append([]string{os.Args[0]}, withoutFlag(args, hiddenFlag)...)
	}
	return false, hidden
}

func postShutdown() {
	if relaunchElevated {
		relaunchAsAdmin(relaunchArgs...)
	}
}
