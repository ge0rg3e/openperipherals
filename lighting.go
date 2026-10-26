package main

import (
	"time"

	"openperipherals/internal/corsair"
	"openperipherals/internal/razer"
)

// The lighting loop owns what the devices show while the app runs:
//
//   - Mouse: always driven live (software mode), because the SABRE only
//     loads its saved colours at power-up, so writing them doesn't change
//     the display. Colours are still saved for when the app isn't running.
//     The DPI LED follows the active stage, polled twice a second.
//   - Keyboard: hardware effects, except during an animated sync, when a
//     keyboard that takes custom frames (Razer) gets them from the same
//     clock as the mouse. Other keyboards run the matching hardware effect.
//
// Settings changes update s.cfg and poke the loop.

const (
	idleTick  = 250 * time.Millisecond
	stagePoll = 500 * time.Millisecond
)

func (s *PeripheralService) startLighting() {
	s.lightStop = make(chan struct{})
	s.lightDone = make(chan struct{})
	s.poke = make(chan struct{}, 1)
	go s.lightLoop()
}

// stopLighting ends the loop, which hands both devices back to their saved
// lighting. Must be called without holding s.mu.
func (s *PeripheralService) stopLighting() {
	if s.lightStop == nil {
		return
	}
	close(s.lightStop)
	<-s.lightDone
}

// pokeLighting asks the loop to apply a settings change now.
func (s *PeripheralService) pokeLighting() {
	select {
	case s.poke <- struct{}{}:
	default:
	}
}

func (s *PeripheralService) lightLoop() {
	defer close(s.lightDone)
	start := time.Now()
	var (
		kbCustom    frameKeyboard  // keyboard currently showing custom frames
		liveMs      *corsair.Mouse // mouse currently in software mode
		sent        [corsair.ZoneCount]corsair.RGB
		stagesFor   *corsair.Mouse
		stageColors [corsair.StageCount]corsair.RGB
		stage       int
		nextPoll    time.Time
	)
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-s.lightStop:
			s.mu.Lock()
			l := s.cfg.Keyboard
			s.mu.Unlock()
			if kbCustom != nil {
				kbCustom.setEffect(l)
			}
			if liveMs != nil {
				liveMs.HardwareMode()
			}
			return
		case <-timer.C:
		case <-s.poke:
		}

		s.mu.Lock()
		kb, ms, sy, kbLight := s.kb, s.ms, s.cfg.Sync, s.cfg.Keyboard
		preview := s.mousePreview
		dpiChanged := s.dpiChanged
		s.dpiChanged = false
		if ms != nil && s.cfg.MouseColors == nil {
			// First run: adopt what the mouse already has.
			if zones, err := ms.Colors(); err == nil {
				s.cfg.MouseColors = zoneHex(zones)
				s.save()
			}
		}
		chosen, chosenErr := zoneColors(s.cfg.MouseColors)
		level := s.mouseBrightness()
		s.mu.Unlock()

		anim := sy.Enabled && animated(sy.Effect)
		t := time.Since(start).Seconds()

		// Keyboard
		fk, frames := kb.(frameKeyboard)
		if fk != kbCustom {
			kbCustom = nil // replugged or gone
		}
		switch {
		case anim && frames:
			if kbCustom == nil && fk.startCustom() == nil {
				kbCustom = fk
			}
			if kbCustom != nil {
				cols := float64(max(fk.columns()-1, 1))
				fk.writeFrame(func(_, col int) razer.RGB {
					c := syncColor(sy, float64(col)/cols, t)
					return razer.RGB{R: c.R, G: c.G, B: c.B}
				})
			}
		case kbCustom != nil:
			// Animation ended: back to the keyboard's hardware effect.
			kbCustom.setEffect(kbLight)
			kbCustom = nil
		}

		// Mouse
		if ms == nil {
			liveMs, stagesFor = nil, nil
		} else {
			s.mouseIO.Lock()
			now := time.Now()
			if ms != stagesFor || dpiChanged {
				if cfg, err := ms.DPI(); err == nil {
					for i, st := range cfg.Stages {
						stageColors[i] = st.Color
					}
					stage = cfg.Current
				}
			}
			if ms != stagesFor || dpiChanged || now.After(nextPoll) {
				if cur, err := ms.CurrentStage(); err == nil && cur < corsair.StageCount {
					stage = cur
				}
				nextPoll = now.Add(stagePoll)
			}
			stagesFor = ms

			var zones [corsair.ZoneCount]corsair.RGB
			show := true
			switch {
			case anim:
				zones = syncZones(sy, t, corsair.RGB{})
			case preview != nil:
				zones = *preview
			case chosenErr == nil:
				zones = chosen
			default:
				show = false
			}
			if show {
				zones = dimZones(zones, level)
				zones[corsair.ZoneDPI] = stageColors[stage]
				if ms != liveMs {
					if ms.SetLiveColors(zones) == nil {
						liveMs, sent = ms, zones
					}
				} else if zones != sent {
					if ms.LiveFrame(zones) == nil {
						sent = zones
					}
				}
			}
			s.mouseIO.Unlock()
		}

		next := idleTick
		if anim {
			next = syncFrame
		}
		timer.Reset(next)
	}
}

// dimZones scales the lighting zones to a brightness percentage; the DPI
// indicator is left alone.
func dimZones(zones [corsair.ZoneCount]corsair.RGB, percent int) [corsair.ZoneCount]corsair.RGB {
	for i, c := range zones {
		if i != corsair.ZoneDPI {
			zones[i] = corsair.RGB{R: uint8(int(c.R) * percent / 100), G: uint8(int(c.G) * percent / 100), B: uint8(int(c.B) * percent / 100)}
		}
	}
	return zones
}
