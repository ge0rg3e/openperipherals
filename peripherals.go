package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"openperipherals/internal/corsair"
	"openperipherals/internal/hid"
	"openperipherals/internal/keyblock"
)

// KeyboardLighting is the keyboard effect selection as edited in the UI.
type KeyboardLighting struct {
	Effect    string `json:"effect"`
	Color     string `json:"color"`
	Color2    string `json:"color2"`
	Mode      string `json:"mode"`
	Speed     int    `json:"speed"`
	Direction string `json:"direction"`
}

// KeyboardState describes the connected keyboard and what it can do.
type KeyboardState struct {
	Name       string           `json:"name"`
	Serial     string           `json:"serial"`
	Firmware   string           `json:"firmware"`
	Brightness int              `json:"brightness"` // %
	Lighting   KeyboardLighting `json:"lighting"`
	Layout     string           `json:"layout"` // "full", "tkl" or "mini"
	Media      bool             `json:"media"`  // Huntsman Elite dial and media keys
	Effects    []string         `json:"effects"`
	// Speeds holds the top speed of each effect that takes one.
	Speeds        map[string]int `json:"speeds"`
	Directional   []string       `json:"directional"` // effects with a direction
	Modes         bool           `json:"modes"`       // dual and random breathing/starlight
	HasBrightness bool           `json:"hasBrightness"`
	Stored        bool           `json:"stored"`  // lighting is saved on the keyboard
	Battery       int            `json:"battery"` // %, -1 without a battery
	Charging      bool           `json:"charging"`
	GameMode      *bool          `json:"gameMode"`  // nil when unsupported
	MacroLEDs     *bool          `json:"macroLeds"` // nil when unsupported
}

// DPIStage is one mouse DPI stage.
type DPIStage struct {
	DPI     int    `json:"dpi"`
	Enabled bool   `json:"enabled"`
	Color   string `json:"color"`
}

// MouseState describes the connected mouse. Kind "corsair" (the SABRE)
// fills the zone and stage fields; kind "sensor" (Razer, Pulsar) fills the
// single DPI and sensor settings. Settings a mouse lacks are -1 or nil.
type MouseState struct {
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	Firmware string `json:"firmware"`
	MaxDPI   int    `json:"maxDpi"`

	Zones      []string   `json:"zones"`
	Colors     []string   `json:"colors"`
	Stages     []DPIStage `json:"stages"`
	Current    int        `json:"current"`
	Brightness int        `json:"brightness"` // %

	MinDPI       int      `json:"minDpi"`
	DPIStep      int      `json:"dpiStep"`
	DPI          int      `json:"dpi"`
	Polling      int      `json:"polling"` // Hz
	PollingRates []int    `json:"pollingRates"`
	LiftOff      int      `json:"liftOff"` // index into LiftOffs
	LiftOffs     []string `json:"liftOffs"`
	Sleep        int      `json:"sleep"` // seconds
	SleepOptions []int    `json:"sleepOptions"`
	Debounce     int      `json:"debounce"` // ms
	DebounceMax  int      `json:"debounceMax"`
	MotionSync   *bool    `json:"motionSync"`
	AngleSnap    *bool    `json:"angleSnap"`
	Ripple       *bool    `json:"ripple"`
	Battery      int      `json:"battery"` // %
	Charging     bool     `json:"charging"`
	Wireless     bool     `json:"wireless"`
	// Effects and Lighting are the hardware lighting of mice that have it.
	// The mouse can't report its effect, so Lighting is the last one set.
	Effects  []string          `json:"effects"`
	Lighting *KeyboardLighting `json:"lighting"`
}

// Status is the full device snapshot sent to the UI.
type Status struct {
	Keyboard *KeyboardState `json:"keyboard"`
	Mouse    *MouseState    `json:"mouse"`
}

// config holds what the devices can't report back themselves.
type config struct {
	Keyboard     KeyboardLighting  `json:"keyboard"`
	DisabledKeys map[string]uint16 `json:"disabledKeys"`
	BlockedKeys  []string          `json:"blockedKeys,omitempty"` // pre-1.0 format, migrated on start
	Sync         SyncLighting      `json:"sync"`
	Brightness   *int              `json:"brightness,omitempty"` // keyboard %, nil until first set
	// The SABRE has no brightness control: colours are kept here as chosen
	// and written scaled by MouseBrightness.
	MouseBrightness *int     `json:"mouseBrightness,omitempty"`
	MouseColors     []string `json:"mouseColors,omitempty"`
	// MouseLighting is the hardware effect last set on a Razer mouse.
	MouseLighting *KeyboardLighting `json:"mouseLighting,omitempty"`
}

