package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"openperipherals/internal/nvgpu"
	"openperipherals/internal/pawnio"
	"openperipherals/internal/superio"
)

const (
	fanTick         = time.Second
	mbMinDuty       = 20 // motherboard fans never go lower under app control
	panicTemp       = 90 // °C on the CPU sensor: every controlled fan to 100 %
	rampUpPerTick   = 10
	rampDownPerTick = 2
)

// FanConfig is how one fan is driven.
type FanConfig struct {
	Mode   string   `json:"mode"` // "auto", "fixed" or "curve"
	Fixed  int      `json:"fixed"`
	Sensor string   `json:"sensor"`
	Points [][2]int `json:"points"` // (°C, %) pairs, sorted by temperature
}

// Fan is one controllable fan as shown in the UI.
type Fan struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Source  string    `json:"source"`
	RPM     int       `json:"rpm"`  // -1 when the fan has no tachometer
	Duty    int       `json:"duty"` // -1 while firmware controls it
	MinDuty int       `json:"minDuty"`
	Config  FanConfig `json:"config"`
}

// TempSensor is a temperature a curve can follow.
type TempSensor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Temp int    `json:"temp"` // -1 when unavailable
}

// FanPreset is a named set of fan configs that can be applied at once.
type FanPreset struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Builtin bool             `json:"builtin"`
	Configs []FanPresetEntry `json:"configs"`
}

// FanPresetEntry is the config a preset applies to one fan.
type FanPresetEntry struct {
	FanID  string    `json:"fanId"`
	Config FanConfig `json:"config"`
}

// FanStatus is the fan page snapshot.
type FanStatus struct {
	Available bool         `json:"available"`
	Notice    string       `json:"notice"` // why motherboard fans are missing, if they are
	Elevated  bool         `json:"elevated"`
	Fans      []Fan        `json:"fans"`
	Sensors   []TempSensor `json:"sensors"`
	Presets   []FanPreset  `json:"presets"`
}

type fanDev struct {
	Fan
	set     func(int) error
	restore func() error
	applied int // last duty written, -1 when in auto
}

// FanService detects fans and runs the control loop.
type FanService struct {
	mu         sync.Mutex
	chip       *superio.Chip
	gpus       []*nvgpu.GPU
	fans       []*fanDev
	sensors    []TempSensor
	cfg        map[string]FanConfig
	presets    []FanPreset // custom presets, built-ins are generated
	path       string
	presetPath string
	notice     string
	stop       chan struct{}
	done       chan struct{}
}

func (f *FanService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	f.cfg = map[string]FanConfig{}
	if dir, err := os.UserConfigDir(); err == nil {
		dir = filepath.Join(dir, "openperipherals")
		f.path = filepath.Join(dir, "fans.json")
		if b, err := os.ReadFile(f.path); err == nil {
			json.Unmarshal(b, &f.cfg)
		}
		f.presetPath = filepath.Join(dir, "fan-presets.json")
		if b, err := os.ReadFile(f.presetPath); err == nil {
			json.Unmarshal(b, &f.presets)
		}
	}
	f.detect()
	f.stop, f.done = make(chan struct{}), make(chan struct{})
	go f.loop()
	return nil
}

// ServiceShutdown hands every fan back to its firmware.
func (f *FanService) ServiceShutdown() error {
	close(f.stop)
	<-f.done
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, d := range f.fans {
		d.restore()
	}
	if f.chip != nil {
		f.chip.Close()
	}
	for _, g := range f.gpus {
		g.Close()
	}
	return nil
}

