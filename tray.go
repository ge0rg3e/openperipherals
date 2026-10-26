package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"openperipherals/internal/sysstats"
)

const trayRefresh = 3 * time.Second

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
