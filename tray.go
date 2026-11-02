package main

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"openperipherals/internal/sysstats"
)

const trayRefresh = 3 * time.Second

// Read hardware away from the UI thread, and rebuild on each opening so device
// capabilities, saved presets and checkmarks follow changes made in the app.
func setupTrayMenu(app *application.App, tray *application.SystemTray, fans *FanService, devices *PeripheralService, wallpapers *WallpaperService, updatesEnabled bool, show func()) {
	var opening sync.Mutex
	var previous *application.Menu
	open := func() {
		if !opening.TryLock() {
			return
		}
		go func() {
			st, fs, sy := devices.Status(), fans.Status(), devices.Sync()
			application.InvokeAsync(func() {
				defer opening.Unlock()
				menu := quickSettingsMenu(app, fans, devices, wallpapers, st, fs, sy, updatesEnabled, show)
				if previous != nil {
					previous.Destroy()
				}
				tray.SetMenu(menu)
				previous = menu
				tray.OpenMenu()
			})
		}()
	}
	tray.OnClick(open)
	tray.OnRightClick(open)
	tray.OnDoubleClick(show)
}

func quickSettingsMenu(app *application.App, fans *FanService, devices *PeripheralService, wallpapers *WallpaperService, st Status, fs FanStatus, sy SyncLighting, updatesEnabled bool, show func()) *application.Menu {
	menu := application.NewMenu()
	menu.Add("Open OpenPeripherals").OnClick(func(*application.Context) { show() })
	menu.AddSeparator()
	// Wails runs menu callbacks in a goroutine, keeping device writes off the UI
	// thread. Report errors in a native dialog even when the app window is hidden.
	action := func(fn func() error) func(*application.Context) {
		return func(*application.Context) {
			if err := fn(); err != nil {
				app.Dialog.Error().SetTitle("Quick settings").SetMessage(err.Error()).Show()
			}
			app.Event.Emit("tray-settings", devices.Status())
		}
	}
	if st.Keyboard != nil || st.Mouse != nil {
		lighting := menu.AddSubmenu("Lighting sync")
		lighting.AddCheckbox("Enabled", sy.Enabled).OnClick(action(func() error {
			next := devices.Sync()
			next.Enabled = !next.Enabled
			return devices.SetSync(next)
		}))
		lighting.AddSeparator()
		for _, effect := range []string{"static", "spectrum", "wave", "breathing", "off"} {
			lighting.AddRadio(effectNames[effect], sy.Enabled && sy.Effect == effect).OnClick(action(func() error {
				next := devices.Sync()
				next.Enabled, next.Effect = true, effect
				return devices.SetSync(next)
			}))
		}
	}
	brightness := func(parent *application.Menu, current int, set func(int) error) {
		for _, percent := range []int{0, 25, 50, 75, 100} {
			parent.AddRadio(fmt.Sprintf("%d%%", percent), current == percent).OnClick(action(func() error { return set(percent) }))
		}
	}
	if k := st.Keyboard; k != nil {
		keyboard := menu.AddSubmenu("Keyboard")
		if k.HasBrightness {
			brightness(keyboard.AddSubmenu("Brightness"), k.Brightness, devices.SetKeyboardBrightness)
		}
		if k.GameMode != nil {
			keyboard.AddCheckbox("Game mode", *k.GameMode).OnClick(action(func() error {
				return devices.SetKeyboardOption("gameMode", !*k.GameMode)
			}))
		}
		effects := keyboard.AddSubmenu("Lighting effect")
		for _, effect := range k.Effects {
			label := effectNames[effect]
			if label == "" {
				label = effect
			}
			effects.AddRadio(label, k.Lighting.Effect == effect).OnClick(action(func() error {
				l := k.Lighting
				l.Effect = effect
				return devices.SetKeyboardLighting(l)
			}))
		}
	}
	if m := st.Mouse; m != nil {
		mouse := menu.AddSubmenu("Mouse")
		dpi := mouse.AddSubmenu("DPI")
		if m.Kind == "corsair" {
			brightness(mouse.AddSubmenu("Brightness"), m.Brightness, devices.SetMouseBrightness)
			for i, stage := range m.Stages {
				if stage.Enabled {
					dpi.AddRadio(fmt.Sprintf("Stage %d: %d DPI", i+1, stage.DPI), m.Current == i).OnClick(action(func() error {
						return devices.SetMouseDPI(m.Stages, i)
					}))
				}
			}
		} else {
			// Include the current value even when it is not one of the shortcuts.
			values := []int{400, 800, 1600, 3200, 6400}
			if !slices.Contains(values, m.DPI) && m.DPI > 0 {
				values = append(values, m.DPI)
				slices.Sort(values)
			}
			for _, value := range values {
				if value < m.MinDPI || value > m.MaxDPI || (m.DPIStep > 0 && value%m.DPIStep != 0) {
					continue
				}
				dpi.AddRadio(fmt.Sprintf("%d DPI", value), m.DPI == value).OnClick(action(func() error {
					return devices.SetMouseOption("dpi", value)
				}))
			}
			if len(m.PollingRates) > 0 {
				polling := mouse.AddSubmenu("Polling rate")
				for _, rate := range m.PollingRates {
					polling.AddRadio(fmt.Sprintf("%d Hz", rate), m.Polling == rate).OnClick(action(func() error {
						return devices.SetMouseOption("polling", rate)
					}))
				}
			}
		}
	}
	if len(fs.Fans) > 0 {
		presets := menu.AddSubmenu("Fan preset")
		for _, preset := range fs.Presets {
			presets.Add(preset.Name).OnClick(action(func() error { return fans.ApplyPreset(preset.ID) }))
		}
	}
	if st.Keyboard == nil && st.Mouse == nil && len(fs.Fans) == 0 {
		menu.Add("No controllable devices connected").SetEnabled(false)
	}
	menu.AddSeparator()
	wp := wallpapers.Status()
	wallpaperMenu := menu.AddSubmenu("Live wallpaper")
	wallpaperMenu.Add("Status: " + wp.Phase).SetEnabled(false)
	if wp.Phase == "playing" || wp.Phase == "paused" || wp.Phase == "loading" || wp.Phase == "resuming" {
		label := "Pause"
		if wp.ManualPause {
			label = "Resume"
		}
		wallpaperMenu.Add(label).OnClick(action(func() error { _, err := wallpapers.SetPaused(!wp.ManualPause); return err }))
		wallpaperMenu.Add("Stop").OnClick(action(func() error { _, err := wallpapers.Stop(); return err }))
	} else {
		wallpaperMenu.Add("Start saved wallpaper").OnClick(action(func() error { _, err := wallpapers.Apply(wp.Config); return err }))
	}
	menu.AddSeparator()
	if updatesEnabled {
		addUpdaterMenu(menu, app)
		menu.AddSeparator()
	}
	menu.Add("Quit").OnClick(func(*application.Context) { app.Quit() })
	return menu
}

