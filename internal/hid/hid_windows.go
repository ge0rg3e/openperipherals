// Package hid is a minimal, cgo-free HID transport for Windows built directly
// on setupapi.dll and hid.dll. It covers exactly what the peripheral drivers
// need: enumeration with capabilities, feature reports and output/input
// reports with timeouts.
package hid

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	setupapi = windows.NewLazySystemDLL("setupapi.dll")
	hiddll   = windows.NewLazySystemDLL("hid.dll")

	procGetClassDevs          = setupapi.NewProc("SetupDiGetClassDevsW")
	procEnumDeviceInterfaces  = setupapi.NewProc("SetupDiEnumDeviceInterfaces")
	procGetDeviceInterfaceDtl = setupapi.NewProc("SetupDiGetDeviceInterfaceDetailW")
	procDestroyDeviceInfoList = setupapi.NewProc("SetupDiDestroyDeviceInfoList")
	procHidGetHidGuid         = hiddll.NewProc("HidD_GetHidGuid")
	procHidGetAttributes      = hiddll.NewProc("HidD_GetAttributes")
	procHidGetPreparsedData   = hiddll.NewProc("HidD_GetPreparsedData")
	procHidFreePreparsedData  = hiddll.NewProc("HidD_FreePreparsedData")
	procHidPGetCaps           = hiddll.NewProc("HidP_GetCaps")
	procHidSetFeature         = hiddll.NewProc("HidD_SetFeature")
	procHidGetFeature         = hiddll.NewProc("HidD_GetFeature")
	procHidGetProductString   = hiddll.NewProc("HidD_GetProductString")
	procHidSetNumInputBuffers = hiddll.NewProc("HidD_SetNumInputBuffers")
	procHidPGetUsages         = hiddll.NewProc("HidP_GetUsages")
)

const (
	digcfPresent         = 0x02
	digcfDeviceInterface = 0x10
	hidpStatusSuccess    = 0x00110000
)

// ErrTimeout is returned when a read does not complete in time.
var ErrTimeout = errors.New("hid: timeout")

type spDeviceInterfaceData struct {
	CbSize             uint32
	InterfaceClassGUID windows.GUID
	Flags              uint32
	Reserved           uintptr
}

type hiddAttributes struct {
	Size          uint32
	VendorID      uint16
	ProductID     uint16
	VersionNumber uint16
}

type hidpCaps struct {
	Usage                     uint16
	UsagePage                 uint16
	InputReportByteLength     uint16
	OutputReportByteLength    uint16
	FeatureReportByteLength   uint16
	Reserved                  [17]uint16
	NumberLinkCollectionNodes uint16
	NumberInputButtonCaps     uint16
	NumberInputValueCaps      uint16
	NumberInputDataIndices    uint16
	NumberOutputButtonCaps    uint16
	NumberOutputValueCaps     uint16
	NumberOutputDataIndices   uint16
	NumberFeatureButtonCaps   uint16
	NumberFeatureValueCaps    uint16
	NumberFeatureDataIndices  uint16
}

// Info describes one HID top-level collection.
type Info struct {
	Path       string
	VendorID   uint16
	ProductID  uint16
	UsagePage  uint16
	Usage      uint16
	Interface  int // USB interface number parsed from the path, -1 if absent
	InputLen   int
	OutputLen  int
	FeatureLen int
	Product    string
}

// Enumerate lists every present HID collection matching vid (and pid when
// non-zero).
func Enumerate(vid, pid uint16) ([]Info, error) {
	return EnumerateFunc(func(v, p uint16) bool { return v == vid && (pid == 0 || p == pid) })
}

// EnumerateFunc lists every present HID collection whose vendor and product
// IDs (read from the device path) match. Only matching collections are
// opened, so one pass can look for many devices cheaply.
func EnumerateFunc(match func(vid, pid uint16) bool) ([]Info, error) {
	var guid windows.GUID
	procHidGetHidGuid.Call(uintptr(unsafe.Pointer(&guid)))

	set, _, err := procGetClassDevs.Call(uintptr(unsafe.Pointer(&guid)), 0, 0, digcfPresent|digcfDeviceInterface)
	if windows.Handle(set) == windows.InvalidHandle {
		return nil, fmt.Errorf("SetupDiGetClassDevs: %w", err)
	}
	defer procDestroyDeviceInfoList.Call(set)

	var out []Info
	for i := uint32(0); ; i++ {
		var did spDeviceInterfaceData
		did.CbSize = uint32(unsafe.Sizeof(did))
		r, _, _ := procEnumDeviceInterfaces.Call(set, 0, uintptr(unsafe.Pointer(&guid)), uintptr(i), uintptr(unsafe.Pointer(&did)))
		if r == 0 {
			break
		}
		path, ok := interfacePath(set, &did)
		if !ok {
			continue
		}
		vid, pid, ok := pathIDs(path)
		if !ok || !match(vid, pid) {
			continue
		}
		info, err := query(path)
		if err != nil {
			continue
		}
		out = append(out, info)
	}
	return out, nil
}

