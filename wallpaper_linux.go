package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/wailsapp/wails/v3/pkg/application"
	"openperipherals/internal/wallpaper"
)

type WallpaperService struct {
	mu      sync.Mutex
	status  WallpaperStatus
	path    string
	process *exec.Cmd
}

func (s *WallpaperService) ServiceStartup(context.Context, application.ServiceOptions) error {
	s.status = WallpaperStatus{Config: wallpaper.DefaultConfig(), Phase: "stopped"}
	s.status.Config.Kind = "video"
	if dir, err := os.UserConfigDir(); err == nil {
		s.path = filepath.Join(dir, "openperipherals", "wallpaper-linux.json")
		if b, err := os.ReadFile(s.path); err == nil {
			var cfg wallpaper.Config
			if json.Unmarshal(b, &cfg) == nil && cfg.Kind == "video" {
				s.status.Config = cfg
			}
		}
	}
	if s.status.Config.Enabled {
		cfg := s.status.Config
		s.status.Config.Enabled = false
		if _, err := s.Apply(cfg); err != nil {
			s.status.Phase, s.status.Reason = "error", err.Error()
		}
	}
	return nil
}

func (s *WallpaperService) ServiceShutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopProcess()
	return nil
}

func (s *WallpaperService) Status() WallpaperStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *WallpaperService) Apply(cfg wallpaper.Config) (WallpaperStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cfg.Kind != "video" {
		return s.status, errors.New("Linux wallpaper supports local MP4 video")
	}
	cfg, err := wallpaper.Validate(cfg)
	if err != nil {
		return s.status, err
	}
	cmd, err := wallpaperCommand(cfg.Source)
	if err != nil {
		return s.status, err
	}
	s.stopProcess()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		s.status.Config.Enabled = false
		s.status.Phase, s.status.Reason = "error", err.Error()
		s.publish()
		return s.status, err
	}
	s.process = cmd
	cfg.Enabled = true
	s.status = WallpaperStatus{Config: cfg, Phase: "playing"}
	go s.wait(cmd)
	s.publish()
	return s.status, s.save()
}

func wallpaperCommand(path string) (*exec.Cmd, error) {
	if os.Getenv("XDG_SESSION_TYPE") == "wayland" || os.Getenv("WAYLAND_DISPLAY") != "" {
		binary, err := exec.LookPath("mpvpaper")
		if err != nil {
			return nil, errors.New("install mpvpaper for video wallpaper on a supported Wayland compositor")
		}
		return exec.Command(binary, "-o", "no-config no-audio loop-file=inf panscan=1", "ALL", path), nil
	}
	if os.Getenv("DISPLAY") == "" {
		return nil, errors.New("video wallpaper requires an X11 or supported Wayland session")
	}
	wrapper, err := exec.LookPath("xwinwrap")
	if err != nil {
		return nil, errors.New("install xwinwrap and mpv for X11 video wallpaper")
	}
	player, err := exec.LookPath("mpv")
	if err != nil {
		return nil, errors.New("install mpv for X11 video wallpaper")
	}
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return exec.Command(wrapper, "-ni", "-b", "-fs", "-st", "-sp", "-nf", "-fdt", "--", exe, "--linux-wallpaper-player", "WID", player, path), nil
}

func (s *WallpaperService) wait(cmd *exec.Cmd) {
	err := cmd.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.process != cmd {
		return
	}
	s.process = nil
	s.status.Config.Enabled = false
	s.status.ManualPause = false
	s.status.Phase, s.status.Reason = "error", "Wallpaper player exited"
	if err != nil {
		s.status.Reason = fmt.Sprintf("Wallpaper player exited: %v", err)
	}
	s.publish()
}

func (s *WallpaperService) stopProcess() {
	if s.process == nil {
		return
	}
	_ = syscall.Kill(-s.process.Process.Pid, syscall.SIGKILL)
	s.process = nil
}

func (s *WallpaperService) Stop() (WallpaperStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopProcess()
	s.status.Config.Enabled = false
	s.status.Phase, s.status.Reason, s.status.ManualPause = "stopped", "", false
	s.publish()
	return s.status, s.save()
}

func (s *WallpaperService) SetPaused(paused bool) (WallpaperStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.process == nil {
		return s.status, errors.New("wallpaper is not playing")
	}
	signal := syscall.SIGCONT
	phase := "playing"
	if paused {
		signal, phase = syscall.SIGSTOP, "paused"
	}
	if err := syscall.Kill(-s.process.Process.Pid, signal); err != nil {
		return s.status, err
	}
	s.status.ManualPause, s.status.Phase = paused, phase
	s.publish()
	return s.status, nil
}

func (s *WallpaperService) publish() {
	if app := application.Get(); app != nil {
		app.Event.Emit("wallpaper", s.status)
	}
}

func (s *WallpaperService) save() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	b, err := json.Marshal(s.status.Config)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0600)
}

func (*WallpaperService) PickHTML() (string, error) {
	return "", errors.New("Linux wallpaper supports local MP4 video")
}
func (*WallpaperService) PickMP4() (string, error) {
	return application.Get().Dialog.OpenFile().SetTitle("Choose MP4 wallpaper").AddFilter("MP4 video", "*.mp4").PromptForSingleSelection()
}
