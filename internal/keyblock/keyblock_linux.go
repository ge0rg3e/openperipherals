package keyblock

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// The Windows media watcher has no Linux equivalent. evsieve filters the
// selected keyboard's event streams, including its consumer controls.
type Watcher struct{}

func (*Watcher) Stop() {}

type Key uint16

type filterProcess struct {
	cmd  *exec.Cmd
	done chan struct{}
	err  error
}

var filter struct {
	sync.Mutex
	proc *filterProcess
	args []string
}

func Set([]Key) error {
	filter.Lock()
	defer filter.Unlock()
	stopFilter()
	filter.args = nil
	return nil
}

func Apply(vid, pid uint16, disabled map[string]uint16) error {
	binary, err := exec.LookPath("evsieve")
	if err != nil {
		return errors.New("install evsieve to block keys on Linux")
	}
	if unix.Access("/dev/uinput", unix.W_OK) != nil {
		return errors.New("key blocking requires write access to /dev/uinput")
	}
	paths, err := keyboardPaths(vid, pid)
	if err != nil {
		return err
	}

	args := []string{"--input"}
	args = append(args, paths...)
	args = append(args, "grab", "persist=reopen", "--block")
	var keys []string
	for id := range disabled {
		name := eventKey(id)
		if name == "" {
			return fmt.Errorf("key %q is unavailable for blocking on Linux", id)
		}
		keys = append(keys, "key:"+name)
	}
	sort.Strings(keys)
	args = append(args, keys...)
	args = append(args, "--output", "name=OpenPeripherals")

	filter.Lock()
	defer filter.Unlock()
	previous := filter.args
	stopFilter()
	if err := startFilter(binary, args); err != nil {
		filter.args = nil
		if previous != nil {
			if rollback := startFilter(binary, previous); rollback != nil {
				return errors.Join(err, rollback)
			}
			filter.args = previous
		}
		return err
	}
	filter.args = args
	return nil
}

func startFilter(binary string, args []string) error {
	cmd := exec.Command(binary, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	p := &filterProcess{cmd: cmd, done: make(chan struct{})}
	go func() { p.err = cmd.Wait(); close(p.done) }()
	select {
	case <-p.done:
		return fmt.Errorf("evsieve exited at startup (%v). Check keyboard event device and uinput permissions", p.err)
	case <-time.After(150 * time.Millisecond):
		filter.proc = p
		return nil
	}
}

func stopFilter() {
	if p := filter.proc; p != nil {
		_ = p.cmd.Process.Kill()
		<-p.done
		filter.proc = nil
	}
}

func keyboardPaths(vid, pid uint16) ([]string, error) {
	links, _ := filepath.Glob("/dev/input/by-id/*-event-*")
	seen := map[string]bool{}
	var paths []string
	var keyboard string
	for _, path := range links {
		node, err := filepath.EvalSymlinks(path)
		if err != nil || filepath.Dir(node) != "/dev/input" || !strings.HasPrefix(filepath.Base(node), "event") || seen[node] {
			continue
		}
		ids := filepath.Join("/sys/class/input", filepath.Base(node), "device", "id")
		vendor, _ := os.ReadFile(filepath.Join(ids, "vendor"))
		product, _ := os.ReadFile(filepath.Join(ids, "product"))
		if strings.TrimSpace(string(vendor)) != fmt.Sprintf("%04x", vid) || strings.TrimSpace(string(product)) != fmt.Sprintf("%04x", pid) {
			continue
		}
		phys, _ := os.ReadFile(filepath.Join(filepath.Dir(ids), "phys"))
		port := strings.TrimSpace(string(phys))
		if i := strings.LastIndex(port, "/input"); i >= 0 {
			port = port[:i]
		}
		if port == "" || (keyboard != "" && keyboard != port) {
			return nil, errors.New("could not identify a single physical keyboard for key blocking")
		}
		keyboard = port
		if unix.Access(node, unix.R_OK) != nil {
			return nil, fmt.Errorf("key blocking needs read access to %s", node)
		}
		seen[node] = true
		paths = append(paths, path)
	}
	if len(paths) == 0 {
		return nil, errors.New("no stable input device links found for this keyboard under /dev/input/by-id")
	}
	return paths, nil
}

var numbered = regexp.MustCompile("^((Key)([A-Z])|(Digit)([0-9])|(Numpad)([0-9])|(F)([1-9]|1[0-2]))$")

func eventKey(id string) string {
	if match := numbered.FindStringSubmatch(id); match != nil {
		switch {
		case match[3] != "":
			return strings.ToLower(match[3])
		case match[5] != "":
			return match[5]
		case match[7] != "":
			return "kp" + match[7]
		default:
			return "f" + match[9]
		}
	}
	return linuxKeys[id]
}

var linuxKeys = map[string]string{
	"Escape": "esc", "Backquote": "grave", "Minus": "minus", "Equal": "equal", "Backspace": "backspace", "Tab": "tab",
	"BracketLeft": "leftbrace", "BracketRight": "rightbrace", "Backslash": "backslash", "CapsLock": "capslock", "Semicolon": "semicolon", "Quote": "apostrophe",
	"Enter": "enter", "ShiftLeft": "leftshift", "ShiftRight": "rightshift", "Comma": "comma", "Period": "dot", "Slash": "slash",
	"ControlLeft": "leftctrl", "ControlRight": "rightctrl", "MetaLeft": "leftmeta", "MetaRight": "rightmeta", "AltLeft": "leftalt", "AltRight": "rightalt", "ContextMenu": "compose", "Space": "space",
	"PrintScreen": "sysrq", "ScrollLock": "scrolllock", "Pause": "pause",
	"Insert": "insert", "Home": "home", "PageUp": "pageup", "Delete": "delete", "End": "end", "PageDown": "pagedown",
	"ArrowLeft": "left", "ArrowUp": "up", "ArrowRight": "right", "ArrowDown": "down",
	"NumLock": "numlock", "NumpadDivide": "kpslash", "NumpadMultiply": "kpasterisk", "NumpadSubtract": "kpminus", "NumpadAdd": "kpplus", "NumpadDecimal": "kpdot", "NumpadEnter": "kpenter",
	"MediaTrackPrevious": "previoussong", "MediaPlayPause": "playpause", "MediaTrackNext": "nextsong", "MediaStop": "stopcd",
	"dial:volume_up": "volumeup", "dial:volume_down": "volumedown", "dial:mute": "mute",
}