func interfacePath(set uintptr, did *spDeviceInterfaceData) (string, bool) {
	var size uint32
	procGetDeviceInterfaceDtl.Call(set, uintptr(unsafe.Pointer(did)), 0, 0, uintptr(unsafe.Pointer(&size)), 0)
	if size == 0 {
		return "", false
	}
	buf := make([]byte, size)
	// cbSize of SP_DEVICE_INTERFACE_DETAIL_DATA_W: DWORD + WCHAR[1], padded.
	cb := uint32(8)
	if unsafe.Sizeof(uintptr(0)) == 4 {
		cb = 6
	}
	*(*uint32)(unsafe.Pointer(&buf[0])) = cb
	r, _, _ := procGetDeviceInterfaceDtl.Call(set, uintptr(unsafe.Pointer(did)), uintptr(unsafe.Pointer(&buf[0])), uintptr(size), 0, 0)
	if r == 0 {
		return "", false
	}
	return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(&buf[4]))), true
}

func query(path string) (Info, error) {
	// Access 0 lets us read attributes even for system-owned keyboards/mice.
	h, err := openPath(path, 0, 0)
	if err != nil {
		return Info{}, err
	}
	defer windows.CloseHandle(h)

	var attr hiddAttributes
	attr.Size = uint32(unsafe.Sizeof(attr))
	if r, _, _ := procHidGetAttributes.Call(uintptr(h), uintptr(unsafe.Pointer(&attr))); r == 0 {
		return Info{}, errors.New("HidD_GetAttributes failed")
	}
	var pp uintptr
	if r, _, _ := procHidGetPreparsedData.Call(uintptr(h), uintptr(unsafe.Pointer(&pp))); r == 0 {
		return Info{}, errors.New("HidD_GetPreparsedData failed")
	}
	defer procHidFreePreparsedData.Call(pp)
	var caps hidpCaps
	if r, _, _ := procHidPGetCaps.Call(pp, uintptr(unsafe.Pointer(&caps))); r != hidpStatusSuccess {
		return Info{}, errors.New("HidP_GetCaps failed")
	}

	var name [128]uint16
	product := ""
	if r, _, _ := procHidGetProductString.Call(uintptr(h), uintptr(unsafe.Pointer(&name[0])), uintptr(len(name)*2)); r != 0 {
		product = windows.UTF16ToString(name[:])
	}

	return Info{
		Path:       path,
		VendorID:   attr.VendorID,
		ProductID:  attr.ProductID,
		UsagePage:  caps.UsagePage,
		Usage:      caps.Usage,
		Interface:  parseInterface(path),
		InputLen:   int(caps.InputReportByteLength),
		OutputLen:  int(caps.OutputReportByteLength),
		FeatureLen: int(caps.FeatureReportByteLength),
		Product:    product,
	}, nil
}

// pathIDs reads the vid_xxxx and pid_xxxx fields of a device path.
func pathIDs(path string) (vid, pid uint16, ok bool) {
	lower := strings.ToLower(path)
	field := func(tag string) (uint16, bool) {
		i := strings.Index(lower, tag)
		if i < 0 || i+len(tag)+4 > len(lower) {
			return 0, false
		}
		v, err := strconv.ParseUint(lower[i+len(tag):i+len(tag)+4], 16, 16)
		return uint16(v), err == nil
	}
	vid, ok1 := field("vid_")
	pid, ok2 := field("pid_")
	return vid, pid, ok1 && ok2
}

func parseInterface(path string) int {
	lower := strings.ToLower(path)
	i := strings.Index(lower, "&mi_")
	if i < 0 || i+6 > len(lower) {
		return -1
	}
	var n int
	if _, err := fmt.Sscanf(lower[i+4:i+6], "%02x", &n); err != nil {
		return -1
	}
	return n
}

func openPath(path string, access uint32, flags uint32) (windows.Handle, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return windows.InvalidHandle, err
	}
	return windows.CreateFile(p, access,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil,
		windows.OPEN_EXISTING, flags, 0)
}

// Device is an open HID collection. All methods are safe for concurrent use.
type Device struct {
	Info
	mu     sync.Mutex
	h      windows.Handle
	ev     windows.Handle
	pp     uintptr // preparsed report descriptor, 0 if unavailable
	closed bool
}

