package wallpaper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"openperipherals/internal/appdata"

	"github.com/wailsapp/go-webview2/pkg/edge"
	"github.com/wailsapp/wails/v3/pkg/w32"
	"golang.org/x/sys/windows"
)

const hostMessage = 0x8001

type host struct {
	videoServer                                                      *http.Server
	videoURL                                                         string
	captureCallback, suspendCallback, revealCallback                 *comHandler
	captureStream                                                    unsafe.Pointer
	cfg                                                              Config
	control, window, parent                                          w32.HWND
	browser                                                          *edge.Chromium
	core                                                             *edge.ICoreWebView2
	suspend                                                          *edge.ICoreWebView2_3
	handlers                                                         []*comHandler
	commands                                                         chan Command
	out                                                              *json.Encoder
	state                                                            State
	manual, displayOff, sleeping, ready, unloaded, pending, stopping bool
	bounds                                                           w32.RECT
	started                                                          time.Time
	pauseReason                                                      string
	cover                                                            w32.HWND
	still                                                            *snapshot
	capturing, captureAvailable, revealing                           bool
}

// RunHost is an isolated mode of the same executable. It never starts Wails,
// registers bindings, or opens a device. EOF on stdin closes it with the parent.
func RunHost(in io.Reader, out io.Writer) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	decoder := json.NewDecoder(in)
	var first Command
	if err := decoder.Decode(&first); err != nil {
		return err
	}
	cfg, err := Validate(first.Config)
	if err != nil {
		return err
	}
	h := &host{cfg: cfg, commands: make(chan Command, 8), out: json.NewEncoder(out), started: time.Now()}
	defer func() {
		if h.control != 0 {
			w32.DestroyWindow(h.control)
		}
	}()
	// COM must live on this STA/message-pump thread.
	ole := windows.NewLazySystemDLL("ole32.dll")
	hr, _, _ := ole.NewProc("CoInitializeEx").Call(0, 2)
	if int32(hr) < 0 {
		return fmt.Errorf("initialise wallpaper COM: 0x%x", hr)
	}
	defer ole.NewProc("CoUninitialize").Call()
	className, _ := windows.UTF16PtrFromString("OpenPeripheralsWallpaperHost")
	// Both the native surface and WebView use the same dark clear colour. A
	// white native class brush otherwise flashes between resume and composition.
	background := w32.CreateSolidBrush(w32.COLORREF(0x190d08))
	defer w32.DeleteObject(w32.HGDIOBJ(background))
	class := w32.WNDCLASSEX{Instance: w32.GetModuleHandle(""), ClassName: className, Background: background}
	class.Size = uint32(unsafe.Sizeof(class))
	class.WndProc = syscall.NewCallback(h.wndProc)
	if w32.RegisterClassEx(&class) == 0 {
		return fmt.Errorf("register wallpaper window")
	}
	h.control = w32.CreateWindowEx(w32.WS_EX_TOOLWINDOW|w32.WS_EX_NOACTIVATE, className, nil, 0, 0, 0, 0, 0, 0, 0, class.Instance, nil)
	if h.control == 0 {
		return fmt.Errorf("create wallpaper controller")
	}
	go func() {
		for {
			var command Command
			if decoder.Decode(&command) != nil {
				command.Action = "stop"
			}
			h.commands <- command
			w32.PostMessage(h.control, hostMessage, 0, 0)
			if command.Action == "stop" {
				return
			}
		}
	}()
	// Messages, rather than a busy loop, handle power/session changes and commands.
	w32.SetTimer(h.control, 1, 1000, 0)
	wts := windows.NewLazySystemDLL("wtsapi32.dll")
	wts.NewProc("WTSRegisterSessionNotification").Call(uintptr(h.control), 0)
	defer wts.NewProc("WTSUnRegisterSessionNotification").Call(uintptr(h.control))
	displayGUID, _ := windows.GUIDFromString("{6fe69556-704a-47a0-8f24-c28d936fda47}")
	powerHandle, _, _ := user32.NewProc("RegisterPowerSettingNotification").Call(uintptr(h.control), uintptr(unsafe.Pointer(&displayGUID)), 0)
	if powerHandle != 0 {
		defer user32.NewProc("UnregisterPowerSettingNotification").Call(powerHandle)
	}
	// Also clean up partially-created native surfaces if startup fails.
	defer h.closeBrowser()
	if err = h.create(className, class.Instance); err != nil {
		return err
	}
	var msg w32.MSG
	for w32.GetMessage(&msg, 0, 0, 0) > 0 {
		w32.TranslateMessage(&msg)
		w32.DispatchMessage(&msg)
	}
	return nil
}