func (f *FanService) detect() {
	switch {
	case !pawnio.Installed():
		f.notice = "Motherboard fans need the PawnIO driver (pawnio.eu)."
	case !elevated():
		f.notice = "Motherboard fans need administrator rights."
	default:
		chip, err := superio.Detect()
		if err != nil {
			f.notice = "Motherboard fans: " + err.Error()
			break
		}
		f.chip = chip
		fanNames, tempNames := superio.Labels(boardName())
		r, _ := chip.Read()
		for i := range chip.Controls {
			name := fmt.Sprintf("Fan #%d", i+1)
			if fanNames != nil {
				n, ok := fanNames[i]
				if !ok {
					continue
				}
				name = n
			}
			f.fans = append(f.fans, &fanDev{
				Fan:     Fan{ID: fmt.Sprintf("mb:%d", i), Name: name, Source: chip.Name, MinDuty: mbMinDuty},
				set:     func(d int) error { return chip.SetDuty(i, d) },
				restore: func() error { return chip.Restore(i) },
				applied: -1,
			})
		}
		for i, t := range r.Temps {
			name := fmt.Sprintf("Temperature #%d", i+1)
			if tempNames != nil {
				n, ok := tempNames[i]
				if !ok {
					continue
				}
				name = n
			} else if t < 0 {
				continue
			}
			f.sensors = append(f.sensors, TempSensor{ID: fmt.Sprintf("mb:t%d", i), Name: name, Temp: t})
		}
	}

	if gpus, err := nvgpu.Enumerate(); err == nil {
		f.gpus = gpus
		for gi, g := range gpus {
			f.sensors = append(f.sensors, TempSensor{ID: fmt.Sprintf("gpu%d", gi), Name: g.Name, Temp: -1})
			for c := range g.Coolers {
				f.fans = append(f.fans, &fanDev{
					Fan:     Fan{ID: fmt.Sprintf("gpu%d:%d", gi, c), Name: "GPU Fan", Source: g.Name, MinDuty: g.MinDuty[c]},
					set:     func(d int) error { return g.SetDuty(c, d) },
					restore: func() error { return g.Restore(c) },
					applied: -1,
				})
			}
		}
	}

	for _, d := range f.fans {
		c, ok := f.cfg[d.ID]
		if !ok {
			c = FanConfig{Mode: "auto", Fixed: 50}
		}
		if c.Sensor == "" {
			c.Sensor = f.defaultSensor(d.ID)
		}
		if len(c.Points) < 2 {
			c.Points = defaultFanPoints()
		}
		d.Config = c
	}
}

// defaultFanPoints is the curve a fan starts with.
func defaultFanPoints() [][2]int {
	return [][2]int{{30, 30}, {50, 40}, {70, 70}, {85, 100}}
}

// defaultSensor picks the CPU sensor for motherboard fans and the GPU's own
// temperature for GPU fans.
func (f *FanService) defaultSensor(fanID string) string {
	if gpu, _, ok := strings.Cut(fanID, ":"); ok && strings.HasPrefix(gpu, "gpu") {
		return gpu
	}
	for _, s := range f.sensors {
		if s.Name == "CPU" {
			return s.ID
		}
	}
	if len(f.sensors) > 0 {
		return f.sensors[0].ID
	}
	return ""
}

func (f *FanService) loop() {
	defer close(f.done)
	t := time.NewTicker(fanTick)
	defer t.Stop()
	for {
		select {
		case <-f.stop:
			return
		case <-t.C:
			f.mu.Lock()
			f.tick()
			st := f.statusLocked()
			f.mu.Unlock()
			if app := application.Get(); app != nil {
				app.Event.Emit("fans", st)
			}
		}
	}
}

// tick reads every sensor, then drives each fan from its config.
func (f *FanService) tick() {
	temps := map[string]int{}
	var mbRPM, mbDuty []int
	if f.chip != nil {
		if r, err := f.chip.Read(); err == nil {
			for i, t := range r.Temps {
				temps[fmt.Sprintf("mb:t%d", i)] = t
			}
			mbRPM, mbDuty = r.FanRPM, r.Duty
		}
	}
	gpuDuty := map[string][]int{}
	for gi, g := range f.gpus {
		id := fmt.Sprintf("gpu%d", gi)
		if t, err := g.Temperature(); err == nil {
			temps[id] = t
		} else {
			temps[id] = -1
		}
		gpuDuty[id], _ = g.Duty()
	}
	for i := range f.sensors {
		if t, ok := temps[f.sensors[i].ID]; ok {
			f.sensors[i].Temp = t
		}
	}
	panicking := false
	for _, s := range f.sensors {
		if s.Name == "CPU" && s.Temp >= panicTemp {
			panicking = true
		}
	}

	for _, d := range f.fans {
		d.RPM, d.Duty = -1, -1
		var idx int
		if gpu, c, ok := strings.Cut(d.ID, ":"); ok && strings.HasPrefix(gpu, "gpu") {
			fmt.Sscanf(c, "%d", &idx)
			if duty := gpuDuty[gpu]; idx < len(duty) {
				d.Duty = duty[idx]
			}
		} else {
			fmt.Sscanf(strings.TrimPrefix(d.ID, "mb:"), "%d", &idx)
			if idx < len(mbRPM) {
				d.RPM = mbRPM[idx]
			}
			if idx < len(mbDuty) {
				d.Duty = mbDuty[idx]
			}
		}

		target := -1
		switch c := d.Config; c.Mode {
		case "fixed":
			target = c.Fixed
		case "curve":
			t, ok := temps[c.Sensor]
			if !ok || t < 0 {
				target = 100 // sensor lost: fail safe
			} else {
				target = interpolate(c.Points, t)
				if d.applied >= 0 { // ramp so fans don't hunt
					target = min(max(target, d.applied-rampDownPerTick), d.applied+rampUpPerTick)
				}
			}
		}
		if panicking && target >= 0 {
			target = 100
		}
		if target < 0 {
			if d.applied >= 0 {
				d.restore()
				d.applied = -1
			}
			continue
		}
		target = min(max(target, d.MinDuty), 100)
		if target != d.applied {
			if err := d.set(target); err == nil {
				d.applied = target
			}
		}
	}
}

