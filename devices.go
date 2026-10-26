package main

import (
	"errors"
	"slices"

	"openperipherals/internal/corsair"
	"openperipherals/internal/hid"
	"openperipherals/internal/logitech"
	"openperipherals/internal/pulsar"
	"openperipherals/internal/razer"
	"openperipherals/internal/redragon"
)

// keyboard is a connected keyboard of any brand.
type keyboard interface {
	// describe fills the parts of the UI state that don't change while the
	// keyboard stays connected.
	describe() KeyboardState
	ids() (vid, pid uint16)
	setEffect(l KeyboardLighting) error
	setBrightness(percent int) error
	// brightness reads the backlight level in %; ok is false when the
	// keyboard can't report it.
	brightness() (percent int, ok bool, err error)
	close()
}

// frameKeyboard is a keyboard that can show host-driven frames, used to keep
// animated sync effects in step with the mouse.
type frameKeyboard interface {
	keyboard
	startCustom() error
	writeFrame(colour func(row, col int) razer.RGB) error
	columns() int
}

// supportedDevice reports whether vid:pid is a device any driver handles.
func supportedDevice(vid, pid uint16, kb, mouse bool) bool {
	switch vid {
	case razer.VendorID:
		return kb && razer.Supported(pid) || mouse && razer.MouseSupported(pid)
	case logitech.VendorID:
		return kb && logitech.Supported(pid)
	case redragon.VendorID:
		return kb && redragon.Supported(pid)
	case corsair.VendorID:
		return mouse && corsair.Supported(pid)
	case pulsar.VendorID, pulsar.VGNVendorID:
		return mouse
	}
	return false
}

func openKeyboard(infos []hid.Info) keyboard {
	if kb, err := razer.OpenFrom(infos); err == nil {
		if kb.Model.Custom {
			return razerFrames{newRazerKeyboard(kb)}
		}
		return newRazerKeyboard(kb)
	}
	if kb, err := logitech.OpenFrom(infos); err == nil {
		return &logitechKeyboard{kb}
	}
	if kb, err := redragon.OpenFrom(infos); err == nil {
		return &redragonKeyboard{kb: kb, level: 100}
	}
	return nil
}

// Razer

type razerKeyboard struct {
	kb                  *razer.Keyboard
	gameMode, macroLEDs *bool // nil when the keyboard doesn't answer
}

func newRazerKeyboard(kb *razer.Keyboard) *razerKeyboard {
	k := &razerKeyboard{kb: kb}
	if on, err := kb.GameMode(); err == nil {
		k.gameMode = &on
	}
	if on, err := kb.MacroLEDs(); err == nil {
		k.macroLEDs = &on
	}
	return k
}

func (k *razerKeyboard) describe() KeyboardState {
	m := k.kb.Model
	effects := []string{"static", "spectrum", "wave", "breathing", "reactive", "starlight", "off"}
	if m.Wheel {
		effects = slices.Insert(effects, 3, "wheel")
	}
	return KeyboardState{
		Name: m.Name, Serial: k.kb.Serial, Firmware: k.kb.Firmware, Layout: m.Layout,
		Media: m.PID == 0x0226, Effects: effects, Modes: true, HasBrightness: true, Stored: true,
		Speeds: map[string]int{"reactive": 4, "starlight": 3}, Directional: []string{"wave", "wheel"},
		Battery: -1, GameMode: k.gameMode, MacroLEDs: k.macroLEDs,
	}
}

func (k *razerKeyboard) ids() (uint16, uint16) { return razer.VendorID, k.kb.Model.PID }

func (k *razerKeyboard) setEffect(l KeyboardLighting) error {
	c1, c2, err := lightingColors(l)
	if err != nil {
		return err
	}
	return k.kb.SetEffect(razer.EffectParams{
		Effect: razer.Effect(l.Effect), Mode: l.Mode, Speed: l.Speed, Direction: l.Direction,
		Color: razer.RGB{R: c1[0], G: c1[1], B: c1[2]}, Color2: razer.RGB{R: c2[0], G: c2[1], B: c2[2]},
	})
}

func (k *razerKeyboard) setBrightness(percent int) error {
	return k.kb.SetBrightness(uint8(min(max(percent, 0), 100) * 255 / 100))
}

func (k *razerKeyboard) brightness() (int, bool, error) {
	b, err := k.kb.Brightness()
	return (int(b)*100 + 127) / 255, true, err
}

func (k *razerKeyboard) close() { k.kb.Close() }

// razerFrames is a Razer keyboard that takes custom frames.
type razerFrames struct{ *razerKeyboard }