// PeripheralService exposes the devices to the frontend.
type PeripheralService struct {
	mu        sync.Mutex
	scanMu    sync.Mutex // one device scan at a time
	kb        keyboard
	kbInfo    KeyboardState // kb.describe(), taken at connect
	ms        *corsair.Mouse
	sm        sensorMouse // a non-Corsair mouse; at most one of ms and sm is set
	cfg       config
	path      string
	stop      chan struct{}
	keyWatch  *keyblock.Watcher // attributes media key presses to the keyboard
	watchFor  [2]uint16         // vid, pid keyWatch follows
	wantWatch bool              // a media key is disabled
	// Lighting loop (lighting.go).
	lightStop, lightDone chan struct{}
	poke                 chan struct{}
	mousePreview         *[corsair.ZoneCount]corsair.RGB // colours shown while a picker is open
	dpiChanged           bool                            // tells the loop to re-read the DPI stages
	mouseIO              sync.Mutex                      // keeps a DPI change from interleaving with frames
}

func (s *PeripheralService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.cfg.Keyboard = KeyboardLighting{Effect: "spectrum", Color: "#00ff88", Color2: "#0066ff", Mode: "single", Speed: 2, Direction: "right"}
	s.cfg.Sync = SyncLighting{Effect: "static", Color: "#33e19a"}
	if dir, err := os.UserConfigDir(); err == nil {
		s.path = filepath.Join(dir, "openperipherals", "config.json")
		if b, err := os.ReadFile(s.path); err == nil {
			json.Unmarshal(b, &s.cfg)
		}
	}
	s.migrateKeys()
	if err := s.applyBlocking(s.cfg.DisabledKeys); err != nil {
		return fmt.Errorf("key blocking: %w", err)
	}
	s.stop = make(chan struct{})
	s.startLighting()
	s.connect()
	go s.watch()
	return nil
}

func (s *PeripheralService) ServiceShutdown() error {
	close(s.stop)
	keyblock.Set(nil)
	if s.keyWatch != nil {
		s.keyWatch.Stop()
	}
	s.stopLighting() // hands the devices back to their saved lighting
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.kb != nil {
		s.kb.close()
	}
	if s.ms != nil {
		s.ms.Close()
	}
	if s.sm != nil {
		s.sm.close()
	}
	return nil
}

// watch reconnects devices that are missing. Polling only happens while a
// device is absent, so an idle app with both devices connected does no I/O.
func (s *PeripheralService) watch() {
	t := time.NewTicker(3 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
			if s.connect() {
				s.emit()
			}
		}
	}
}

