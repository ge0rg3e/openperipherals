package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"openperipherals/internal/nvgpu"
	"openperipherals/internal/pawnio"
	"openperipherals/internal/superio"
)

type fanHardware struct {
	chip *superio.Chip
	gpus []*nvgpu.GPU
}

func (f *FanService) closeHardware() {
	if f.hardware.chip != nil {
		f.hardware.chip.Close()
	}
	for _, g := range f.hardware.gpus {
		g.Close()
	}
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
		f.hardware.chip = chip
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
		f.hardware.gpus = gpus
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

}
func (f *FanService) readHardware() map[string]int {
	temps := map[string]int{}
	var mbRPM, mbDuty []int
	if f.hardware.chip != nil {
		if r, err := f.hardware.chip.Read(); err == nil {
			for i, t := range r.Temps {
				temps[fmt.Sprintf("mb:t%d", i)] = t
			}
			mbRPM, mbDuty = r.FanRPM, r.Duty
		}
	}
	gpuDuty := map[string][]int{}
	for gi, g := range f.hardware.gpus {
		id := fmt.Sprintf("gpu%d", gi)
		if t, err := g.Temperature(); err == nil {
			temps[id] = t
		} else {
			temps[id] = -1
		}
		gpuDuty[id], _ = g.Duty()
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

	}
	return temps
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
