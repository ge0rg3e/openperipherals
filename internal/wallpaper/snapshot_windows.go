package wallpaper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/w32"
	"golang.org/x/sys/windows"
)

type snapshot struct {
	width, height int32
	pixels        []byte
}

var (
	streamDLL          = windows.NewLazySystemDLL("ole32.dll")
	createMemoryStream = streamDLL.NewProc("CreateStreamOnHGlobal")
	getStreamMemory    = streamDLL.NewProc("GetHGlobalFromStream")
	memoryDLL          = windows.NewLazySystemDLL("kernel32.dll")
	globalSize         = memoryDLL.NewProc("GlobalSize")
	globalLock         = memoryDLL.NewProc("GlobalLock")
	globalUnlock       = memoryDLL.NewProc("GlobalUnlock")
	stretchPixels      = windows.NewLazySystemDLL("gdi32.dll").NewProc("StretchDIBits")
	setStretchMode     = windows.NewLazySystemDLL("gdi32.dll").NewProc("SetStretchBltMode")
)

var captureVtbl = [4]uintptr{handlerVtbl[0], handlerVtbl[1], handlerVtbl[2], syscall.NewCallback(func(this, hr uintptr) uintptr {
	(*comHandler)(unsafe.Pointer(this)).invoke(hr, 0)
	return 0
})}

func readSnapshot(stream unsafe.Pointer) (*snapshot, error) {
	var memory uintptr
	hr, _, _ := getStreamMemory.Call(uintptr(stream), uintptr(unsafe.Pointer(&memory)))
	if int32(hr) < 0 {
		return nil, fmt.Errorf("read wallpaper snapshot")
	}
	size, _, _ := globalSize.Call(memory)
	if size == 0 || size > 64<<20 {
		return nil, fmt.Errorf("wallpaper snapshot exceeds memory limit")
	}
	data, _, _ := globalLock.Call(memory)
	if data == 0 {
		return nil, fmt.Errorf("lock wallpaper snapshot")
	}
	defer globalUnlock.Call(memory)
	reader := bytes.NewReader(unsafe.Slice((*byte)(unsafe.Pointer(data)), int(size)))
	config, err := png.DecodeConfig(reader)
	if err != nil {
		return nil, err
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 32<<20 {
		return nil, fmt.Errorf("wallpaper snapshot dimensions exceed memory limit")
	}
	_, _ = reader.Seek(0, 0)
	img, err := png.Decode(reader)
	if err != nil {
		return nil, err
	}
	return makeSnapshot(img), nil
}

func makeSnapshot(img image.Image) *snapshot {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	// Keep only one still in memory, capped at 1080p even on 4K/multiple screens.
	if width > 1920 {
		height = height * 1920 / width
		width = 1920
	}
	if height > 1080 {
		width = width * 1080 / height
		height = 1080
	}
	width = max(1, width)
	height = max(1, height)
	s := &snapshot{width: int32(width), height: int32(height), pixels: make([]byte, width*height*4)}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x*bounds.Dx()/width, bounds.Min.Y+y*bounds.Dy()/height).RGBA()
			i := (y*width + x) * 4
			s.pixels[i] = byte(b >> 8)
			s.pixels[i+1] = byte(g >> 8)
			s.pixels[i+2] = byte(r >> 8)
			s.pixels[i+3] = 255
		}
	}
	return s
}

func (s *snapshot) paint(hwnd w32.HWND) {
	var paint w32.PAINTSTRUCT
	dc := w32.BeginPaint(hwnd, &paint)
	defer w32.EndPaint(hwnd, &paint)
	if dc == 0 {
		return
	}
	rect := w32.GetClientRect(hwnd)
	if rect == nil {
		return
	}
	// A top-down, opaque BGRA DIB. Windows redraws it only on exposure/resize.
	info := struct {
		Size                   uint32
		Width, Height          int32
		Planes, BitCount       uint16
		Compression, SizeImage uint32
		XPels, YPels           int32
		Used, Important        uint32
	}{Size: 40, Width: s.width, Height: -s.height, Planes: 1, BitCount: 32}
	setStretchMode.Call(uintptr(dc), 4) // HALFTONE for scaled snapshots
	stretchPixels.Call(uintptr(dc), 0, 0, uintptr(rect.Right), uintptr(rect.Bottom), 0, 0, uintptr(s.width), uintptr(s.height), uintptr(unsafe.Pointer(&s.pixels[0])), uintptr(unsafe.Pointer(&info)), 0, 0x00cc0020)
}

