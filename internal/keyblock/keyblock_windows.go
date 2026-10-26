// Package keyblock swallows selected keys with a low-level keyboard hook
// (WH_KEYBOARD_LL). The hook only exists while at least one key is blocked,
// so it costs nothing when unused.
//
// The hook can't tell devices apart. In device-only mode a key is blocked
// only when a device watcher has claimed the press (see Claim), so the same
// key from another device passes through.
package keyblock

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32 = windows.NewLazySystemDLL("user32.dll")

	procSetWindowsHookEx   = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHook  = user32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx     = user32.NewProc("CallNextHookEx")
	procGetMessage         = user32.NewProc("GetMessageW")
	procPostThreadMessage  = user32.NewProc("PostThreadMessageW")
	procPeekMessage        = user32.NewProc("PeekMessageW")
	procGetCurrentThreadID = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetCurrentThreadId")
)

const (
	whKeyboardLL = 13
	wmQuit       = 0x0012
	wmKeyUp      = 0x0101
	wmSysKeyUp   = 0x0105
)

type kbdllHookStruct struct {
	VkCode    uint32
	ScanCode  uint32
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      [2]int32
	Private uint32
}

// Key is a virtual-key code (low byte) plus optional flags that narrow it to
// the extended or non-extended variant, e.g. numpad Enter vs Enter.
type Key uint16

const (
	ExtendedOnly Key = 0x100
	PlainOnly    Key = 0x200
)

const llkhfExtended = 0x01

// The hook callback can't carry state, so the blocked set is package-level:
// per virtual key, bit 0 blocks the plain variant and bit 1 the extended one.
var blocked [256]atomic.Uint32

var hookProc = windows.NewCallback(func(code int, wParam, lParam uintptr) uintptr {
	if code == 0 {
		kb := (*kbdllHookStruct)(unsafe.Pointer(lParam))
		up := wParam == wmKeyUp || wParam == wmSysKeyUp
		bit := uint32(1)
		if kb.Flags&llkhfExtended != 0 {
			bit = 2
		}
		if kb.VkCode < 256 && blocked[kb.VkCode].Load()&bit != 0 && shouldBlock(uint8(kb.VkCode), up) {
			return 1
		}
	}
	r, _, _ := procCallNextHookEx.Call(0, uintptr(code), wParam, lParam)
	return r
})

var (
	mu       sync.Mutex
	threadID uint32 // hook thread, started on first use and kept idle
	hooked   bool
)

const (
	wmHook   = 0x8000 + 1 // WM_APP+1: install the hook
	wmUnhook = 0x8000 + 2 // WM_APP+2: remove the hook
)

// Set replaces the blocked key set, installing or removing the hook as
// needed.
func Set(keys []Key) error {
	mu.Lock()
	defer mu.Unlock()
	var next [256]uint32
	for _, k := range keys {
		switch {
		case k&ExtendedOnly != 0:
			next[uint8(k)] |= 2
		case k&PlainOnly != 0:
			next[uint8(k)] |= 1
		default:
			next[uint8(k)] |= 3
		}
	}
	for i := range blocked {
		blocked[i].Store(next[i])
	}
	want := len(keys) > 0
	if want == hooked {
		return nil
	}
	if threadID == 0 {
		if !want {
			return nil
		}
		if err := startThread(); err != nil {
			return err
		}
	}
	return command(want)
}

var results = make(chan error)

// command asks the hook thread to install or remove the hook. Doing both on
// that thread means it keeps pumping messages afterwards, so a key event that
// was already in flight is still delivered rather than dropped.
func command(install bool) error {
	m := uintptr(wmUnhook)
	if install {
		m = wmHook
	}
	procPostThreadMessage.Call(uintptr(threadID), m, 0, 0)
	if err := <-results; err != nil {
		return err
	}
	hooked = install
	return nil
}

func startThread() error {
	ready := make(chan uint32)
	go func() {
		runtime.LockOSThread()
		var m msg
		// Create the thread's message queue before anyone posts to it.
		procPeekMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, 0)
		tid, _, _ := procGetCurrentThreadID.Call()
		ready <- uint32(tid)
		var hook uintptr
		for {
			r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			if r == 0 || int32(r) == -1 {
				return
			}
			switch m.Message {
			case wmHook:
				h, _, err := procSetWindowsHookEx.Call(whKeyboardLL, hookProc, 0, 0)
				if h == 0 {
					results <- err
					continue
				}
				hook = h
				results <- nil
			case wmUnhook:
				procUnhookWindowsHook.Call(hook)
				hook = 0
				results <- nil
			}
		}
	}()
	threadID = <-ready
	return nil
}
