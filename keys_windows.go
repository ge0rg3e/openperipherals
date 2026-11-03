package main

import "openperipherals/internal/keyblock"

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
func (s *PeripheralService) startBlocking() error { return s.applyBlocking(s.cfg.DisabledKeys) }
