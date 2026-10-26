package main

import (
	"fmt"
	"maps"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows/registry"

	"openperipherals/internal/keyblock"
)

// Disabled keys are stored by id: the KeyboardEvent.code of the key (e.g.
// "KeyQ", "MediaPlayPause") or "dial:<action>" for the dial, mapped to the
// virtual key the hook blocks. Media keys and the dial can be blocked on the
// connected keyboard only; other keys can't be traced to a device and are blocked on
// every keyboard.

// legacyKeys converts ids saved by earlier versions.
var legacyKeys = map[string]struct {
	id  string
	key uint16
}{
	"volume_up":   {"dial:volume_up", 0xaf},
	"volume_down": {"dial:volume_down", 0xae},
	"mute":        {"dial:mute", 0xad},
	"prev":        {"MediaTrackPrevious", 0xb1},
	"play":        {"MediaPlayPause", 0xb3},
	"next":        {"MediaTrackNext", 0xb0},
	"stop":        {"MediaStop", 0xb2},
}

const (
	runKey  = `Software\Microsoft\Windows\CurrentVersion\Run`
	runName = "OpenPeripherals"
)

// migrateKeys moves keys saved in the old list format into DisabledKeys.
func (s *PeripheralService) migrateKeys() {
	if len(s.cfg.BlockedKeys) == 0 {
		return
	}
	if s.cfg.DisabledKeys == nil {
		s.cfg.DisabledKeys = map[string]uint16{}
	}
	for _, old := range s.cfg.BlockedKeys {
		if k, ok := legacyKeys[old]; ok {
			s.cfg.DisabledKeys[k.id] = k.key
		}
	}
	s.cfg.BlockedKeys = nil
	s.save()
}

// DisabledKeys returns the disabled keys by id.
func (s *PeripheralService) DisabledKeys() map[string]uint16 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return maps.Clone(s.cfg.DisabledKeys)
}

// SetKeyDisabled disables or re-enables one key. `key` is the virtual-key
// code, optionally with keyblock.ExtendedOnly / PlainOnly. Blocking stays
// active while the app runs, including when the window is closed to the tray.
func (s *PeripheralService) SetKeyDisabled(id string, key uint16, disabled bool) error {
	if id == "" || len(id) > 64 || key&0xff == 0 {
		return fmt.Errorf("invalid key %q", id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := maps.Clone(s.cfg.DisabledKeys)
	if next == nil {
		next = map[string]uint16{}
	}
	if disabled {
		next[id] = key
	} else {
		delete(next, id)
	}
	if err := s.applyBlocking(next); err != nil {
		return err
	}
	s.cfg.DisabledKeys = next
	s.save()
	return nil
}

// applyBlocking installs the hook for the given keys and, when media keys
// are among them, runs the watcher that attributes their presses to the
// keyboard so the same keys on other devices keep working.
func (s *PeripheralService) applyBlocking(disabled map[string]uint16) error {
	keys := make([]keyblock.Key, 0, len(disabled))
	s.wantWatch = false
	for _, k := range disabled {
		keys = append(keys, keyblock.Key(k))
		if keyblock.Attributable(uint8(k)) {
			s.wantWatch = true
		}
	}
	keyblock.SetDeviceOnly(true)
	s.followKeyboard()
	return keyblock.Set(keys)
}

// followKeyboard points the media key watcher at the connected keyboard.
// While no keyboard is connected it keeps watching the last one, which it
// picks up again when replugged. Caller holds s.mu, or is starting up.
func (s *PeripheralService) followKeyboard() {
	target := s.watchFor
	if s.kb != nil {
		vid, pid := s.kb.ids()
		target = [2]uint16{vid, pid}
	}
	if !s.wantWatch {
		target = [2]uint16{}
	}
	if target == s.watchFor && (s.keyWatch != nil) == (target != [2]uint16{}) {
		return
	}
	if s.keyWatch != nil {
		s.keyWatch.Stop()
		s.keyWatch = nil
	}
	s.watchFor = target
	if target != ([2]uint16{}) {
		s.keyWatch = keyblock.Watch(target[0], target[1])
	}
}

// Autostart reports whether the app starts with Windows.
func (s *PeripheralService) Autostart() bool {
	if schtasks("/Query", "/TN", runName) == nil {
		return true
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(runName)
	return err == nil
}

// SetAutostart starts the app hidden in the tray at login, or stops doing so.
// When elevated it registers a highest-privilege logon task, so fan control
// works at startup without a UAC prompt; otherwise it uses the Run key.
func (s *PeripheralService) SetAutostart(enabled bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if err := k.DeleteValue(runName); err != nil && err != registry.ErrNotExist {
		return err
	}
	if schtasks("/Query", "/TN", runName) == nil {
		if err := schtasks("/Delete", "/TN", runName, "/F"); err != nil {
			return fmt.Errorf("remove startup task (needs administrator): %w", err)
		}
	}
	if !enabled {
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := fmt.Sprintf(`"%s" %s`, exe, hiddenFlag)
	if elevated() {
		return schtasks("/Create", "/TN", runName, "/TR", cmd, "/SC", "ONLOGON", "/RL", "HIGHEST", "/F")
	}
	return k.SetStringValue(runName, cmd)
}

func schtasks(args ...string) error {
	c := exec.Command("schtasks.exe", args...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	return c.Run()
}
