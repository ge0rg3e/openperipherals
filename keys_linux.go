package main

import (
	"errors"
	"log"

	"openperipherals/internal/keyblock"
)

func (*PeripheralService) startBlocking() error { return nil }

func (s *PeripheralService) applyBlocking(disabled map[string]uint16) error {
	if len(disabled) == 0 {
		return keyblock.Set(nil)
	}
	if s.kb == nil {
		return errors.New("connect the keyboard before blocking keys")
	}
	vid, pid := s.kb.ids()
	return keyblock.Apply(vid, pid, disabled)
}

func (s *PeripheralService) followKeyboard() {
	if len(s.cfg.DisabledKeys) > 0 {
		if err := s.applyBlocking(s.cfg.DisabledKeys); err != nil {
			log.Printf("key blocking: %v", err)
		}
	}
}