// Open opens a collection for I/O. It falls back to a zero-access handle
// (feature reports only) when Windows owns the collection exclusively.
func Open(info Info) (*Device, error) {
	h, err := openPath(info.Path, windows.GENERIC_READ|windows.GENERIC_WRITE, windows.FILE_FLAG_OVERLAPPED)
	if err != nil {
		h, err = openPath(info.Path, 0, windows.FILE_FLAG_OVERLAPPED)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", info.Path, err)
		}
	}
	ev, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		windows.CloseHandle(h)
		return nil, err
	}
	procHidSetNumInputBuffers.Call(uintptr(h), 64)
	d := &Device{Info: info, h: h, ev: ev}
	procHidGetPreparsedData.Call(uintptr(h), uintptr(unsafe.Pointer(&d.pp)))
	return d, nil
}

// Close releases the handle.
func (d *Device) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	if d.pp != 0 {
		procHidFreePreparsedData.Call(d.pp)
	}
	windows.CloseHandle(d.ev)
	return windows.CloseHandle(d.h)
}

// SetFeature sends a feature report. data[0] must be the report ID.
func (d *Device) SetFeature(data []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	buf := d.pad(data, d.FeatureLen)
	r, _, err := procHidSetFeature.Call(uintptr(d.h), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r == 0 {
		return fmt.Errorf("HidD_SetFeature: %w", err)
	}
	return nil
}

// GetFeature reads a feature report with the given report ID.
func (d *Device) GetFeature(reportID byte) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	buf := make([]byte, d.FeatureLen)
	buf[0] = reportID
	r, _, err := procHidGetFeature.Call(uintptr(d.h), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r == 0 {
		return nil, fmt.Errorf("HidD_GetFeature: %w", err)
	}
	return buf, nil
}

// Write sends an output report. data[0] must be the report ID.
func (d *Device) Write(data []byte, timeout time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	buf := d.pad(data, d.OutputLen)
	_, err := d.overlapped(func(ov *windows.Overlapped) error {
		return windows.WriteFile(d.h, buf, nil, ov)
	}, timeout)
	return err
}

// Read reads one input report, including the leading report ID byte.
func (d *Device) Read(timeout time.Duration) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	buf := make([]byte, max(d.InputLen, 1))
	n, err := d.overlapped(func(ov *windows.Overlapped) error {
		return windows.ReadFile(d.h, buf, nil, ov)
	}, timeout)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func (d *Device) overlapped(op func(*windows.Overlapped) error, timeout time.Duration) (int, error) {
	if d.closed {
		return 0, errors.New("hid: device closed")
	}
	windows.ResetEvent(d.ev)
	ov := windows.Overlapped{HEvent: d.ev}
	err := op(&ov)
	if err != nil && err != windows.ERROR_IO_PENDING {
		return 0, err
	}
	ev, _ := windows.WaitForSingleObject(d.ev, uint32(timeout.Milliseconds()))
	if ev != windows.WAIT_OBJECT_0 {
		windows.CancelIo(d.h)
		var n uint32
		windows.GetOverlappedResult(d.h, &ov, &n, true)
		return 0, ErrTimeout
	}
	var n uint32
	if err := windows.GetOverlappedResult(d.h, &ov, &n, false); err != nil {
		return 0, err
	}
	return int(n), nil
}

// Usages decodes an input report (as returned by Read) and lists the active
// button/selector usages on the given usage page, using the device's own
// report descriptor.
func (d *Device) Usages(report []byte, page uint16) ([]uint16, error) {
	if d.pp == 0 || len(report) == 0 {
		return nil, errors.New("hid: no report descriptor")
	}
	list := make([]uint16, 32)
	n := uint32(len(list))
	r, _, _ := procHidPGetUsages.Call(0 /* HidP_Input */, uintptr(page), 0,
		uintptr(unsafe.Pointer(&list[0])), uintptr(unsafe.Pointer(&n)), d.pp,
		uintptr(unsafe.Pointer(&report[0])), uintptr(len(report)))
	if r != hidpStatusSuccess {
		return nil, fmt.Errorf("HidP_GetUsages: status %#x", r)
	}
	return list[:n], nil
}

// pad copies data into a buffer of exactly n bytes (Windows rejects reports
// whose length differs from the collection's declared report length).
func (d *Device) pad(data []byte, n int) []byte {
	if n <= 0 {
		n = len(data)
	}
	buf := make([]byte, n)
	copy(buf, data)
	return buf
}