func (h *host) captureThenSuspend() {
	if h.cfg.Kind == "video" {
		// Stop media decoding before TrySuspend, which can reject playing media.
		_ = h.core.ExecuteScript("window.wallpaperPlayback?.(false)", nil)
	}
	if h.capturing || h.pending || h.unloaded {
		return
	}
	if !h.captureAvailable {
		h.beginSuspend()
		return
	}
	h.capturing = true
	var stream unsafe.Pointer
	hr, _, _ := createMemoryStream.Call(0, 1, uintptr(unsafe.Pointer(&stream)))
	if int32(hr) < 0 {
		h.capturing = false
		h.beginSuspend()
		return
	}
	h.captureStream = stream
	if h.captureCallback == nil {
		h.captureCallback = h.handler("{697e05e9-3d8f-45fa-96f4-8ffe1ededa89}", func(hr, _ uintptr) {
			stream := h.captureStream
			h.captureStream = nil
			h.capturing = false
			defer comCall(stream, 2)
			if h.stopping {
				return
			}
			if int32(hr) >= 0 {
				if still, err := readSnapshot(stream); err == nil {
					h.still = still
				}
			}
			if h.pauseReason != "" {
				h.beginSuspend()
			}
		})
		h.captureCallback.vtbl = &captureVtbl
	}
	if err := comCall(unsafe.Pointer(h.core), 30, 0, uintptr(stream), uintptr(unsafe.Pointer(h.captureCallback))); err != nil {
		h.capturing = false
		h.captureStream = nil
		_ = comCall(stream, 2)
		h.beginSuspend()
	}
}

func (h *host) showCover() {
	if h.still == nil {
		return
	}
	rect := w32.GetClientRect(h.window)
	if rect == nil {
		return
	}
	w32.SetWindowPos(h.cover, 0, 0, 0, int(rect.Right), int(rect.Bottom), w32.SWP_NOACTIVATE|w32.SWP_SHOWWINDOW)
	w32.InvalidateRect(h.cover, nil, false)
	w32.UpdateWindow(h.cover)
}

// Keep the still above the WebView while its compositor resumes. Two real page
// frames acknowledge a painted view; no per-frame screenshots or polling.
func (h *host) revealAfterPaint() {
	if h.revealing || h.still == nil {
		return
	}
	h.revealing = true
	h.showCover()
	method, _ := windows.UTF16PtrFromString("Runtime.evaluate")
	params, _ := json.Marshal(map[string]any{"expression": "new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve(true))))", "awaitPromise": true, "returnByValue": true})
	parameters, _ := windows.UTF16PtrFromString(string(params))
	if h.revealCallback == nil {
		h.revealCallback = h.handler("{5c4889f0-5ef6-4c5a-952c-d8f1b92d0574}", func(hr, result uintptr) {
			h.revealing = false
			if h.stopping || h.pauseReason != "" {
				return
			}
			if int32(hr) < 0 {
				h.emit("playing", "Snapshot held while the page resumes")
				return
			}
			// A page that overwrites rAF can throw; keep the still in that case.
			var response struct {
				ExceptionDetails json.RawMessage `json:"exceptionDetails"`
			}
			if result != 0 {
				_ = json.Unmarshal([]byte(windows.UTF16PtrToString((*uint16)(unsafe.Pointer(result)))), &response)
			}
			if len(response.ExceptionDetails) > 0 {
				return
			}
			w32.ShowWindow(h.cover, w32.SW_HIDE)
			h.captureAvailable = true
			h.emit("playing", "")
		})
	}
	if err := comCall(unsafe.Pointer(h.core), 36, uintptr(unsafe.Pointer(method)), uintptr(unsafe.Pointer(parameters)), uintptr(unsafe.Pointer(h.revealCallback))); err != nil {
		h.revealing = false
	}
}