func interpolate(points [][2]int, t int) int {
	if len(points) == 0 {
		return 100
	}
	if t <= points[0][0] {
		return points[0][1]
	}
	for i := 1; i < len(points); i++ {
		a, b := points[i-1], points[i]
		if t <= b[0] {
			if b[0] == a[0] {
				return b[1]
			}
			return a[1] + (b[1]-a[1])*(t-a[0])/(b[0]-a[0])
		}
	}
	return points[len(points)-1][1]
}

// Status returns the current fan snapshot.
func (f *FanService) Status() FanStatus {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.statusLocked()
}

func (f *FanService) statusLocked() FanStatus {
	st := FanStatus{Available: f.chip != nil || len(f.gpus) > 0, Notice: f.notice, Elevated: elevated()}
	for _, d := range f.fans {
		st.Fans = append(st.Fans, d.Fan)
	}
	st.Sensors = append(st.Sensors, f.sensors...)
	st.Presets = f.presetsLocked()
	return st
}

// SetFanConfig changes how a fan is driven; the loop applies it next tick.
func (f *FanService) SetFanConfig(id string, c FanConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, err := f.normalizeConfig(id, c)
	if err != nil {
		return err
	}
	if err := f.setConfigLocked(id, c); err != nil {
		return err
	}
	f.save()
	return nil
}

// ApplyPreset switches every fan in a preset at once.
func (f *FanService) ApplyPreset(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.findPresetLocked(id)
	if !ok {
		return fmt.Errorf("unknown preset %q", id)
	}
	for _, e := range p.Configs {
		c, err := f.normalizeConfig(e.FanID, e.Config)
		if err != nil {
			return fmt.Errorf("preset %q: %w", p.Name, err)
		}
		// A fan in the preset may be gone (unplugged GPU, different board).
		f.setConfigLocked(e.FanID, c)
	}
	f.save()
	return nil
}

// SavePreset stores the current config of every fan under a name and returns
// the new preset's id.
func (f *FanService) SavePreset(name string) (string, error) {
	name, err := presetName(name)
	if err != nil {
		return "", err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.fans) == 0 {
		return "", errors.New("there are no fans to save")
	}
	for _, p := range f.builtinPresetsLocked() {
		if strings.EqualFold(p.Name, name) {
			return "", fmt.Errorf("%q is a built-in preset name", name)
		}
	}
	for _, p := range f.presets {
		if strings.EqualFold(p.Name, name) {
			return "", fmt.Errorf("a preset named %q already exists", name)
		}
	}
	id := fmt.Sprintf("custom:%d", time.Now().UnixNano())
	f.presets = append(f.presets, FanPreset{
		ID:      id,
		Name:    name,
		Configs: f.currentConfigsLocked(),
	})
	f.savePresets()
	return id, nil
}

// UpdatePreset replaces a custom preset with the current config of every fan.
func (f *FanService) UpdatePreset(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.fans) == 0 {
		return errors.New("there are no fans to save")
	}
	for i := range f.presets {
		if f.presets[i].ID == id {
			f.presets[i].Configs = f.currentConfigsLocked()
			f.savePresets()
			return nil
		}
	}
	return errors.New("preset not found")
}

// DeletePreset removes a custom preset.
func (f *FanService) DeletePreset(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.presets {
		if f.presets[i].ID == id {
			f.presets = append(f.presets[:i], f.presets[i+1:]...)
			f.savePresets()
			return nil
		}
	}
	return errors.New("preset not found")
}

func presetName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("a preset needs a name")
	}
	if len([]rune(name)) > 32 {
		return "", errors.New("a preset name can be at most 32 characters")
	}
	return name, nil
}