func (h *host) emit(phase, reason string) {
	next := State{Phase: phase, Reason: reason, Snapshot: h.still != nil}
	if h.state != next {
		h.state = next
		_ = h.out.Encode(next)
	}
}

func (h *host) fail(err error) {
	h.emit("error", err.Error())
	h.stopping = true
	w32.PostQuitMessage(1)
}

func (h *host) create(className *uint16, instance w32.HINSTANCE) error {
	parent, err := desktopParent()
	if err != nil {
		return err
	}
	h.parent = parent
	h.bounds = targetBounds(h.cfg.Span)
	h.window = w32.CreateWindowEx(w32.WS_EX_TOOLWINDOW|w32.WS_EX_NOACTIVATE|w32.WS_EX_TRANSPARENT, className, nil, w32.WS_CHILD, 0, 0, 10, 10, parent, 0, instance, nil)
	if h.window == 0 {
		return fmt.Errorf("create desktop wallpaper surface")
	}
	h.cover = w32.CreateWindowEx(w32.WS_EX_NOACTIVATE|w32.WS_EX_TRANSPARENT, className, nil, w32.WS_CHILD, 0, 0, 10, 10, h.window, 0, instance, nil)
	if h.cover == 0 {
		return fmt.Errorf("create wallpaper snapshot surface")
	}
	if err = attachDesktop(h.window, parent, h.bounds); err != nil {
		return err
	}
	h.emit("loading", "Starting wallpaper")
	b := edge.NewChromium()
	h.browser = b
	dir, err := appdata.Dir()
	if err != nil {
		return err
	}
	b.DataPath = filepath.Join(dir, "wallpaper-webview")
	b.AdditionalBrowserArgs = []string{"--mute-audio", "--autoplay-policy=no-user-gesture-required", "--disable-background-networking"}
	b.SetGlobalPermission(edge.CoreWebView2PermissionStateDeny)
	b.SetErrorCallback(func(err error) {
		if !h.stopping {
			h.fail(err)
		}
	})
	// Only our fixed video document gets this status-only channel. HTML and URL
	// wallpapers retain disabled web messaging and no device-control bridge.
	b.MessageCallback = func(message string, _ *edge.ICoreWebView2, _ *edge.ICoreWebView2WebMessageReceivedEventArgs) {
		if h.cfg.Kind != "video" || h.unloaded || h.stopping {
			return
		}
		switch message {
		case "video-ready":
			h.contentReady()
		case "video-playing":
			if h.ready && h.pauseReason == "" && !h.revealing {
				h.emit("playing", "")
			}
		case "video-error":
			h.fail(fmt.Errorf("MP4 could not play; choose a supported video such as H.264 in an MP4 file"))
		}
	}
	b.NavigationCompletedCallback = func(_ *edge.ICoreWebView2, args *edge.ICoreWebView2NavigationCompletedEventArgs) {
		if h.unloaded || h.stopping {
			return
		}
		var success int32
		if err := comCall(unsafe.Pointer(args), 3, uintptr(unsafe.Pointer(&success))); err != nil || success == 0 {
			h.fail(fmt.Errorf("wallpaper could not load; check the URL, network, or local assets"))
			return
		}
		if h.cfg.Kind != "video" {
			h.contentReady()
		} else {
			// Chromium defers media loading in a hidden WebView. Make the
			// surface visible before waiting for the first decoded video frame.
			if err := h.browser.Show(); err != nil {
				h.fail(err)
				return
			}
			w32.ShowWindow(h.window, w32.SW_SHOWNA)
		}
	}
	b.ProcessFailedCallback = func(*edge.ICoreWebView2, *edge.ICoreWebView2ProcessFailedEventArgs) {
		h.fail(fmt.Errorf("wallpaper renderer stopped; apply the wallpaper again"))
	}
	if !b.Embed(uintptr(h.window)) || h.stopping {
		return fmt.Errorf("could not initialise WebView2")
	}
	controller := b.GetController()
	if controller == nil {
		return fmt.Errorf("WebView2 controller unavailable")
	}
	if controller2 := controller.GetICoreWebView2Controller2(); controller2 != nil {
		err = controller2.PutDefaultBackgroundColor(edge.COREWEBVIEW2_COLOR{A: 255, R: 8, G: 13, B: 25})
		_ = comCall(unsafe.Pointer(controller2), 2)
		if err != nil {
			return err
		}
	}
	h.core, err = controller.GetCoreWebView2()
	if err != nil {
		return err
	}
	h.suspend = h.core.GetICoreWebView2_3()
	if h.suspend == nil {
		return fmt.Errorf("update WebView2 to enable wallpaper suspension")
	}
	settings, err := b.GetSettings()
	if err != nil {
		return err
	}
	defer settings.Release()
	for _, set := range []func(bool) error{settings.PutIsWebMessageEnabled, settings.PutAreHostObjectsAllowed, settings.PutAreDefaultScriptDialogsEnabled, settings.PutAreDefaultContextMenusEnabled, settings.PutAreDevToolsEnabled, settings.PutIsStatusBarEnabled, settings.PutIsZoomControlEnabled} {
		if err = set(false); err != nil {
			return err
		}
	}
	if h.cfg.Kind == "video" {
		if err = settings.PutIsWebMessageEnabled(true); err != nil {
			return err
		}
	}
	_ = b.PutIsGeneralAutofillEnabled(false)
	_ = b.PutIsPasswordAutosaveEnabled(false)
	// NewWindowRequested is handled natively; scripts cannot open browser windows.
	popup := h.handler("{d4c185fe-c81c-4989-97af-2d3fa7ab5651}", func(_, args uintptr) {
		_ = comCall(unsafe.Pointer(args), 6, 1) // put_Handled
	})
	var token int64
	if err = comCall(unsafe.Pointer(h.core), 44, uintptr(unsafe.Pointer(popup)), uintptr(unsafe.Pointer(&token))); err != nil {
		return err
	}
	// Wallpaper content cannot launch external protocols or save downloads.
	navigation := h.handler("{9adbe429-f36d-432b-9ddc-f8881fbd76e3}", func(_, args uintptr) {
		var address *uint16
		if comCall(unsafe.Pointer(args), 3, uintptr(unsafe.Pointer(&address))) != nil {
			return
		}
		defer windows.CoTaskMemFree(unsafe.Pointer(address))
		uri, err := url.Parse(windows.UTF16PtrToString(address))
		if err != nil || (uri.Scheme != "https" && uri.Scheme != "http" && uri.Scheme != "data" && uri.Scheme != "blob" && uri.String() != "about:blank") {
			_ = comCall(unsafe.Pointer(args), 8, 1)
		}
	})
	if err = comCall(unsafe.Pointer(h.core), 7, uintptr(unsafe.Pointer(navigation)), uintptr(unsafe.Pointer(&token))); err != nil {
		return err
	}
	core4, err := comQuery(unsafe.Pointer(h.core), "{20d02d59-6df2-42dc-bd06-f98a694b1302}")
	if err != nil {
		return fmt.Errorf("update WebView2 to enable wallpaper download blocking: %w", err)
	}
	defer comCall(core4, 2)
	download := h.handler("{efedc989-c396-41ca-83f7-07f845a55724}", func(_, args uintptr) {
		_ = comCall(unsafe.Pointer(args), 5, 1) // put_Cancel
		_ = comCall(unsafe.Pointer(args), 9, 1) // put_Handled
	})
	if err = comCall(core4, 75, uintptr(unsafe.Pointer(download)), uintptr(unsafe.Pointer(&token))); err != nil {
		return err
	}
	b.Resize()
	// Wait for script registration before navigating, including on URL redirects.
	script, _ := windows.UTF16PtrFromString(animationScript(h.cfg.FPS))
	registered := h.handler("{b99369f3-9b11-47b5-bc6f-8e7895fcea17}", func(hr, _ uintptr) {
		if int32(hr) < 0 {
			h.fail(fmt.Errorf("could not install wallpaper animation limiter"))
			return
		}
		if err := h.navigate(); err != nil {
			h.fail(err)
		}
	})
	return comCall(unsafe.Pointer(h.core), 27, uintptr(unsafe.Pointer(script)), uintptr(unsafe.Pointer(registered)))
}