// connect opens any missing device and reports whether anything changed.
// Every brand is found in one HID enumeration that only opens supported
// products. Probing a device that doesn't answer can take a while, so it
// happens outside s.mu.
func (s *PeripheralService) connect() bool {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	s.mu.Lock()
	needKB, needMouse := s.kb == nil, s.ms == nil && s.sm == nil
	s.mu.Unlock()
	if !needKB && !needMouse {
		return false
	}
	infos, err := hid.EnumerateFunc(func(vid, pid uint16) bool { return supportedDevice(vid, pid, needKB, needMouse) })
	if err != nil || len(infos) == 0 {
		return false
	}
	var (
		kb keyboard
		ms *corsair.Mouse
		sm sensorMouse
	)
	if needKB {
		kb = openKeyboard(infos)
	}
	if needMouse {
		if m, err := corsair.OpenFrom(infos); err == nil {
			ms = m
		} else {
			sm = openSensorMouse(infos)
		}
	}
	if kb == nil && ms == nil && sm == nil {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if kb != nil {
		s.kb, s.kbInfo = kb, kb.describe()
		s.restoreKeyboard()
		s.followKeyboard()
	}
	if ms != nil {
		s.ms = ms
	}
	if sm != nil {
		s.sm = sm
		s.restoreMouseLighting()
	}
	s.pokeLighting()
	return true
}

// restoreKeyboard re-applies the saved lighting and brightness to a newly
// connected keyboard, so it shows what the app shows even after a power
// cycle or another program changed it. Caller holds s.mu.
func (s *PeripheralService) restoreKeyboard() {
	l := s.cfg.Keyboard
	if s.cfg.Sync.Enabled {
		if _, frames := s.kb.(frameKeyboard); frames && animated(s.cfg.Sync.Effect) {
			// The lighting loop drives it; just restore brightness.
			s.restoreBrightness()
			return
		}
		l = syncKeyboardLighting(s.cfg.Sync, l)
	}
	s.kb.setEffect(l)
	s.restoreBrightness()
}

func (s *PeripheralService) restoreBrightness() {
	if s.cfg.Brightness != nil {
		s.kb.setBrightness(*s.cfg.Brightness)
	}
}

// restoreMouseLighting applies the sync setting to a newly connected mouse
// with hardware effects. Caller holds s.mu.
func (s *PeripheralService) restoreMouseLighting() {
	lm, ok := s.sm.(lightingMouse)
	if !ok || !s.cfg.Sync.Enabled {
		return
	}
	l := syncMouseLighting(s.cfg.Sync)
	if lm.setLighting(l) == nil {
		s.cfg.MouseLighting = &l
		s.save()
	}
}

func (s *PeripheralService) emit() {
	if app := application.Get(); app != nil {
		app.Event.Emit("status", s.Status())
	}
}

// Status returns a snapshot of both devices; a nil field means disconnected.
func (s *PeripheralService) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	var st Status
	if s.kb != nil {
		k := s.kbInfo
		k.Lighting = s.cfg.Keyboard
		k.Brightness = 100
		if s.cfg.Brightness != nil {
			k.Brightness = *s.cfg.Brightness
		}
		b, ok, err := s.kb.brightness()
		if err != nil && s.dropKeyboard() {
			return s.mouseStatus(st)
		}
		if ok && err == nil {
			k.Brightness = b
		}
		if rk, ok := asRazer(s.kb); ok && rk.kb.Model.Battery {
			if pct, charging, err := rk.kb.Battery(); err == nil {
				k.Battery, k.Charging = pct, charging
			}
		}
		st.Keyboard = &k
	}
	return s.mouseStatus(st)
}

func (s *PeripheralService) mouseStatus(st Status) Status {
	if s.sm != nil {
		m, err := s.sm.state()
		if err != nil {
			s.dropMouse()
			return st
		}
		if _, ok := s.sm.(lightingMouse); ok {
			m.Lighting = s.cfg.MouseLighting
		}
		st.Mouse = m
		return st
	}
	if s.ms == nil {
		return st
	}
	dpi, err := s.ms.DPI()
	if err != nil {
		s.dropMouse()
		return st
	}
	colors, err := s.ms.Colors()
	if err != nil {
		s.dropMouse()
		return st
	}
	if chosen, err := zoneColors(s.cfg.MouseColors); err == nil {
		colors = chosen // the mouse holds them dimmed by brightness
	}
	m := &MouseState{
		Kind: "corsair", Name: s.ms.Model.Name, Firmware: s.ms.Firmware, MaxDPI: s.ms.Model.MaxDPI,
		Zones: corsair.ZoneNames[:], Current: dpi.Current, Brightness: s.mouseBrightness(),
		LiftOff: -1, Sleep: -1, Debounce: -1, Battery: -1,
	}
	// The DPI indicator shows the active stage's colour.
	colors[corsair.ZoneDPI] = dpi.Stages[dpi.Current].Color
	for _, c := range colors {
		m.Colors = append(m.Colors, hex(c.R, c.G, c.B))
	}
	for _, st := range dpi.Stages {
		m.Stages = append(m.Stages, DPIStage{DPI: st.DPI, Enabled: st.Enabled, Color: hex(st.Color.R, st.Color.G, st.Color.B)})
	}
	st.Mouse = m
	return st
}

// Refresh rescans for devices.
func (s *PeripheralService) Refresh() Status {
	s.connect()
	return s.Status()
}

var (
	errNoKeyboard = errors.New("keyboard not connected")
	errNoMouse    = errors.New("mouse not connected")
)