// runTrayTooltip keeps the tray tooltip showing a live system summary until
// stop is closed.
func runTrayTooltip(tray *application.SystemTray, fans *FanService, devices *PeripheralService, stop <-chan struct{}) {
	var sampler sysstats.Sampler
	sampler.Sample() // prime CPU usage
	t := time.NewTicker(trayRefresh)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
		}
		text := traySummary(sampler.Sample(), fans.Status(), devices.lightingLabel())
		application.InvokeAsync(func() { tray.SetTooltip(text) })
	}
}

// traySummary formats the most useful readings in as few lines as fit the
// Windows tooltip limit (127 characters).
func traySummary(st sysstats.Stats, fs FanStatus, lighting string) string {
	temp := func(match func(TempSensor) bool) string {
		for _, s := range fs.Sensors {
			if match(s) && s.Temp >= 0 {
				return fmt.Sprintf(" · %d °C", s.Temp)
			}
		}
		return ""
	}
	gb := func(b uint64) float64 { return float64(b) / (1 << 30) }

	lines := []string{"OpenPeripherals"}
	if st.CPU >= 0 {
		lines = append(lines, fmt.Sprintf("CPU %d%%%s", st.CPU, temp(func(s TempSensor) bool { return s.Name == "CPU" })))
	}
	for i, g := range st.GPUs {
		id := fmt.Sprintf("gpu%d", i)
		lines = append(lines, fmt.Sprintf("GPU %d%%%s", g.Usage, temp(func(s TempSensor) bool { return s.ID == id })))
	}
	if st.RAMTotal > 0 {
		lines = append(lines, fmt.Sprintf("RAM %.1f / %.1f GB", gb(st.RAMUsed), gb(st.RAMTotal)))
	}
	if lighting != "" {
		lines = append(lines, "Lighting: "+lighting)
	}

	// Drop trailing lines rather than letting Windows cut one mid-way.
	for len(lines) > 1 && len([]rune(strings.Join(lines, "\n"))) > 127 {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

var effectNames = map[string]string{
	"static": "Static", "spectrum": "Spectrum", "wave": "Wave", "wheel": "Wheel", "breathing": "Breathing",
	"reactive": "Reactive", "starlight": "Starlight", "off": "Off",
}

// lightingLabel describes the current lighting for the tray: the sync
// effect when synced, otherwise the keyboard's effect.
func (s *PeripheralService) lightingLabel() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sy := s.cfg.Sync; sy.Enabled {
		label := "Sync · " + effectNames[sy.Effect]
		if sy.Effect == "static" || sy.Effect == "breathing" {
			label += " " + strings.ToUpper(sy.Color)
		}
		return label
	}
	if s.kb != nil {
		return "Keyboard · " + effectNames[s.cfg.Keyboard.Effect]
	}
	return ""
}