// normalizeConfig validates a config and fills in defaults.
func (f *FanService) normalizeConfig(id string, c FanConfig) (FanConfig, error) {
	switch c.Mode {
	case "auto", "fixed", "curve":
	default:
		return c, fmt.Errorf("unknown fan mode %q", c.Mode)
	}
	c.Points = append([][2]int(nil), c.Points...)
	if c.Mode == "curve" && len(c.Points) < 2 {
		return c, errors.New("a curve needs at least two points")
	}
	sort.Slice(c.Points, func(i, j int) bool { return c.Points[i][0] < c.Points[j][0] })
	c.Fixed = min(max(c.Fixed, 0), 100)
	if c.Sensor == "" {
		c.Sensor = f.defaultSensor(id)
	}
	if len(c.Points) < 2 {
		c.Points = defaultFanPoints()
	}
	return c, nil
}

func (f *FanService) setConfigLocked(id string, c FanConfig) error {
	for _, d := range f.fans {
		if d.ID == id {
			d.Config = c
			f.cfg[id] = c
			return nil
		}
	}
	return fmt.Errorf("unknown fan %q", id)
}

// presetsLocked returns the built-in Auto preset followed by the user's own.
func (f *FanService) presetsLocked() []FanPreset {
	out := f.builtinPresetsLocked()
	for _, p := range f.presets {
		out = append(out, clonePreset(p))
	}
	return out
}

// builtinPresetsLocked builds the Auto preset for the connected fans.
func (f *FanService) builtinPresetsLocked() []FanPreset {
	p := FanPreset{ID: "auto", Name: "Auto", Builtin: true}
	for _, d := range f.fans {
		p.Configs = append(p.Configs, FanPresetEntry{
			FanID:  d.ID,
			Config: FanConfig{Mode: "auto"},
		})
	}
	return []FanPreset{p}
}

func (f *FanService) findPresetLocked(id string) (FanPreset, bool) {
	for _, p := range f.builtinPresetsLocked() {
		if p.ID == id {
			return p, true
		}
	}
	for _, p := range f.presets {
		if p.ID == id {
			return clonePreset(p), true
		}
	}
	return FanPreset{}, false
}

func (f *FanService) currentConfigsLocked() []FanPresetEntry {
	out := make([]FanPresetEntry, 0, len(f.fans))
	for _, d := range f.fans {
		out = append(out, FanPresetEntry{FanID: d.ID, Config: cloneConfig(d.Config)})
	}
	return out
}

func clonePreset(p FanPreset) FanPreset {
	p.Configs = append([]FanPresetEntry(nil), p.Configs...)
	for i := range p.Configs {
		p.Configs[i].Config = cloneConfig(p.Configs[i].Config)
	}
	return p
}

func cloneConfig(c FanConfig) FanConfig {
	c.Points = append([][2]int(nil), c.Points...)
	return c
}

// relaunchElevated is set by RestartAsAdmin and acted on after shutdown,
// once the single-instance lock is released. relaunchArgs carries a one-time
// setup flag for the elevated copy.
var (
	relaunchElevated bool
	relaunchArgs     []string
)

// RestartAsAdmin quits and relaunches the app elevated so motherboard fans
// can be controlled.
func (f *FanService) RestartAsAdmin() {
	relaunchElevated = true
	relaunchArgs = nil
	application.Get().Quit()
}

func relaunchAsAdmin(args ...string) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	verb, _ := windows.UTF16PtrFromString("runas")
	file, _ := windows.UTF16PtrFromString(exe)
	var params *uint16
	if len(args) > 0 {
		params, _ = windows.UTF16PtrFromString(strings.Join(args, " "))
	}
	windows.ShellExecute(0, verb, file, params, nil, windows.SW_SHOWNORMAL)
}

func (f *FanService) save() {
	if f.path == "" {
		return
	}
	os.MkdirAll(filepath.Dir(f.path), 0o755)
	if b, err := json.MarshalIndent(f.cfg, "", "  "); err == nil {
		os.WriteFile(f.path, b, 0o644)
	}
}

func (f *FanService) savePresets() {
	if f.presetPath == "" {
		return
	}
	os.MkdirAll(filepath.Dir(f.presetPath), 0o755)
	if b, err := json.MarshalIndent(f.presets, "", "  "); err == nil {
		os.WriteFile(f.presetPath, b, 0o644)
	}
}

func elevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

func boardName() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System\BIOS`, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	v, _, _ := k.GetStringValue("BaseBoardProduct")
	return v
}