// SetKeyboardLighting applies an effect; it is stored on the keyboard where
// the keyboard supports that, and re-applied on connect otherwise.
func (s *PeripheralService) SetKeyboardLighting(l KeyboardLighting) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.kb == nil {
		return errNoKeyboard
	}
	s.stopSyncForDevice()
	if err := s.kb.setEffect(l); err != nil {
		s.dropKeyboard()
		return err
	}
	s.cfg.Keyboard = l
	s.save()
	return nil
}

// SetKeyboardBrightness sets backlight brightness, 0-100 %.
func (s *PeripheralService) SetKeyboardBrightness(percent int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.kb == nil {
		return errNoKeyboard
	}
	percent = min(max(percent, 0), 100)
	if err := s.kb.setBrightness(percent); err != nil {
		s.dropKeyboard()
		return err
	}
	s.cfg.Brightness = &percent
	s.save()
	return nil
}

// SetKeyboardOption toggles a Razer keyboard setting stored on the
// keyboard: "gameMode" (Windows key disabled) or "macroLeds".
func (s *PeripheralService) SetKeyboardOption(option string, on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rk, ok := asRazer(s.kb)
	if !ok {
		return errNoKeyboard
	}
	var err error
	switch option {
	case "gameMode":
		if err = rk.kb.SetGameMode(on); err == nil {
			rk.gameMode = &on
		}
	case "macroLeds":
		if err = rk.kb.SetMacroLEDs(on); err == nil {
			rk.macroLEDs = &on
		}
	default:
		return fmt.Errorf("unknown keyboard setting %q", option)
	}
	if err != nil {
		s.dropKeyboard()
		return err
	}
	s.kbInfo = rk.describe()
	return nil
}

// PreviewMouseColors shows zone colours immediately without saving them.
func (s *PeripheralService) PreviewMouseColors(colors []string) error {
	zones, err := zoneColors(colors)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ms == nil {
		return errNoMouse
	}
	s.stopSyncForDevice()
	s.mousePreview = &zones
	s.pokeLighting()
	return nil
}

// SetMouseColors saves zone colours to the mouse's onboard profile. The DPI
// zone colour is ignored here; it comes from the active DPI stage.
func (s *PeripheralService) SetMouseColors(colors []string) error {
	zones, err := zoneColors(colors)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ms == nil {
		return errNoMouse
	}
	s.stopSyncForDevice()
	s.cfg.MouseColors = colors
	s.mousePreview = nil
	s.save()
	s.pokeLighting()
	s.mouseIO.Lock()
	err = s.ms.StoreColors(s.dimMouse(zones)) // for when the app isn't running
	s.mouseIO.Unlock()
	if err != nil {
		s.dropMouse()
		return err
	}
	return nil
}

// SetMouseBrightness sets the mouse lighting brightness (0-100 %). The
// SABRE has no brightness control, so colours are scaled before they are
// written; the DPI indicator is left at full brightness.
func (s *PeripheralService) SetMouseBrightness(percent int) error {
	percent = min(max(percent, 0), 100)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.MouseBrightness = &percent
	s.save()
	s.pokeLighting() // shown immediately by the lighting loop
	zones, err := zoneColors(s.cfg.MouseColors)
	if s.ms == nil || err != nil {
		return nil
	}
	s.mouseIO.Lock()
	defer s.mouseIO.Unlock()
	return s.ms.StoreColors(s.dimMouse(zones)) // for when the app isn't running
}

func (s *PeripheralService) mouseBrightness() int {
	if s.cfg.MouseBrightness == nil {
		return 100
	}
	return *s.cfg.MouseBrightness
}

// dimMouse scales the lighting zones by the mouse brightness.
func (s *PeripheralService) dimMouse(zones [corsair.ZoneCount]corsair.RGB) [corsair.ZoneCount]corsair.RGB {
	return dimZones(zones, s.mouseBrightness())
}

func zoneHex(zones [corsair.ZoneCount]corsair.RGB) []string {
	out := make([]string, 0, len(zones))
	for _, c := range zones {
		out = append(out, hex(c.R, c.G, c.B))
	}
	return out
}