func (h *host) handler(iid string, fn func(uintptr, uintptr)) *comHandler {
	callback := newHandler(iid, fn)
	h.handlers = append(h.handlers, callback)
	return callback
}

func (h *host) contentReady() {
	h.ready = true
	h.captureAvailable = true
	h.updatePolicy()
	if h.pauseReason == "" {
		h.present()
	}
}

func (h *host) navigate() error {
	h.unloaded = false
	h.ready = false
	h.started = time.Now()
	switch h.cfg.Kind {
	case "builtin":
		h.browser.NavigateToString(builtinHTML)
	case "url":
		h.browser.Navigate(h.cfg.Source)
	case "local":
		if err := h.suspend.SetVirtualHostNameToFolderMapping("wallpaper.openperipherals.invalid", filepath.Dir(h.cfg.Source), edge.COREWEBVIEW2_HOST_RESOURCE_ACCESS_KIND_DENY_CORS); err != nil {
			return err
		}
		h.browser.Navigate("https://wallpaper.openperipherals.invalid/" + url.PathEscape(filepath.Base(h.cfg.Source)))
	case "video":
		if h.videoServer == nil {
			var err error
			h.videoServer, h.videoURL, err = serveVideo(h.cfg.Source)
			if err != nil {
				return err
			}
		}
		h.browser.Navigate(h.videoURL)
	}
	// NavigationCompleted presents the surface. This also avoids showing the
	// empty about:blank page during the rare unload-and-reload fallback.
	return nil
}

