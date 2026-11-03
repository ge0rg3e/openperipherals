package main

import (
	"fmt"
	"maps"
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
