// Package sysstats samples CPU, memory and NVIDIA GPU usage. CPU and RAM come
// from kernel32; GPU utilization and VRAM from NVML (nvml.dll, installed with
// the NVIDIA driver).
package sysstats

import (
	"errors"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32                 = windows.NewLazySystemDLL("kernel32.dll")
	procGetSystemTimes       = kernel32.NewProc("GetSystemTimes")
	procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
)

// GPU is one GPU's load.
type GPU struct {
	Name      string `json:"name"`
	Usage     int    `json:"usage"` // %
	VRAMUsed  uint64 `json:"vramUsed"`
	VRAMTotal uint64 `json:"vramTotal"`
}

// Stats is one sample.
type Stats struct {
	CPU      int    `json:"cpu"` // %, -1 on the first sample
	RAMUsed  uint64 `json:"ramUsed"`
	RAMTotal uint64 `json:"ramTotal"`
	GPUs     []GPU  `json:"gpus"`
}

// Sampler keeps the previous CPU times so usage covers the interval between
// two calls.
type Sampler struct {
	mu                 sync.Mutex
	idle, kernel, user uint64
	primed             bool
	nvmlOnce           sync.Once
	nvmlDevices        []uintptr
	nvmlNames          []string
}

// Sample returns current usage.
func (s *Sampler) Sample() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := Stats{CPU: s.cpu()}
	st.RAMUsed, st.RAMTotal = memory()
	s.nvmlOnce.Do(s.initNVML)
	for i, d := range s.nvmlDevices {
		if g, err := nvmlSample(d); err == nil {
			g.Name = s.nvmlNames[i]
			st.GPUs = append(st.GPUs, g)
		}
	}
	return st
}

func (s *Sampler) cpu() int {
	var idle, kernel, user windows.Filetime
	if r, _, _ := procGetSystemTimes.Call(uintptr(unsafe.Pointer(&idle)), uintptr(unsafe.Pointer(&kernel)), uintptr(unsafe.Pointer(&user))); r == 0 {
		return -1
	}
	i, k, u := ft(idle), ft(kernel), ft(user)
	defer func() { s.idle, s.kernel, s.user, s.primed = i, k, u, true }()
	if !s.primed {
		return -1
	}
	// Kernel time includes idle time.
	total := (k - s.kernel) + (u - s.user)
	if total == 0 {
		return 0
	}
	busy := total - (i - s.idle)
	return int((busy*100 + total/2) / total)
}

func ft(f windows.Filetime) uint64 { return uint64(f.HighDateTime)<<32 | uint64(f.LowDateTime) }

type memoryStatusEx struct {
	Length, MemoryLoad                                 uint32
	TotalPhys, AvailPhys, TotalPageFile, AvailPageFile uint64
	TotalVirtual, AvailVirtual, AvailExtendedVirtual   uint64
}

func memory() (used, total uint64) {
	m := memoryStatusEx{Length: uint32(unsafe.Sizeof(memoryStatusEx{}))}
	if r, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&m))); r == 0 {
		return 0, 0
	}
	return m.TotalPhys - m.AvailPhys, m.TotalPhys
}

// NVML

var (
	nvml              = windows.NewLazySystemDLL("nvml.dll")
	nvmlInit          = nvml.NewProc("nvmlInit_v2")
	nvmlDeviceCount   = nvml.NewProc("nvmlDeviceGetCount_v2")
	nvmlDeviceByIndex = nvml.NewProc("nvmlDeviceGetHandleByIndex_v2")
	nvmlDeviceName    = nvml.NewProc("nvmlDeviceGetName")
	nvmlUtilization   = nvml.NewProc("nvmlDeviceGetUtilizationRates")
	nvmlMemoryInfo    = nvml.NewProc("nvmlDeviceGetMemoryInfo")
	errNVML           = errors.New("nvml call failed")
)

func (s *Sampler) initNVML() {
	if nvml.Load() != nil {
		return
	}
	if r, _, _ := nvmlInit.Call(); r != 0 {
		return
	}
	var n uint32
	if r, _, _ := nvmlDeviceCount.Call(uintptr(unsafe.Pointer(&n))); r != 0 {
		return
	}
	for i := range n {
		var d uintptr
		if r, _, _ := nvmlDeviceByIndex.Call(uintptr(i), uintptr(unsafe.Pointer(&d))); r != 0 {
			continue
		}
		var name [96]byte
		nvmlDeviceName.Call(d, uintptr(unsafe.Pointer(&name[0])), uintptr(len(name)))
		s.nvmlDevices = append(s.nvmlDevices, d)
		s.nvmlNames = append(s.nvmlNames, windows.ByteSliceToString(name[:]))
	}
}

func nvmlSample(d uintptr) (GPU, error) {
	var util struct{ GPU, Memory uint32 }
	if r, _, _ := nvmlUtilization.Call(d, uintptr(unsafe.Pointer(&util))); r != 0 {
		return GPU{}, errNVML
	}
	var mem struct{ Total, Free, Used uint64 }
	if r, _, _ := nvmlMemoryInfo.Call(d, uintptr(unsafe.Pointer(&mem))); r != 0 {
		return GPU{}, errNVML
	}
	return GPU{Usage: int(util.GPU), VRAMUsed: mem.Used, VRAMTotal: mem.Total}, nil
}
