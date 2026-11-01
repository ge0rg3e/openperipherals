package wallpaper

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The edge package does not wrap suspension or popup cancellation. These slots
// follow the Microsoft WebView2 COM ABI, including all inherited interface slots.
// Do not use the generated pkg/webview2 extension vtables, which omit inheritance.
//
//go:uintptrescapes
func comCall(object unsafe.Pointer, slot int, args ...uintptr) error {
	vtbl := *(*unsafe.Pointer)(object)
	proc := *(*uintptr)(unsafe.Add(vtbl, uintptr(slot)*unsafe.Sizeof(uintptr(0))))
	args = append([]uintptr{uintptr(object)}, args...)
	hr, _, _ := syscall.SyscallN(proc, args...)
	if int32(hr) < 0 {
		return fmt.Errorf("WebView2: 0x%08x", uint32(hr))
	}
	return nil
}

type comHandler struct {
	vtbl   *[4]uintptr
	invoke func(uintptr, uintptr)
	iid    windows.GUID
	pin    runtime.Pinner
}

var handlerVtbl = [4]uintptr{
	syscall.NewCallback(func(this, iid, out uintptr) uintptr {
		h := (*comHandler)(unsafe.Pointer(this))
		guid := *(*windows.GUID)(unsafe.Pointer(iid))
		if guid != h.iid && guid != (windows.GUID{Data4: [8]byte{0xc0, 0, 0, 0, 0, 0, 0, 0x46}}) {
			*(*uintptr)(unsafe.Pointer(out)) = 0
			return 0x80004002
		}
		*(*uintptr)(unsafe.Pointer(out)) = this
		return 0
	}),
	syscall.NewCallback(func(uintptr) uintptr { return 2 }),
	syscall.NewCallback(func(uintptr) uintptr { return 1 }),
	syscall.NewCallback(func(this, a, b uintptr) uintptr { (*comHandler)(unsafe.Pointer(this)).invoke(a, b); return 0 }),
}

func newHandler(iid string, invoke func(uintptr, uintptr)) *comHandler {
	guid, _ := windows.GUIDFromString(iid)
	h := &comHandler{vtbl: &handlerVtbl, invoke: invoke, iid: guid}
	h.pin.Pin(h)
	return h
}

func comQuery(object unsafe.Pointer, iid string) (unsafe.Pointer, error) {
	guid, err := windows.GUIDFromString(iid)
	if err != nil {
		return nil, err
	}
	var result unsafe.Pointer
	err = comCall(object, 0, uintptr(unsafe.Pointer(&guid)), uintptr(unsafe.Pointer(&result)))
	return result, err
}
