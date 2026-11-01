package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/w32"
	"golang.org/x/sys/windows"

	"openperipherals/internal/wallpaper"
)

func TestMain(m *testing.M) {
	// Exercise the real subprocess/job-object path without starting peripheral
	// services or touching the user's saved wallpaper configuration.
	if slices.Contains(os.Args, "--wallpaper-host") {
		if err := wallpaper.RunHost(os.Stdin, os.Stdout); err != nil {
			_ = json.NewEncoder(os.Stdout).Encode(wallpaper.State{Phase: "error", Reason: err.Error()})
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestWallpaperServiceLifecycle(t *testing.T) {
	if os.Getenv("OPENPERIPHERALS_WALLPAPER_INTEGRATION") != "1" {
		t.Skip("opt-in native desktop integration test")
	}
	config := wallpaper.DefaultConfig()
	t.Run("builtin", func(t *testing.T) { testWallpaperLifecycle(t, config) })
	if video := os.Getenv("OPENPERIPHERALS_WALLPAPER_TEST_MP4"); video != "" {
		config.Kind, config.Source = "video", video
		t.Run("mp4", func(t *testing.T) { testWallpaperLifecycle(t, config) })
	}
	t.Run("host crash", func(t *testing.T) {
		original := configuredWindowsWallpaper(t)
		s := &WallpaperService{cfg: wallpaper.DefaultConfig(), path: filepath.Join(t.TempDir(), "wallpaper.json")}
		t.Cleanup(func() { _ = s.ServiceShutdown() })
		cfg := wallpaper.DefaultConfig()
		cfg.PauseCovered, cfg.PauseBattery = false, false
		if _, err := s.Apply(cfg); err != nil {
			t.Fatal(err)
		}
		waitWallpaperPhase(t, s, "playing")
		handles := wallpaperHostWindows(t, s)
		s.mu.Lock()
		process := s.host
		s.mu.Unlock()
		if err := process.cmd.Process.Kill(); err != nil {
			t.Fatal(err)
		}
		select {
		case <-process.done:
		case <-time.After(10 * time.Second):
			t.Fatal("host crash cleanup timed out")
		}
		assertWallpaperReleased(t, handles, original)
	})
	t.Run("invalid mp4", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "broken.mp4")
		if err := os.WriteFile(path, []byte("not a video"), 0600); err != nil {
			t.Fatal(err)
		}
		s := &WallpaperService{cfg: wallpaper.DefaultConfig(), path: filepath.Join(t.TempDir(), "wallpaper.json")}
		t.Cleanup(func() { _ = s.ServiceShutdown() })
		config.Kind, config.Source = "video", path
		if _, err := s.Apply(config); err != nil {
			t.Fatal(err)
		}
		waitWallpaperPhase(t, s, "error")
		if !strings.Contains(s.Status().Reason, "MP4 could not play") {
			t.Fatalf("missing playback error: %+v", s.Status())
		}
	})
}

func testWallpaperLifecycle(t *testing.T, config wallpaper.Config) {
	t.Helper()
	original := configuredWindowsWallpaper(t)
	s := &WallpaperService{cfg: wallpaper.DefaultConfig(), path: filepath.Join(t.TempDir(), "wallpaper.json")}
	t.Cleanup(func() { _ = s.ServiceShutdown() })
	config.PauseCovered = false
	config.PauseBattery = false
	if _, err := s.Apply(config); err != nil {
		t.Fatal(err)
	}
	waitWallpaperPhase(t, s, "playing")
	if _, err := s.SetPaused(true); err != nil {
		t.Fatal(err)
	}
	waitWallpaperPhase(t, s, "paused")
	if config.Kind == "video" && strings.Contains(s.Status().Reason, "unloaded") {
		t.Fatal("video failed to suspend without unloading")
	}
	if !s.Status().Snapshot {
		t.Fatal("pause did not retain a wallpaper snapshot")
	}
	if _, err := s.SetPaused(false); err != nil {
		t.Fatal(err)
	}
	waitWallpaperPhase(t, s, "playing")
	// Disable from the paused state too: the retained snapshot must disappear.
	if _, err := s.SetPaused(true); err != nil {
		t.Fatal(err)
	}
	waitWallpaperPhase(t, s, "paused")
	windows := wallpaperHostWindows(t, s)
	if _, err := s.Stop(); err != nil {
		t.Fatal(err)
	}
	assertWallpaperReleased(t, windows, original)
	if status := s.Status(); status.Config.Enabled || status.Phase != "stopped" {
		t.Fatalf("after stop: %+v", status)
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatal(err)
	}
	var saved wallpaper.Config
	if err = json.Unmarshal(data, &saved); err != nil || saved.Enabled {
		t.Fatalf("stop did not persist disabled state: %s (%v)", data, err)
	}
	// Shutdown must clean up processes without discarding the next-start choice.
	if _, err = s.Apply(config); err != nil {
		t.Fatal(err)
	}
	waitWallpaperPhase(t, s, "playing")
	windows = wallpaperHostWindows(t, s)
	if err = s.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	host := s.host
	s.mu.Unlock()
	if host != nil || !s.Status().Config.Enabled {
		t.Fatal("shutdown did not release host and retain saved choice")
	}
	assertWallpaperReleased(t, windows, original)
}

func configuredWindowsWallpaper(t *testing.T) string {
	t.Helper()
	var path [32768]uint16
	ok, _, err := windows.NewLazySystemDLL("user32.dll").NewProc("SystemParametersInfoW").Call(0x0073, uintptr(len(path)), uintptr(unsafe.Pointer(&path[0])), 0) // SPI_GETDESKWALLPAPER
	if ok == 0 {
		t.Fatalf("read Windows wallpaper: %v", err)
	}
	return windows.UTF16ToString(path[:])
}

func wallpaperHostWindows(t *testing.T, s *WallpaperService) []w32.HWND {
	t.Helper()
	s.mu.Lock()
	pid := s.host.cmd.Process.Pid
	s.mu.Unlock()
	var handles []w32.HWND
	childCallback := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		window := w32.HWND(hwnd)
		_, owner := w32.GetWindowThreadProcessId(window)
		if owner == pid && w32.GetClassName(window) == "OpenPeripheralsWallpaperHost" {
			handles = append(handles, window)
		}
		return 1
	})
	enumChildren := windows.NewLazySystemDLL("user32.dll").NewProc("EnumChildWindows")
	callback := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		enumChildren.Call(hwnd, childCallback, 0)
		return 1
	})
	w32.EnumWindows(callback, 0)
	if len(handles) < 2 {
		t.Fatalf("expected wallpaper surface and snapshot window, found %d", len(handles))
	}
	return handles
}

func assertWallpaperReleased(t *testing.T, handles []w32.HWND, original string) {
	t.Helper()
	for _, hwnd := range handles {
		if w32.IsWindow(hwnd) {
			t.Errorf("wallpaper window %v remained after stopping", hwnd)
		}
	}
	if got := configuredWindowsWallpaper(t); got != original {
		t.Errorf("Windows wallpaper changed: got %q, want %q", got, original)
	}
}

func waitWallpaperPhase(t *testing.T, s *WallpaperService, phase string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		status := s.Status()
		if status.Phase == phase {
			return
		}
		if status.Phase == "error" {
			t.Fatalf("wallpaper failed: %s", status.Reason)
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("wallpaper did not reach %s: %+v", phase, s.Status())
}
