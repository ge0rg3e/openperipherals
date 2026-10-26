// Package nvgpu reads NVIDIA GPU temperature and fan speed and sets fan duty
// through NVAPI (nvapi64.dll, shipped with the driver). Function IDs and
// struct layouts follow LibreHardwareMonitor.
package nvgpu

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	idInitialize        = 0x0150e828
	idEnumPhysicalGPUs  = 0xe5ac921f
	idGetFullName       = 0xceee8e9f
	idGetThermal        = 0xe3640a56
	idGetTach           = 0x5f608315
	idGetCoolerSettings = 0xda141340
	idSetCoolerLevels   = 0x891fa0ae

	maxGPUs          = 64
	maxCoolers       = 20
	maxThermal       = 3
	thermalTargetAll = 15
	coolerTargetAll  = 7

	policyManual = 1
	policyAuto   = 32
)

type cooler struct {
	Type, Controller, DefaultMin, DefaultMax, CurrentMin, CurrentMax, CurrentLevel,
	DefaultPolicy, CurrentPolicy, Target, ControlType, Active int32
}

type coolerSettings struct {
	Version, Count uint32
	Cooler         [maxCoolers]cooler
}

type level struct {
	Level  int32
	Policy uint32
}

type coolerLevels struct {
	Version uint32
	Levels  [maxCoolers]level
}

type sensor struct {
	Controller, DefaultMin, DefaultMax, Current, Target int32
}

type thermalSettings struct {
	Version, Count uint32
	Sensor         [maxThermal]sensor
}

func version[T any](v uint32) uint32 {
	var t T
	return uint32(unsafe.Sizeof(t)) | v<<16
}

var (
	loadOnce sync.Once
	loadErr  error
	query    *windows.LazyProc
	fnsMu    sync.Mutex
	fns      = map[uint32]uintptr{}
)

func fn(id uint32) (uintptr, error) {
	fnsMu.Lock()
	defer fnsMu.Unlock()
	if p, ok := fns[id]; ok {
		return p, nil
	}
	p, _, _ := query.Call(uintptr(id))
	if p == 0 {
		return 0, fmt.Errorf("NVAPI function %#x unavailable", id)
	}
	fns[id] = p
	return p, nil
}

func call(id uint32, args ...uintptr) error {
	p, err := fn(id)
	if err != nil {
		return err
	}
	r, _, _ := syscall.SyscallN(p, args...)
	if int32(r) != 0 {
		return fmt.Errorf("NVAPI %#x failed: status %d", id, int32(r))
	}
	return nil
}

func load() error {
	loadOnce.Do(func() {
		dll := windows.NewLazySystemDLL("nvapi64.dll")
		if err := dll.Load(); err != nil {
			loadErr = errors.New("no NVIDIA driver")
			return
		}
		query = dll.NewProc("nvapi_QueryInterface")
		if err := query.Find(); err != nil {
			loadErr = err
			return
		}
		loadErr = call(idInitialize)
	})
	return loadErr
}

// GPU is one physical NVIDIA GPU.
type GPU struct {
	Name    string
	Coolers int
	MinDuty []int // per cooler, driver-enforced lower bound
	h       uintptr
	mu      sync.Mutex
	manual  []bool
}

// Enumerate returns all NVIDIA GPUs that expose fan control.
func Enumerate() ([]*GPU, error) {
	if err := load(); err != nil {
		return nil, err
	}
	var handles [maxGPUs]uintptr
	var n uint32
	if err := call(idEnumPhysicalGPUs, uintptr(unsafe.Pointer(&handles[0])), uintptr(unsafe.Pointer(&n))); err != nil {
		return nil, err
	}
	var out []*GPU
	for _, h := range handles[:n] {
		g := &GPU{h: h, Name: "NVIDIA GPU"}
		var name [64]byte
		if call(idGetFullName, h, uintptr(unsafe.Pointer(&name[0]))) == nil {
			g.Name = windows.ByteSliceToString(name[:])
		}
		if cs, err := g.coolerSettings(); err == nil {
			g.Coolers = int(cs.Count)
			for i := range g.Coolers {
				g.MinDuty = append(g.MinDuty, int(cs.Cooler[i].DefaultMin))
			}
			g.manual = make([]bool, g.Coolers)
		}
		out = append(out, g)
	}
	return out, nil
}

func (g *GPU) coolerSettings() (*coolerSettings, error) {
	cs := &coolerSettings{Version: version[coolerSettings](2)}
	err := call(idGetCoolerSettings, g.h, coolerTargetAll, uintptr(unsafe.Pointer(cs)))
	return cs, err
}

// Temperature returns the GPU core temperature in °C.
func (g *GPU) Temperature() (int, error) {
	ts := thermalSettings{Version: version[thermalSettings](2), Count: maxThermal}
	if err := call(idGetThermal, g.h, thermalTargetAll, uintptr(unsafe.Pointer(&ts))); err != nil {
		return 0, err
	}
	if ts.Count == 0 {
		return 0, errors.New("no GPU thermal sensor")
	}
	return int(ts.Sensor[0].Current), nil
}

// RPM returns the fan tachometer reading.
func (g *GPU) RPM() (int, error) {
	var rpm uint32
	err := call(idGetTach, g.h, uintptr(unsafe.Pointer(&rpm)))
	return int(rpm), err
}

// Duty returns each cooler's current level (0-100 %).
func (g *GPU) Duty() ([]int, error) {
	cs, err := g.coolerSettings()
	if err != nil {
		return nil, err
	}
	out := make([]int, cs.Count)
	for i := range out {
		out[i] = int(cs.Cooler[i].CurrentLevel)
	}
	return out, nil
}

// SetDuty puts a cooler in manual mode at the given duty.
func (g *GPU) SetDuty(i, percent int) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if i < 0 || i >= g.Coolers {
		return fmt.Errorf("GPU cooler %d out of range", i)
	}
	lv := coolerLevels{Version: version[coolerLevels](1)}
	lv.Levels[0] = level{Level: int32(max(min(percent, 100), g.MinDuty[i])), Policy: policyManual}
	if err := call(idSetCoolerLevels, g.h, uintptr(i), uintptr(unsafe.Pointer(&lv))); err != nil {
		return err
	}
	g.manual[i] = true
	return nil
}

// Restore hands a cooler back to the driver's automatic curve.
func (g *GPU) Restore(i int) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if i < 0 || i >= g.Coolers || !g.manual[i] {
		return nil
	}
	lv := coolerLevels{Version: version[coolerLevels](1)}
	lv.Levels[0].Policy = policyAuto
	if err := call(idSetCoolerLevels, g.h, uintptr(i), uintptr(unsafe.Pointer(&lv))); err != nil {
		return err
	}
	g.manual[i] = false
	return nil
}

// Close restores automatic control on every cooler.
func (g *GPU) Close() {
	for i := range g.Coolers {
		g.Restore(i)
	}
}