func (k razerFrames) startCustom() error { return k.kb.StartCustom() }
func (k razerFrames) writeFrame(c func(row, col int) razer.RGB) error {
	return k.kb.WriteFrame(c)
}
func (k razerFrames) columns() int { return k.kb.Model.Cols }

// asRazer returns the Razer keyboard behind kb, if it is one.
func asRazer(kb keyboard) (*razerKeyboard, bool) {
	switch k := kb.(type) {
	case *razerKeyboard:
		return k, true
	case razerFrames:
		return k.razerKeyboard, true
	}
	return nil, false
}

// Logitech

type logitechKeyboard struct{ kb *logitech.Keyboard }

func (k *logitechKeyboard) describe() KeyboardState {
	m := k.kb.Model
	speeds := map[string]int{}
	var effects []string
	for _, e := range m.Effects {
		effects = append(effects, string(e))
		if e != logitech.EffectOff && e != logitech.EffectStatic {
			speeds[string(e)] = 4
		}
	}
	var dir []string
	if m.Family == logitech.G213 {
		dir = []string{"wave"}
	}
	return KeyboardState{Name: m.Name, Layout: m.Layout, Effects: effects, Speeds: speeds, Directional: dir, Battery: -1}
}

func (k *logitechKeyboard) ids() (uint16, uint16) { return logitech.VendorID, k.kb.Model.PID }

func (k *logitechKeyboard) setEffect(l KeyboardLighting) error {
	c, _, err := lightingColors(l)
	if err != nil {
		return err
	}
	return k.kb.SetEffect(logitech.EffectParams{
		Effect: logitech.Effect(l.Effect), Color: logitech.RGB{R: c[0], G: c[1], B: c[2]},
		Speed: l.Speed, Direction: l.Direction,
	})
}

// Brightness lives inside the effect packets; there is no separate control.
func (k *logitechKeyboard) setBrightness(int) error        { return nil }
func (k *logitechKeyboard) brightness() (int, bool, error) { return 100, false, nil }
func (k *logitechKeyboard) close()                         { k.kb.Close() }

// Redragon

type redragonKeyboard struct {
	kb    *redragon.Keyboard
	last  *KeyboardLighting // brightness is part of the effect packet
	level int               // %
}

func (k *redragonKeyboard) describe() KeyboardState {
	return KeyboardState{
		Name: k.kb.Name, Layout: k.kb.Model.Layout, Effects: redragon.Effects, HasBrightness: true,
		Speeds:      map[string]int{"wave": 4, "breathing": 4, "reactive": 4, "starlight": 4},
		Directional: []string{"wave"}, Battery: -1,
	}
}

func (k *redragonKeyboard) ids() (uint16, uint16) { return redragon.VendorID, k.kb.Model.PID }

func (k *redragonKeyboard) setEffect(l KeyboardLighting) error {
	c, _, err := lightingColors(l)
	if err != nil {
		return err
	}
	if err := k.kb.SetEffect(redragon.EffectParams{
		Effect: l.Effect, Color: redragon.RGB{R: c[0], G: c[1], B: c[2]}, Speed: l.Speed,
		Direction: l.Direction, Brightness: (k.level*4 + 50) / 100,
	}); err != nil {
		return err
	}
	k.last = &l
	return nil
}

func (k *redragonKeyboard) setBrightness(percent int) error {
	k.level = min(max(percent, 0), 100)
	if k.last == nil {
		return nil
	}
	return k.setEffect(*k.last)
}

func (k *redragonKeyboard) brightness() (int, bool, error) { return k.level, false, nil }
func (k *redragonKeyboard) close()                         { k.kb.Close() }

// lightingColors parses both colours of a lighting setting.
func lightingColors(l KeyboardLighting) (c1, c2 [3]uint8, err error) {
	if c1, err = parseHex(l.Color); err != nil {
		return
	}
	c2, err = parseHex(l.Color2)
	return
}

// Mice other than the SABRE: one DPI value plus sensor settings.

type sensorMouse interface {
	state() (*MouseState, error)
	set(option string, value int) error
	ids() (vid, pid uint16)
	close()
}

// lightingMouse is a sensor mouse with hardware lighting effects.
type lightingMouse interface {
	sensorMouse
	setLighting(l KeyboardLighting) error
}

var errUnknownOption = errors.New("unknown mouse setting")

func openSensorMouse(infos []hid.Info) sensorMouse {
	if m, err := razer.OpenMouseFrom(infos); err == nil {
		return &razerMouse{m}
	}
	if m, err := pulsar.OpenFrom(infos); err == nil {
		return &pulsarMouse{m}
	}
	return nil
}

type razerMouse struct{ m *razer.Mouse }

