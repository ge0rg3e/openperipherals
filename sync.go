package main

import (
	"errors"
	"fmt"
	"math"
	"time"

	"openperipherals/internal/corsair"
)

// SyncLighting is one lighting setting applied to every device at once.
type SyncLighting struct {
	Enabled bool   `json:"enabled"`
	Effect  string `json:"effect"` // static, spectrum, wave, breathing, off
	Color   string `json:"color"`
}

// The SABRE has no hardware effects, so animated sync effects are streamed
// to it (and to keyboards that take custom frames) at this rate.
const syncFrame = 40 * time.Millisecond

func animated(effect string) bool {
	return effect == "spectrum" || effect == "wave" || effect == "breathing"
}

// Sync returns the current sync setting.
func (s *PeripheralService) Sync() SyncLighting {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.Sync
}

// SetSync applies one effect and colour to every device, or turns sync off
// (each device then keeps what it shows).
func (s *PeripheralService) SetSync(sy SyncLighting) error {
	switch sy.Effect {
	case "static", "spectrum", "wave", "breathing", "off":
	default:
		return fmt.Errorf("unknown sync effect %q", sy.Effect)
	}
	if _, err := parseHex(sy.Color); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.Sync = sy
	s.save()
	defer s.pokeLighting()
	if !sy.Enabled {
		return nil // each device keeps what it shows
	}

	s.cfg.Keyboard = syncKeyboardLighting(sy, s.cfg.Keyboard)
	s.save()
	anim := animated(sy.Effect)
	var errs []error
	// The lighting loop streams animations to keyboards that take custom
	// frames; every other keyboard runs the matching hardware effect.
	if _, frames := s.kb.(frameKeyboard); s.kb != nil && (!anim || !frames) {
		if err := s.kb.setEffect(s.cfg.Keyboard); err != nil {
			errs = append(errs, fmt.Errorf("keyboard: %w", err))
		}
	}
	if lm, ok := s.sm.(lightingMouse); ok {
		l := syncMouseLighting(sy)
		if err := lm.setLighting(l); err != nil {
			errs = append(errs, fmt.Errorf("mouse: %w", err))
		} else {
			s.cfg.MouseLighting = &l
			s.save()
		}
	}
	if anim {
		return errors.Join(errs...) // the lighting loop drives the SABRE from the same clock
	}
	zones, _ := syncStaticZones(sy)
	s.cfg.MouseColors = zoneHex(zones)
	s.mousePreview = nil
	s.save()
	if s.ms != nil {
		s.mouseIO.Lock()
		err := s.ms.StoreColors(s.dimMouse(zones))
		s.mouseIO.Unlock()
		if err != nil {
			errs = append(errs, fmt.Errorf("mouse: %w", err))
		}
	}
	return errors.Join(errs...)
}

// syncKeyboardLighting is the keyboard lighting a sync setting maps to,
// keeping the keyboard's other preferences.
func syncKeyboardLighting(sy SyncLighting, prev KeyboardLighting) KeyboardLighting {
	return KeyboardLighting{Effect: sy.Effect, Color: sy.Color, Color2: prev.Color2, Mode: "single", Speed: prev.Speed, Direction: "right"}
}

// syncMouseLighting is the hardware effect a sync setting maps to on a mouse
// with effects; it has no wave, so wave shows the spectrum.
func syncMouseLighting(sy SyncLighting) KeyboardLighting {
	effect := sy.Effect
	if effect == "wave" {
		effect = "spectrum"
	}
	return KeyboardLighting{Effect: effect, Color: sy.Color, Color2: sy.Color, Mode: "single", Speed: 2, Direction: "right"}
}

// syncStaticZones is every mouse zone in the sync colour (black for off).
func syncStaticZones(sy SyncLighting) ([corsair.ZoneCount]corsair.RGB, error) {
	c, err := parseHex(sy.Color)
	if err != nil {
		return [corsair.ZoneCount]corsair.RGB{}, err
	}
	zone := corsair.RGB{R: c[0], G: c[1], B: c[2]}
	if sy.Effect == "off" {
		zone = corsair.RGB{}
	}
	return [corsair.ZoneCount]corsair.RGB{zone, zone, zone, zone}, nil
}

// stopSyncForDevice turns sync off when a device's lighting is edited
// directly. Caller holds s.mu.
func (s *PeripheralService) stopSyncForDevice() {
	if !s.cfg.Sync.Enabled {
		return
	}
	s.cfg.Sync.Enabled = false
	s.save()
	s.pokeLighting()
}

// Zone positions front-to-back, so a wave travels along the mouse.
var zonePos = [corsair.ZoneCount]float64{0.45, 0.7, 0.2, 0.1}

// syncColor is the colour of the sync effect at position pos (0..1 across a
// device) and time t in seconds. The formulas match the UI preview.
func syncColor(sy SyncLighting, pos, t float64) corsair.RGB {
	c, _ := parseHex(sy.Color)
	var rgb [3]float64
	switch sy.Effect {
	case "spectrum":
		rgb = hue(t / 6)
	case "wave":
		rgb = hue(pos - t/4)
	case "breathing":
		k := (1 - math.Cos(2*math.Pi*t/3.2)) / 2
		rgb = [3]float64{float64(c[0]) * k, float64(c[1]) * k, float64(c[2]) * k}
	case "off":
	default:
		rgb = [3]float64{float64(c[0]), float64(c[1]), float64(c[2])}
	}
	return corsair.RGB{R: uint8(rgb[0]), G: uint8(rgb[1]), B: uint8(rgb[2])}
}

// syncZones computes one mouse frame; the DPI LED shows the active stage.
func syncZones(sy SyncLighting, t float64, dpi corsair.RGB) [corsair.ZoneCount]corsair.RGB {
	var out [corsair.ZoneCount]corsair.RGB
	for i := range out {
		out[i] = syncColor(sy, zonePos[i], t)
	}
	out[corsair.ZoneDPI] = dpi
	return out
}

// hue returns a fully saturated colour around the colour wheel (h in turns).
func hue(h float64) [3]float64 {
	h -= math.Floor(h)
	f := func(n float64) float64 {
		k := math.Mod(n+h*6, 6)
		return 255 * (1 - math.Max(0, math.Min(math.Min(k, 4-k), 1)))
	}
	return [3]float64{f(5), f(3), f(1)}
}
