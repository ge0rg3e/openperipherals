package main

import (
	"log"
	"os"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

const windowsDesktop = false
const hiddenFlag = "--hidden"

func launchOptions(args []string) (exit, hidden bool) {
	if len(args) == 2 && args[0] == "--linux-fan-restore" {
		original, err := strconv.Atoi(args[1])
		if err != nil || original < 2 {
			log.Fatal("invalid automatic PWM mode")
		}
		if err := restoreFanOnExit(original); err != nil {
			log.Fatal(err)
		}
		return true, false
	}
	if len(args) == 4 && args[0] == "--linux-wallpaper-player" {
		parent := os.Getppid()
		if parent == 1 || unix.Prctl(unix.PR_SET_PDEATHSIG, uintptr(syscall.SIGKILL), 0, 0, 0) != nil || os.Getppid() != parent {
			log.Fatal("wallpaper host is unavailable")
		}
		window, err := strconv.ParseUint(args[1], 0, 32)
		if err != nil || window == 0 {
			log.Fatal("invalid wallpaper window ID")
		}
		player := args[2]
		if err := syscall.Exec(player, []string{player, "--wid=" + strconv.FormatUint(window, 10), "--no-config", "--no-audio", "--loop-file=inf", "--panscan=1", "--", args[3]}, os.Environ()); err != nil {
			log.Fatal(err)
		}
	}
	return false, false
}

func postShutdown() {}