func (h *host) wndProc(hwnd w32.HWND, msg uint32, wp, lp uintptr) uintptr {
	switch msg {
	case w32.WM_PAINT:
		if h.still != nil && (hwnd == h.window || hwnd == h.cover) {
			h.still.paint(hwnd)
			return 0
		}
	case w32.WM_ERASEBKGND:
		if h.still != nil && (hwnd == h.window || hwnd == h.cover) {
			return 1
		}
	case hostMessage:
		select {
		case cmd := <-h.commands:
			switch cmd.Action {
			case "stop":
				h.stopping = true
				w32.PostQuitMessage(0)
			case "pause":
				h.manual = true
				h.updatePolicy()
			case "resume":
				h.manual = false
				h.updatePolicy()
			}
		default:
		}
		return 0
	case w32.WM_TIMER:
		if !h.ready && time.Since(h.started) > 30*time.Second {
			h.fail(fmt.Errorf("wallpaper startup timed out"))
			return 0
		}
		h.updatePolicy()
		return 0
	case 0x2b1: // WM_WTSSESSION_CHANGE; check input desktop on the next policy tick.
		h.updatePolicy()
		return 0
	case 0x218: // WM_POWERBROADCAST
		switch wp {
		case 4:
			h.sleeping = true
		case 7, 18:
			h.sleeping = false
		case 0x8013:
			if lp != 0 { // POWERBROADCAST_SETTING: GUID + DWORD length + byte data
				length := *(*uint32)(unsafe.Pointer(lp + 16))
				if length >= 4 {
					h.displayOff = *(*uint32)(unsafe.Pointer(lp + 20)) != 1
				}
			}
		}
		h.updatePolicy()
		return 1
	case w32.WM_SIZE:
		if h.browser != nil && h.ready && hwnd == h.window {
			h.browser.Resize()
			if h.pauseReason != "" || h.revealing {
				h.showCover()
			}
		}
		return 0
	case w32.WM_MOUSEACTIVATE:
		return 3 // MA_NOACTIVATE
	case w32.WM_NCHITTEST:
		return ^uintptr(0) // HTTRANSPARENT
	}
	return w32.DefWindowProc(hwnd, msg, wp, lp)
}

