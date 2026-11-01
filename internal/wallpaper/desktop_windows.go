package wallpaper

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/w32"
	"golang.org/x/sys/windows"
)

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	findWindowEx     = user32.NewProc("FindWindowExW")
	setParent        = user32.NewProc("SetParent")
	sendTimeout      = user32.NewProc("SendMessageTimeoutW")
	openInputDesktop = user32.NewProc("OpenInputDesktop")
	closeDesktop     = user32.NewProc("CloseDesktop")
	getObjectInfo    = user32.NewProc("GetUserObjectInformationW")
	getPowerStatus   = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetSystemPowerStatus")
	getFrameBounds   = windows.NewLazySystemDLL("dwmapi.dll").NewProc("DwmGetWindowAttribute")
)

func findChild(parent, after w32.HWND, class string) w32.HWND {
	p, _ := windows.UTF16PtrFromString(class)
	h, _, _ := findWindowEx.Call(uintptr(parent), uintptr(after), uintptr(unsafe.Pointer(p)), 0)
	return w32.HWND(h)
}

// RefreshDesktop asks Explorer to repaint its configured wallpaper after our
// cross-process child surface disappears. Do not set a wallpaper file here:
// that would replace slideshows, Spotlight, or per-monitor wallpaper choices.
// Run this after the host and its WebView descendants have exited, including
// forced termination where the host cannot run its own cleanup.
func RefreshDesktop() {
	// Invalidating WorkerW only repaints its cached surface, which may still
	// contain our renderer's last clear colour. Reload Explorer's background
	// through the wallpaper API first. A nil path reloads the configured
	// background; do not supply a filename or write the user's profile.
	user32.NewProc("SystemParametersInfoW").Call(0x0014, 0, 0, 0) // SPI_SETDESKWALLPAPER
	w32.EnumWindows(refreshDesktopCallback, 0)
}

// Reuse the callback; native Go callbacks cannot be freed after registration.
var refreshDesktopCallback = syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
	window := w32.HWND(hwnd)
	switch w32.GetClassName(window) {
	case "Progman", "WorkerW":
		w32.RedrawWindow(window, nil, 0, w32.RDW_INVALIDATE|w32.RDW_ERASE|w32.RDW_ALLCHILDREN|w32.RDW_FRAME)
	}
	return 1
})

// Explorer's wallpaper host is undocumented. Handle both the classic sibling
// WorkerW and the newer WorkerW nested under Progman; fail instead of covering icons.
func desktopParent() (w32.HWND, error) {
	class, _ := windows.UTF16PtrFromString("Progman")
	progman := w32.FindWindowW(class, nil)
	if progman == 0 {
		return 0, fmt.Errorf("Windows Explorer desktop is unavailable")
	}
	var result uintptr
	for _, param := range [][2]uintptr{{0xd, 0}, {0xd, 1}, {0, 0}} {
		sendTimeout.Call(uintptr(progman), 0x052c, param[0], param[1], 2, 1000, uintptr(unsafe.Pointer(&result)))
	}
	if worker := findChild(progman, 0, "WorkerW"); worker != 0 && findChild(worker, 0, "SHELLDLL_DefView") == 0 {
		return worker, nil
	}
	var worker w32.HWND
	callback := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		if findChild(w32.HWND(hwnd), 0, "SHELLDLL_DefView") != 0 {
			worker = findChild(0, w32.HWND(hwnd), "WorkerW")
		}
		if worker != 0 {
			return 0
		}
		return 1
	})
	w32.EnumWindows(callback, 0)
	if worker == 0 {
		return 0, fmt.Errorf("Explorer did not provide a wallpaper layer; restart Explorer and try again")
	}
	return worker, nil
}

func targetBounds(span bool) w32.RECT {
	if span {
		x, y := w32.GetSystemMetrics(76), w32.GetSystemMetrics(77)
		return w32.RECT{Left: int32(x), Top: int32(y), Right: int32(x + w32.GetSystemMetrics(78)), Bottom: int32(y + w32.GetSystemMetrics(79))}
	}
	var info w32.MONITORINFO
	info.CbSize = uint32(unsafe.Sizeof(info))
	w32.GetMonitorInfo(w32.MonitorFromPoint(0, 0, 1), &info)
	return info.RcMonitor
}

func attachDesktop(hwnd w32.HWND, parent w32.HWND, bounds w32.RECT) error {
	// The window is created with WS_CHILD, so SetParent needn't change its style.
	setParent.Call(uintptr(hwnd), uintptr(parent))
	actual, _, _ := user32.NewProc("GetParent").Call(uintptr(hwnd))
	if actual != uintptr(parent) {
		return fmt.Errorf("could not attach the wallpaper to Explorer")
	}
	x, y, ok := w32.ScreenToClient(parent, int(bounds.Left), int(bounds.Top))
	if !ok || !w32.SetWindowPos(hwnd, 0, x, y, int(bounds.Right-bounds.Left), int(bounds.Bottom-bounds.Top), w32.SWP_NOACTIVATE) {
		return fmt.Errorf("could not position the wallpaper")
	}
	return nil
}

func desktopLocked() bool {
	desktop, _, _ := openInputDesktop.Call(0, 0, 1) // DESKTOP_READOBJECTS
	if desktop == 0 {
		return true
	}
	defer closeDesktop.Call(desktop)
	var name [64]uint16
	var needed uint32
	ok, _, _ := getObjectInfo.Call(desktop, 2, uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Sizeof(name)), uintptr(unsafe.Pointer(&needed)))
	return ok == 0 || windows.UTF16ToString(name[:]) != "Default"
}

func onBattery() bool {
	var status struct {
		AC, Battery, Percent, Reserved byte
		Life, FullLife                 uint32
	}
	ok, _, _ := getPowerStatus.Call(uintptr(unsafe.Pointer(&status)))
	return ok != 0 && status.AC == 0
}

func foregroundCovers(bounds w32.RECT, span bool) bool {
	hwnd := w32.GetForegroundWindow()
	class := w32.GetClassName(hwnd)
	if hwnd == 0 || class == "Progman" || class == "WorkerW" || class == "Shell_TrayWnd" {
		return false
	}
	var info w32.MONITORINFO
	info.CbSize = uint32(unsafe.Sizeof(info))
	if !w32.GetMonitorInfo(w32.MonitorFromWindow(hwnd, 2), &info) {
		return false
	}
	if !span && (info.RcMonitor.Left != bounds.Left || info.RcMonitor.Top != bounds.Top) {
		return false
	}
	r := w32.GetWindowRect(hwnd)
	if r == nil {
		return false
	}
	frame := *r
	getFrameBounds.Call(uintptr(hwnd), 9, uintptr(unsafe.Pointer(&frame)), unsafe.Sizeof(frame))
	return Covers([4]int32{frame.Left, frame.Top, frame.Right, frame.Bottom}, [4]int32{info.RcWork.Left, info.RcWork.Top, info.RcWork.Right, info.RcWork.Bottom})
}