// SetMouseDPI applies and saves the SABRE's DPI stages and active stage.
func (s *PeripheralService) SetMouseDPI(stages []DPIStage, current int) error {
	if len(stages) != corsair.StageCount {
		return fmt.Errorf("expected %d DPI stages", corsair.StageCount)
	}
	var cfg corsair.DPIConfig
	cfg.Current = current
	for i, st := range stages {
		c, err := parseHex(st.Color)
		if err != nil {
			return err
		}
		cfg.Stages[i] = corsair.Stage{DPI: st.DPI, Enabled: st.Enabled, Color: corsair.RGB{R: c[0], G: c[1], B: c[2]}}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ms == nil {
		return errNoMouse
	}
	// DPI is independent of lighting sync: it never ends sync, and the
	// animator pauses for the write so the packets aren't interleaved.
	s.mouseIO.Lock()
	err := s.ms.SetDPI(cfg)
	s.mouseIO.Unlock()
	if err != nil {
		s.dropMouse()
		return err
	}
	s.dpiChanged = true
	s.pokeLighting()
	return nil
}

// SetMouseOption changes one setting of a Razer or Pulsar mouse: "dpi",
// "polling" (Hz), "liftOff" (index), "sleep" (seconds), "debounce" (ms), or
// "motionSync", "angleSnap", "ripple" (0 or 1). Each is saved on the mouse
// and confirmed by reading it back.
func (s *PeripheralService) SetMouseOption(option string, value int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sm == nil {
		return errNoMouse
	}
	if err := s.sm.set(option, value); err != nil {
		if !errors.Is(err, errUnknownOption) {
			s.dropMouse()
		}
		return err
	}
	return nil
}

// SetMouseLighting applies a hardware effect to a mouse that has them
// (Razer). It is stored on the mouse.
func (s *PeripheralService) SetMouseLighting(l KeyboardLighting) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	lm, ok := s.sm.(lightingMouse)
	if !ok {
		return errNoMouse
	}
	s.stopSyncForDevice()
	if err := lm.setLighting(l); err != nil {
		s.dropMouse()
		return err
	}
	s.cfg.MouseLighting = &l
	s.save()
	return nil
}

// dropKeyboard forgets the keyboard if it was unplugged and reports whether
// it did. Errors from a device that is still present (e.g. a reply lost to
// other RGB software) keep the connection.
func (s *PeripheralService) dropKeyboard() bool {
	if present(s.kb.ids()) {
		return false
	}
	s.kb.close()
	s.kb = nil
	go s.emit()
	return true
}

func (s *PeripheralService) dropMouse() {
	switch {
	case s.sm != nil:
		if present(s.sm.ids()) {
			return
		}
		s.sm.close()
		s.sm = nil
	case s.ms != nil:
		if present(corsair.VendorID, s.ms.Model.PID) {
			return
		}
		s.ms.Close()
		s.ms = nil
	}
	go s.emit()
}

func present(vid, pid uint16) bool {
	infos, err := hid.Enumerate(vid, pid)
	return err == nil && len(infos) > 0
}

func (s *PeripheralService) save() {
	if s.path == "" {
		return
	}
	os.MkdirAll(filepath.Dir(s.path), 0o755)
	if b, err := json.MarshalIndent(s.cfg, "", "  "); err == nil {
		os.WriteFile(s.path, b, 0o644)
	}
}

func zoneColors(colors []string) ([corsair.ZoneCount]corsair.RGB, error) {
	var zones [corsair.ZoneCount]corsair.RGB
	if len(colors) != corsair.ZoneCount {
		return zones, fmt.Errorf("expected %d zone colours", corsair.ZoneCount)
	}
	for i, h := range colors {
		c, err := parseHex(h)
		if err != nil {
			return zones, err
		}
		zones[i] = corsair.RGB{R: c[0], G: c[1], B: c[2]}
	}
	return zones, nil
}

func parseHex(h string) ([3]uint8, error) {
	var c [3]uint8
	if len(h) == 7 && h[0] == '#' {
		h = h[1:]
	}
	if _, err := fmt.Sscanf(h, "%02x%02x%02x", &c[0], &c[1], &c[2]); err != nil || len(h) != 6 {
		return c, fmt.Errorf("invalid colour %q", h)
	}
	return c, nil
}

func hex(r, g, b uint8) string { return fmt.Sprintf("#%02x%02x%02x", r, g, b) }
