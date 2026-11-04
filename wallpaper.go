//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/sys/windows"
	"openperipherals/internal/appdata"
	"openperipherals/internal/wallpaper"
)

type wallpaperProcess struct {
	cmd     *exec.Cmd
	in      io.WriteCloser
	encoder *json.Encoder
	write   sync.Mutex
	done    chan struct{}
	job     windows.Handle
}

type WallpaperService struct {
	op     sync.Mutex
	mu     sync.Mutex
	cfg    wallpaper.Config
	state  wallpaper.State
	manual bool
	path   string
	host   *wallpaperProcess
}

func (s *WallpaperService) ServiceStartup(context.Context, application.ServiceOptions) error {
	s.cfg = wallpaper.DefaultConfig()
	s.state = wallpaper.State{Phase: "stopped"}
	if dir, err := appdata.Dir(); err == nil {
		s.path = filepath.Join(dir, "wallpaper.json")
		if data, err := os.ReadFile(s.path); err == nil {
			if err = json.Unmarshal(data, &s.cfg); err != nil {
				s.cfg = wallpaper.DefaultConfig()
				s.state = wallpaper.State{Phase: "error", Reason: "Saved wallpaper settings could not be read"}
			}
		}
	}
	if s.cfg.Enabled {
		if _, err := s.Apply(s.cfg); err != nil {
			s.mu.Lock()
			s.state = wallpaper.State{Phase: "error", Reason: err.Error()}
			s.mu.Unlock()
		}
	}
	return nil
}

func (s *WallpaperService) ServiceShutdown() error {
	s.op.Lock()
	defer s.op.Unlock()
	s.stopHost() // Keep Enabled for restoration on the next app start.
	return nil
}

func (s *WallpaperService) Status() WallpaperStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return WallpaperStatus{Config: s.cfg, Phase: s.state.Phase, Reason: s.state.Reason, ManualPause: s.manual, Snapshot: s.state.Snapshot}
}

func (s *WallpaperService) publish() {
	if app := application.Get(); app != nil {
		app.Event.Emit("wallpaper", s.Status())
	}
}

func (s *WallpaperService) save() error {
	if s.path == "" {
		return fmt.Errorf("wallpaper settings directory is unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	s.mu.Lock()
	data, err := json.MarshalIndent(s.cfg, "", "  ")
	s.mu.Unlock()
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err = os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Apply validates and starts the chosen wallpaper; all work stays in a separate
// mode of this executable. No second executable is installed or required.
func (s *WallpaperService) Apply(cfg wallpaper.Config) (WallpaperStatus, error) {
	s.op.Lock()
	defer s.op.Unlock()
	cfg, err := wallpaper.Validate(cfg)
	if err != nil {
		return s.Status(), err
	}
	cfg.Enabled = true
	exe, err := os.Executable()
	if err != nil {
		return s.Status(), err
	}
	s.stopHost()
	cmd := exec.Command(exe, "--wallpaper-host")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	in, err := cmd.StdinPipe()
	if err != nil {
		return s.Status(), err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		in.Close()
		return s.Status(), err
	}
	p := &wallpaperProcess{cmd: cmd, in: in, encoder: json.NewEncoder(in), done: make(chan struct{})}
	// Closing this job kills the host and all WebView2 descendants, including
	// after an app crash. The host waits on stdin until assignment is complete.
	p.job, err = windows.CreateJobObject(nil, nil)
	if err != nil {
		in.Close()
		out.Close()
		return s.Status(), err
	}
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	_, err = windows.SetInformationJobObject(p.job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits)))
	if err != nil {
		windows.CloseHandle(p.job)
		in.Close()
		out.Close()
		return s.Status(), err
	}
	if err = cmd.Start(); err != nil {
		windows.CloseHandle(p.job)
		in.Close()
		out.Close()
		return s.Status(), err
	}
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err == nil {
		err = windows.AssignProcessToJobObject(p.job, process)
		windows.CloseHandle(process)
	}
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		windows.CloseHandle(p.job)
		in.Close()
		out.Close()
		return s.Status(), fmt.Errorf("isolate wallpaper process: %w", err)
	}
	s.mu.Lock()
	s.host = p
	s.cfg = cfg
	s.manual = false
	s.state = wallpaper.State{Phase: "loading", Reason: "Starting wallpaper"}
	s.mu.Unlock()
	go s.watchHost(p, out)
	if err = p.send(wallpaper.Command{Action: "start", Config: cfg}); err != nil {
		s.stopHost()
		return s.Status(), err
	}
	if err = s.save(); err != nil {
		s.stopHost()
		s.mu.Lock()
		s.cfg.Enabled = false
		s.state = wallpaper.State{Phase: "error", Reason: err.Error()}
		s.mu.Unlock()
		s.publish()
		return s.Status(), err
	}
	s.publish()
	return s.Status(), nil
}

