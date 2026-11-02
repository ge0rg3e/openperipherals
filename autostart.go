package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	runKey  = `Software\Microsoft\Windows\CurrentVersion\Run`
	runName = "OpenPeripherals"

	// Flags used by the scheduled task and the one-time administrator setup.
	hiddenFlag         = "--hidden"
	viaTaskFlag        = "--via-task"
	setupAutostartFlag = "--setup-autostart"
)

// showRequestMaxAge keeps a stale "show the window" marker from opening the
// app on a later login.
const showRequestMaxAge = 30 * time.Second

// Autostart reports whether the app starts with Windows, either through the
// elevated task or through the legacy Run-key entry.
func (s *PeripheralService) Autostart() bool {
	return autostartTaskExists() || runKeyExists()
}

// AutostartElevated reports whether the Windows startup entry runs the app
// with administrator rights.
func (s *PeripheralService) AutostartElevated() bool {
	return autostartTaskExists()
}

// SetAutostart starts the app hidden in the tray at login, or stops doing so.
// Startup uses a highest-privilege scheduled task, so fan control works at
// login without a UAC prompt. Enabling it from an unelevated copy restarts the
// app elevated once, because Windows only lets administrators create such a
// task.
func (s *PeripheralService) SetAutostart(enabled bool) error {
	if !enabled {
		if autostartTaskExists() {
			if err := schtasks("/Delete", "/TN", runName, "/F"); err != nil {
				return fmt.Errorf("remove startup task (needs administrator): %w", err)
			}
		}
		removeRunValue()
		return nil
	}
	if !elevated() {
		// Relaunch elevated after this instance has released the single-instance
		// lock. The elevated copy installs the task and keeps running. Any
		// existing Run-key entry stays until then, so cancelling the UAC prompt
		// leaves startup as it was.
		relaunchElevated = true
		relaunchArgs = []string{setupAutostartFlag}
		application.Get().Quit()
		return nil
	}
	if err := installAutostartTask(); err != nil {
		return err
	}
	removeRunValue()
	return nil
}

// prepareAutostart handles startup modes before the UI is created:
//   - a one-time elevated setup installs the task,
//   - an elevated copy migrates the old unelevated Run-key entry,
//   - an unelevated copy starts the existing task, so a manual launch is
//     elevated without a prompt too.
//
// It returns true when another copy is starting and this process should exit.
func prepareAutostart(args []string) bool {
	if slices.Contains(args, setupAutostartFlag) {
		if err := installAutostartTask(); err != nil {
			log.Printf("enable administrator startup: %v", err)
			messageBox("OpenPeripherals", "Could not enable administrator startup:\n"+err.Error())
		} else {
			removeRunValue()
		}
		return false
	}

	if elevated() && runKeyExists() {
		// The user enabled startup in an older version, when it ran
		// unelevated. Move it to the highest-privilege task.
		if err := installAutostartTask(); err == nil {
			removeRunValue()
		} else {
			log.Printf("migrate startup task: %v", err)
		}
	}

	viaTask := slices.Contains(args, viaTaskFlag)
	if elevated() || viaTask || !autostartTaskExists() {
		return false
	}
	if !autostartTaskTargetsCurrentExecutable() {
		log.Printf("startup task targets a different executable; launching current copy normally")
		return false
	}
	return relaunchThroughTask()
}

// consumeShowRequest returns true when this launch came from a manual start
// that asked the elevated task to show its window. The marker is single-use.
func consumeShowRequest(viaTask bool) bool {
	if !viaTask {
		return false
	}
	p := showRequestPath()
	if p == "" {
		return false
	}
	st, err := os.Stat(p)
	if err != nil {
		return false
	}
	os.Remove(p)
	return time.Since(st.ModTime()) < showRequestMaxAge
}

func installAutostartTask() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// --via-task stops a copy that still is not elevated from relaunching
	// through the task again.
	cmd := fmt.Sprintf(`"%s" %s %s`, exe, hiddenFlag, viaTaskFlag)
	return schtasks("/Create", "/TN", runName, "/TR", cmd, "/SC", "ONLOGON", "/RL", "HIGHEST", "/F")
}

func autostartTaskExists() bool {
	return schtasks("/Query", "/TN", runName) == nil
}

func autostartTaskTargetsCurrentExecutable() bool {
	executable, err := os.Executable()
	if err != nil {
		return false
	}
	cmd := exec.Command("schtasks.exe", "/Query", "/TN", runName, "/XML")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	var task struct {
		Actions struct {
			Exec struct {
				Command string `xml:"Command"`
			} `xml:"Exec"`
		} `xml:"Actions"`
	}
	if err := xml.Unmarshal(output, &task); err != nil {
		return false
	}
	command := strings.Trim(strings.TrimSpace(task.Actions.Exec.Command), `"`)
	return strings.EqualFold(filepath.Clean(command), filepath.Clean(executable))
}

func relaunchThroughTask() bool {
	if err := requestShowOnNextLaunch(); err != nil {
		return false
	}
	if err := schtasks("/Run", "/TN", runName); err != nil {
		clearShowRequest()
		return false
	}
	return true
}

func showRequestPath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "openperipherals", "show")
}

func requestShowOnNextLaunch() error {
	p := showRequestPath()
	if p == "" {
		return errors.New("no cache directory")
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(time.Now().Format(time.RFC3339Nano)), 0o644)
}

func clearShowRequest() {
	if p := showRequestPath(); p != "" {
		os.Remove(p)
	}
}

func runKeyExists() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(runName)
	return err == nil
}

func removeRunValue() {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()
	if err := k.DeleteValue(runName); err != nil && err != registry.ErrNotExist {
		log.Printf("remove startup registry value: %v", err)
	}
}

func schtasks(args ...string) error {
	c := exec.Command("schtasks.exe", args...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	return c.Run()
}

func messageBox(title, text string) {
	t, err := windows.UTF16PtrFromString(text)
	if err != nil {
		return
	}
	c, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	windows.MessageBox(0, t, c, windows.MB_OK|windows.MB_ICONWARNING)
}

func withoutFlag(items []string, flag string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item != flag {
			out = append(out, item)
		}
	}
	return out
}