func (h *host) updatePolicy() {
	if !h.ready || h.stopping {
		return
	}
	if !w32.IsWindow(h.parent) || !w32.IsWindow(h.window) {
		h.fail(fmt.Errorf("Explorer restarted; apply the wallpaper again"))
		return
	}
	bounds := targetBounds(h.cfg.Span)
	if bounds != h.bounds {
		h.bounds = bounds
		if err := attachDesktop(h.window, h.parent, bounds); err != nil {
			h.fail(err)
			return
		}
		h.browser.Resize()
	}
	reason := ""
	switch {
	case h.manual:
		reason = "Paused by you"
	case h.sleeping:
		reason = "PC is sleeping"
	case h.displayOff:
		reason = "Display is off or dimmed"
	case desktopLocked():
		reason = "Session is locked"
	case h.cfg.PauseBattery && onBattery():
		reason = "Running on battery"
	case h.cfg.PauseCovered && foregroundCovers(h.bounds, h.cfg.Span):
		reason = "Desktop is covered"
	}
	if reason == h.pauseReason {
		return
	}
	h.pauseReason = reason
	if reason == "" {
		if h.pending {
			return
		} // completion resumes a raced suspension
		h.resume()
		return
	}
	h.captureThenSuspend()
}

func (h *host) beginSuspend() {
	if h.stopping || h.pauseReason == "" {
		return
	}
	h.captureAvailable = false
	h.showCover()
	_ = h.browser.Hide()
	if h.still == nil {
		w32.ShowWindow(h.window, w32.SW_HIDE)
	}
	if h.pending || h.unloaded {
		return
	}
	h.pending = true
	if h.suspendCallback == nil {
		h.suspendCallback = h.handler("{9da43ccc-26e1-4dad-b56c-d8961c94c571}", func(hr, result uintptr) {
			h.pending = false
			if h.stopping {
				return
			}
			if h.pauseReason == "" {
				h.resume()
				return
			}
			if int32(hr) < 0 || result == 0 {
				h.unload()
			} else {
				h.emit("paused", h.pauseReason)
			}
		})
	}
	// ICoreWebView2_2 has 68 slots; TrySuspend is the first _3 method.
	if err := comCall(unsafe.Pointer(h.suspend), 68, uintptr(unsafe.Pointer(h.suspendCallback))); err != nil {
		h.pending = false
		h.unload()
	}
}

func (h *host) unload() {
	h.unloaded = true
	h.browser.Navigate("about:blank")
	h.emit("paused", h.pauseReason+" (page unloaded to save resources)")
}

func (h *host) resume() {
	if h.stopping {
		return
	}
	_ = comCall(unsafe.Pointer(h.suspend), 69)
	if h.unloaded {
		if err := h.navigate(); err != nil {
			h.fail(err)
		}
		return
	}
	h.present()
}

func (h *host) present() {
	if h.cfg.Kind == "video" {
		_ = h.core.ExecuteScript("window.wallpaperPlayback?.(true)", nil)
	}
	if h.still != nil {
		h.showCover()
	}
	if err := h.browser.Show(); err != nil {
		h.fail(err)
		return
	}
	w32.ShowWindow(h.window, w32.SW_SHOWNA)
	if h.still != nil {
		h.captureAvailable = false
		h.emit("resuming", "Showing last frame while the live view resumes")
		h.revealAfterPaint()
	} else {
		h.captureAvailable = true
		if h.cfg.Kind == "video" {
			h.emit("loading", "Starting video playback")
		} else {
			h.emit("playing", "")
		}
	}
}

func (h *host) closeBrowser() {
	h.stopping = true
	// Reveal Explorer before closing WebView2, so its clear colour or the
	// retained pause snapshot cannot remain visible during teardown.
	if h.window != 0 && w32.IsWindow(h.window) {
		w32.ShowWindow(h.window, w32.SW_HIDE)
	}
	if h.videoServer != nil {
		_ = h.videoServer.Close()
	}
	if h.browser != nil {
		h.browser.ShuttingDown()
		if c := h.browser.GetController(); c != nil {
			_ = comCall(unsafe.Pointer(c), 24)
			c.Release()
		}
	}
	if h.suspend != nil {
		h.suspend.Release()
	}
	if h.core != nil {
		h.core.Release()
	}
	if h.window != 0 && w32.IsWindow(h.window) {
		w32.DestroyWindow(h.window)
	}
	h.window, h.cover = 0, 0
	h.still = nil
	// Callback objects remain pinned until this isolated process exits. WebView2
	// may finish releasing asynchronous callbacks after the controller closes.
}