func (p *wallpaperProcess) send(c wallpaper.Command) error {
	p.write.Lock()
	defer p.write.Unlock()
	return p.encoder.Encode(c)
}

func (s *WallpaperService) watchHost(p *wallpaperProcess, out io.ReadCloser) {
	decoder := json.NewDecoder(out)
	for {
		var state wallpaper.State
		if decoder.Decode(&state) != nil {
			break
		}
		s.mu.Lock()
		current := s.host == p
		if current {
			s.state = state
		}
		s.mu.Unlock()
		if current {
			s.publish()
		}
	}
	out.Close()
	err := p.cmd.Wait()
	s.mu.Lock()
	current := s.host == p
	if current {
		s.host = nil
		if s.state.Phase != "error" {
			s.state = wallpaper.State{Phase: "error", Reason: "Wallpaper host exited; apply the wallpaper again"}
			if err != nil {
				s.state.Reason = "Wallpaper host exited: " + err.Error()
			}
		}
	}
	s.mu.Unlock()
	p.write.Lock()
	p.in.Close()
	p.write.Unlock()
	windows.CloseHandle(p.job)
	// Child windows belong to Explorer's desktop layer. Refresh only once the
	// host and WebView processes are gone, also covering crashes/forced stops.
	wallpaper.RefreshDesktop()
	close(p.done)
	if current {
		s.publish()
	}
}

func (s *WallpaperService) stopHost() {
	s.mu.Lock()
	p := s.host
	s.host = nil
	s.mu.Unlock()
	if p == nil {
		return
	}
	_ = p.send(wallpaper.Command{Action: "stop"})
	p.write.Lock()
	p.in.Close()
	p.write.Unlock()
	select {
	case <-p.done:
	case <-time.After(5 * time.Second):
		p.cmd.Process.Kill()
		<-p.done
	}
}

func (s *WallpaperService) Stop() (WallpaperStatus, error) {
	s.op.Lock()
	defer s.op.Unlock()
	s.stopHost()
	s.mu.Lock()
	s.cfg.Enabled = false
	s.manual = false
	s.state = wallpaper.State{Phase: "stopped"}
	s.mu.Unlock()
	err := s.save()
	s.publish()
	return s.Status(), err
}

func (s *WallpaperService) SetPaused(paused bool) (WallpaperStatus, error) {
	s.op.Lock()
	defer s.op.Unlock()
	s.mu.Lock()
	p := s.host
	s.mu.Unlock()
	if p == nil {
		return s.Status(), fmt.Errorf("start a wallpaper first")
	}
	action := "resume"
	if paused {
		action = "pause"
	}
	if err := p.send(wallpaper.Command{Action: action}); err != nil {
		return s.Status(), err
	}
	s.mu.Lock()
	s.manual = paused
	s.mu.Unlock()
	s.publish()
	return s.Status(), nil
}

func (s *WallpaperService) PickHTML() (string, error) {
	return application.Get().Dialog.OpenFile().SetTitle("Choose HTML wallpaper").AddFilter("HTML wallpaper", "*.html;*.htm").PromptForSingleSelection()
}

func (s *WallpaperService) PickMP4() (string, error) {
	return application.Get().Dialog.OpenFile().SetTitle("Choose MP4 wallpaper").AddFilter("MP4 video", "*.mp4").PromptForSingleSelection()
}
