// Package pawnio talks to the PawnIO kernel driver (https://pawnio.eu), which
// runs small signed modules that give user mode controlled hardware access.
// Opening it requires administrator rights.
package pawnio

import (
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

//go:embed modules/LpcIO.bin
var LpcIO []byte

const (
	deviceType    = 41394 << 16
	ioctlLoad     = deviceType | 0x821<<2
	ioctlExecute  = deviceType | 0x841<<2
	fnNameLength  = 32
	devicePath    = `\\?\GLOBALROOT\Device\PawnIO`
	uninstallKey  = `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\PawnIO`
	isaBusMutex   = `Global\Access_ISABUS.HTP.Method`
	mutexWaitMsec = 100
)

// ErrNotInstalled is returned when the PawnIO driver is not present.
var ErrNotInstalled = errors.New("PawnIO driver is not installed")

// Module is one loaded PawnIO module bound to its own driver handle.
type Module struct {
	h windows.Handle
}

// Load opens the driver and loads a signed module blob.
func Load(blob []byte) (*Module, error) {
	p, _ := windows.UTF16PtrFromString(devicePath)
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		switch err {
		case windows.ERROR_FILE_NOT_FOUND, windows.ERROR_PATH_NOT_FOUND:
			return nil, ErrNotInstalled
		case windows.ERROR_ACCESS_DENIED:
			return nil, fmt.Errorf("PawnIO needs administrator rights: %w", err)
		}
		return nil, fmt.Errorf("open PawnIO: %w", err)
	}
	var n uint32
	if err := windows.DeviceIoControl(h, ioctlLoad, &blob[0], uint32(len(blob)), nil, 0, &n, nil); err != nil {
		windows.CloseHandle(h)
		return nil, fmt.Errorf("load PawnIO module: %w", err)
	}
	return &Module{h: h}, nil
}

// Close unloads the module.
func (m *Module) Close() error { return windows.CloseHandle(m.h) }

// Execute calls a module function with exactly the argument and result
// counts it declares.
func (m *Module) Execute(fn string, in []uint64, outLen int) ([]uint64, error) {
	buf := make([]byte, fnNameLength+8*len(in))
	copy(buf[:fnNameLength-1], fn)
	for i, v := range in {
		binary.LittleEndian.PutUint64(buf[fnNameLength+8*i:], v)
	}
	out := make([]uint64, outLen)
	var outPtr *byte
	if outLen > 0 {
		outPtr = (*byte)(unsafe.Pointer(&out[0]))
	}
	var n uint32
	if err := windows.DeviceIoControl(m.h, ioctlExecute, &buf[0], uint32(len(buf)), outPtr, uint32(8*outLen), &n, nil); err != nil {
		return nil, fmt.Errorf("%s: %w", fn, err)
	}
	return out, nil
}

// Installed reports whether the PawnIO driver package is installed.
func Installed() bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, uninstallKey, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return false
	}
	k.Close()
	return true
}

// IsaBus is the system-wide mutex that monitoring tools (LibreHardwareMonitor,
// HWiNFO, FanControl, ...) hold while touching Super I/O ports.
type IsaBus struct{ h windows.Handle }

// OpenIsaBus opens or creates the shared ISA bus mutex.
func OpenIsaBus() (*IsaBus, error) {
	p, _ := windows.UTF16PtrFromString(isaBusMutex)
	h, err := windows.CreateMutex(nil, false, p)
	if h == 0 {
		return nil, fmt.Errorf("ISA bus mutex: %w", err)
	}
	return &IsaBus{h: h}, nil
}

// Lock waits for exclusive bus access. Windows mutexes are owned by an OS
// thread, so the calling goroutine stays on its thread until Unlock.
func (b *IsaBus) Lock() error {
	runtime.LockOSThread()
	ev, err := windows.WaitForSingleObject(b.h, mutexWaitMsec)
	switch ev {
	case windows.WAIT_OBJECT_0, windows.WAIT_ABANDONED:
		return nil
	case uint32(windows.WAIT_TIMEOUT):
		err = errors.New("ISA bus busy")
	}
	runtime.UnlockOSThread()
	return err
}

// Unlock releases the bus. It must be called from the goroutine that locked it.
func (b *IsaBus) Unlock() {
	windows.ReleaseMutex(b.h)
	runtime.UnlockOSThread()
}

// Close releases the mutex handle.
func (b *IsaBus) Close() { windows.CloseHandle(b.h) }