func (r *razerMouse) ids() (uint16, uint16) { return razer.VendorID, r.m.Model.PID }
func (r *razerMouse) close()                { r.m.Close() }

func (r *razerMouse) state() (*MouseState, error) {
	m := r.m.Model
	x, _, err := r.m.DPI()
	if err != nil {
		return nil, err
	}
	st := &MouseState{
		Kind: "sensor", Name: m.Name, Firmware: r.m.Firmware, MaxDPI: m.MaxDPI, MinDPI: razer.MinDPI,
		DPIStep: 50, DPI: x, PollingRates: m.PollingRates, LiftOff: -1, Sleep: -1, Debounce: -1,
		Battery: -1, Wireless: m.Wireless, Effects: []string{"static", "spectrum", "breathing", "reactive", "off"},
	}
	// Wired mice without a cell answer battery and power commands as
	// unsupported; that only hides the control.
	if hz, err := r.m.PollingRate(); err == nil {
		st.Polling = hz
	}
	if lvl, err := r.m.LiftOff(); err == nil {
		st.LiftOff, st.LiftOffs = lvl, razer.LiftOffs
	}
	if sec, err := r.m.Sleep(); err == nil {
		st.Sleep, st.SleepOptions = sec, razer.SleepOptions
	}
	if pct, charging, err := r.m.Battery(); err == nil {
		st.Battery, st.Charging = pct, charging
	}
	return st, nil
}

func (r *razerMouse) set(option string, v int) error {
	switch option {
	case "dpi":
		v = min(max(v, razer.MinDPI), r.m.Model.MaxDPI)
		return r.m.SetDPI(v, v)
	case "polling":
		return r.m.SetPollingRate(v)
	case "liftOff":
		return r.m.SetLiftOff(v)
	case "sleep":
		return r.m.SetSleep(v)
	}
	return errUnknownOption
}

func (r *razerMouse) setLighting(l KeyboardLighting) error {
	c1, c2, err := lightingColors(l)
	if err != nil {
		return err
	}
	return r.m.SetEffect(razer.EffectParams{
		Effect: razer.Effect(l.Effect), Mode: l.Mode, Speed: l.Speed,
		Color: razer.RGB{R: c1[0], G: c1[1], B: c1[2]}, Color2: razer.RGB{R: c2[0], G: c2[1], B: c2[2]},
	})
}

type pulsarMouse struct{ m *pulsar.Mouse }

func (p *pulsarMouse) ids() (uint16, uint16) { return p.m.VID, p.m.PID }
func (p *pulsarMouse) close()                { p.m.Close() }

// Pulsar sleep values are in units of 10 seconds; the UI works in seconds.
const pulsarSleepUnit = 10

func (p *pulsarMouse) state() (*MouseState, error) {
	s, err := p.m.Status()
	if err != nil {
		return nil, err
	}
	dpis := p.m.DPIOptions()
	lo, hi := dpis[0], dpis[len(dpis)-1]
	var rates []int
	for _, hz := range pulsar.PollingRates {
		if hz <= p.m.MaxPolling {
			rates = append(rates, hz)
		}
	}
	sleep := make([]int, len(pulsar.SleepOptions))
	for i, v := range pulsar.SleepOptions {
		sleep[i] = v * pulsarSleepUnit
	}
	return &MouseState{
		Kind: "sensor", Name: p.m.Name, Firmware: s.Firmware, MaxDPI: hi, MinDPI: lo, DPIStep: 10,
		DPI: s.DPI, Polling: s.Polling, PollingRates: rates, LiftOff: s.LiftOff, LiftOffs: pulsar.LiftOffs,
		Sleep: s.Sleep * pulsarSleepUnit, SleepOptions: sleep, Debounce: s.Debounce, DebounceMax: pulsar.DebounceMax,
		MotionSync: &s.MotionSync, AngleSnap: &s.AngleSnap, Ripple: &s.Ripple,
		Battery: s.Battery, Charging: s.Charging, Wireless: !p.m.Wired,
	}, nil
}

func (p *pulsarMouse) set(option string, v int) error {
	switch option {
	case "dpi":
		return p.m.SetDPI(p.m.SnapDPI(v))
	case "polling":
		return p.m.SetPolling(v)
	case "liftOff":
		return p.m.SetLiftOff(v)
	case "sleep":
		return p.m.SetSleep(v / pulsarSleepUnit)
	case "debounce":
		return p.m.SetDebounce(v)
	case "motionSync":
		return p.m.SetMotionSync(v != 0)
	case "angleSnap":
		return p.m.SetAngleSnap(v != 0)
	case "ripple":
		return p.m.SetRipple(v != 0)
	}
	return errUnknownOption
}
