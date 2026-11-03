// Package hid exposes Linux hidraw devices to the peripheral protocols.
package hid

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

var ErrTimeout = errors.New("hid: timeout")

// Info describes a top-level collection on a hidraw node.
type Info struct {
	Path                                  string
	VendorID, ProductID, UsagePage, Usage uint16
	Interface                             int
	InputLen, OutputLen, FeatureLen       int
	Product                               string
	reports                               *reportLengths
}

func Enumerate(vid, pid uint16) ([]Info, error) {
	return EnumerateFunc(func(v, p uint16) bool { return v == vid && (pid == 0 || p == pid) })
}

func EnumerateFunc(match func(vid, pid uint16) bool) ([]Info, error) {
	const class = "/sys/class/hidraw"
	nodes, err := os.ReadDir(class)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var out []Info
	for _, node := range nodes {
		dir := filepath.Join(class, node.Name(), "device")
		raw, err := os.ReadFile(filepath.Join(dir, "uevent"))
		if err != nil {
			continue
		}
		var base Info
		var validID bool
		for line := range strings.SplitSeq(string(raw), "\n") {
			key, value, _ := strings.Cut(line, "=")
			switch key {
			case "HID_ID":
				fields := strings.Split(value, ":")
				if len(fields) != 3 {
					continue
				}
				v, verr := strconv.ParseUint(fields[1], 16, 16)
				p, perr := strconv.ParseUint(fields[2], 16, 16)
				validID = verr == nil && perr == nil
				base.VendorID, base.ProductID = uint16(v), uint16(p)
			case "HID_NAME":
				base.Product = value
			}
		}
		if !validID || !match(base.VendorID, base.ProductID) {
			continue
		}
		descriptor, err := os.ReadFile(filepath.Join(dir, "report_descriptor"))
		if err != nil {
			continue
		}
		base.Path = filepath.Join("/dev", node.Name())
		base.Interface = -1
		// A USB HID device is directly below its USB interface. Bluetooth
		// devices do not have a bInterfaceNumber.
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			if raw, err := os.ReadFile(filepath.Join(filepath.Dir(resolved), "bInterfaceNumber")); err == nil {
				if n, err := strconv.ParseUint(strings.TrimSpace(string(raw)), 16, 8); err == nil {
					base.Interface = int(n)
				}
			}
		}
		out = append(out, collections(base, descriptor)...)
	}
	return out, nil
}

// Device serializes I/O with Close so an fd cannot be used after reuse.
type Device struct {
	Info
	mu     sync.Mutex
	fd     int
	closed bool
}

func Open(info Info) (*Device, error) {
	fd, err := unix.Open(info.Path, unix.O_RDWR|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: info.Path, Err: err}
	}
	// Verify the IDs after opening: unplugging and replugging may reuse a node.
	var ids struct {
		Bus             uint32
		Vendor, Product uint16
	}
	const hidGetRawInfo = 0x80084803 // _IOR('H', 0x03, struct hidraw_devinfo)
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), hidGetRawInfo, uintptr(unsafe.Pointer(&ids)))
	if errno != 0 || ids.Vendor != info.VendorID || ids.Product != info.ProductID {
		_ = unix.Close(fd)
		if errno != 0 {
			return nil, fmt.Errorf("hid: raw device info: %w", errno)
		}
		return nil, errors.New("hid: device changed during discovery")
	}
	return &Device{Info: info, fd: fd}, nil
}

func (d *Device) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	return unix.Close(d.fd)
}

func (d *Device) SetFeature(data []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	buf, err := d.outgoing(featureReport, data)
	if err != nil {
		return err
	}
	n, err := d.feature(0x06, buf)
	if err == nil && n != len(buf) {
		return io.ErrShortWrite
	}
	return err
}

func (d *Device) GetFeature(reportID byte) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := d.reportLen(featureReport, reportID)
	if n == 0 {
		return nil, errors.New("hid: unknown feature report")
	}
	buf := make([]byte, n)
	buf[0] = reportID
	n, err := d.feature(0x07, buf)
	if err != nil {
		return nil, err
	}
	if n <= 0 || n > len(buf) {
		return nil, io.ErrUnexpectedEOF
	}
	// USB hidraw includes the ID slot for GET_FEATURE, even for report 0.
	// Some other HID transports return just the unnumbered payload.
	if reportID == 0 && d.Interface < 0 && n == len(buf)-1 {
		copy(buf[1:], buf[:n])
		buf[0] = 0
		n++
	}
	return buf[:n], nil
}

func (d *Device) feature(op uintptr, buf []byte) (int, error) {
	if d.closed {
		return 0, os.ErrClosed
	}
	// HIDIOC{S,G}FEATURE(len) uses _IOC(_IOC_READ|_IOC_WRITE, 'H', op, len).
	request := uintptr(0xc0004800) | uintptr(len(buf))<<16 | op
	n, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(d.fd), request, uintptr(unsafe.Pointer(&buf[0])))
	if errno != 0 {
		return 0, fmt.Errorf("hid: feature report: %w", errno)
	}
	return int(n), nil
}

func (d *Device) reportLen(kind int, id byte) int {
	if d.reports == nil {
		return 0
	}
	return d.reports[kind][id]
}

func (d *Device) outgoing(kind int, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("hid: missing report ID")
	}
	n := d.reportLen(kind, data[0])
	if n == 0 || len(data) > n {
		return nil, fmt.Errorf("hid: invalid length %d for report %#x (expected %d)", len(data), data[0], n)
	}
	buf := make([]byte, n)
	copy(buf, data)
	return buf, nil
}

func (d *Device) Write(data []byte, timeout time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	buf, err := d.outgoing(outputReport, data)
	if err != nil {
		return err
	}
	n, err := d.transfer(buf, unix.POLLOUT, timeout, unix.Write)
	if err == nil && n != len(buf) {
		return io.ErrShortWrite
	}
	return err
}

func (d *Device) Read(timeout time.Duration) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.InputLen == 0 {
		return nil, errors.New("hid: no input reports")
	}
	buf := make([]byte, d.InputLen)
	offset := 0
	if d.reportLen(inputReport, 0) != 0 {
		offset = 1 // hidraw omits the zero report ID on unnumbered reads
	}
	n, err := d.transfer(buf[offset:], unix.POLLIN, timeout, unix.Read)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, io.EOF
	}
	return buf[:n+offset], nil
}

func (d *Device) transfer(buf []byte, events int16, timeout time.Duration, op func(int, []byte) (int, error)) (int, error) {
	if d.closed {
		return 0, os.ErrClosed
	}
	deadline := time.Now().Add(timeout)
	fds := []unix.PollFd{{Fd: int32(d.fd), Events: events}}
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return 0, ErrTimeout
		}
		ms := (remaining-1)/time.Millisecond + 1
		n, err := unix.Poll(fds, int(min(ms, 1<<31-1)))
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return 0, err
		}
		if n == 0 {
			continue
		}
		if fds[0].Revents&(unix.POLLERR|unix.POLLHUP|unix.POLLNVAL) != 0 {
			return 0, unix.ENODEV
		}
		n, err = op(d.fd, buf)
		if err == unix.EAGAIN || err == unix.EINTR {
			continue
		}
		return n, err
	}
}
