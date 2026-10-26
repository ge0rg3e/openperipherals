// Package pulsar drives Pulsar mice and the Pulsar 4K receiver over the
// 16-byte report 8 configuration protocol. Settings live in a flat flash
// address space; every write is wrapped in a host-control toggle and
// confirmed by reading it back.
package pulsar

import (
	"crypto/rand"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"openperipherals/internal/hid"
)

const (
	VendorID    = 0x3710
	VGNVendorID = 0x3554 // the 4K receiver enumerates under VGN's vendor ID

	reportID    = 0x08
	packetLen   = 16
	ioTimeout   = 1200 * time.Millisecond
	DebounceMax = 15
)

// Commands.
const (
	cmdIdentify   = 0x01
	cmdOnline     = 0x03
	cmdBattery    = 0x04
	cmdWriteFlash = 0x07
	cmdReadFlash  = 0x08
	cmdVersion    = 0x12
)

// Flash addresses.
const (
	flashReportRate = 0
	flashCurrentDPI = 4
	flashLiftOff    = 10
	flashDPIValues  = 12
	flashDebounce   = 169
	flashMotionSync = 171
	flashSleep      = 173
	flashAngleSnap  = 175
	flashRipple     = 177
	flashPerfTime   = 183
)

// PollingRates are the rates the protocol can encode.
var PollingRates = []int{125, 250, 500, 1000, 2000, 4000, 8000}

// SleepOptions are the auto-sleep values the firmware accepts, in its own
// units of 10 seconds.
var SleepOptions = []int{1, 3, 6, 12, 30, 60, 180}

// LiftOffs are the lift-off distances in UI order (Low, Medium, High).
var LiftOffs = []string{"Low", "Medium", "High"}

// liftOffBytes encodes LiftOffs.
var liftOffBytes = []byte{3, 1, 2}

// Mouse is an open Pulsar mouse or receiver.
type Mouse struct {
	Name       string
	PID        uint16
	VID        uint16
	Wired      bool
	MaxPolling int
	dev        *hid.Device
	mu         sync.Mutex
}

// Supported reports whether vid is a Pulsar vendor ID.
func Supported(vid uint16) bool { return vid == VendorID || vid == VGNVendorID }

// OpenFrom opens the first collection that answers the identification
// handshake.
func OpenFrom(infos []hid.Info) (*Mouse, error) {
	for _, info := range infos {
		if !Supported(info.VendorID) || info.OutputLen != packetLen+1 || info.InputLen != packetLen+1 {
			continue
		}
		dev, err := hid.Open(info)
		if err != nil {
			continue
		}
		m := &Mouse{Name: info.Product, PID: info.ProductID, VID: info.VendorID, dev: dev}
		if m.Name == "" {
			m.Name = "Pulsar mouse"
		}
		if err := m.identify(); err != nil {
			dev.Close()
			continue
		}
		return m, nil
	}
	return nil, errors.New("no Pulsar mouse found")
}

// Close releases the device.
func (m *Mouse) Close() error { return m.dev.Close() }

func (m *Mouse) vgn() bool { return m.VID == VGNVendorID }

func (m *Mouse) identify() error {
	var challenge [8]byte
	rand.Read(challenge[:4])
	resp, err := m.query(cmdIdentify, challenge[:]...)
	if err != nil {
		return err
	}
	typ := resp[11]
	m.Wired = typ == 2 || typ == 3
	m.MaxPolling = map[byte]int{0: 1000, 1: 4000, 2: 1000, 3: 8000, 4: 2000, 5: 8000}[typ]
	if m.MaxPolling == 0 {
		m.MaxPolling = 1000
	}
	return nil
}

func checksum(p []byte) byte {
	sum := byte(reportID)
	for _, b := range p[:packetLen-1] {
		sum += b
	}
	return 0x55 - sum
}

// exchange sends one packet and waits for the reply to the same command
// (and, for flash access, the same address). Caller holds m.mu.
func (m *Mouse) exchange(p []byte) ([]byte, error) {
	p[packetLen-1] = checksum(p)
	buf := make([]byte, packetLen+1)
	buf[0] = reportID
	copy(buf[1:], p)
	if err := m.dev.Write(buf, ioTimeout); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(ioTimeout)
	for time.Now().Before(deadline) {
		in, err := m.dev.Read(time.Until(deadline))
		if err != nil {
			return nil, err
		}
		if len(in) < packetLen+1 || in[0] != reportID {
			continue
		}
		r := in[1:]
		if r[0] != p[0] || (p[0] == cmdReadFlash || p[0] == cmdWriteFlash) && (r[2] != p[2] || r[3] != p[3]) {
			continue
		}
		if r[1] != 0 {
			return nil, fmt.Errorf("pulsar: command %#02x rejected (status %d)", p[0], r[1])
		}
		return r, nil
	}
	return nil, fmt.Errorf("pulsar: no reply to command %#02x", p[0])
}

func (m *Mouse) query(cmd byte, params ...byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.queryLocked(cmd, params...)
}

func (m *Mouse) queryLocked(cmd byte, params ...byte) ([]byte, error) {
	p := make([]byte, packetLen)
	p[0], p[4] = cmd, byte(len(params))
	copy(p[5:], params)
	return m.exchange(p)
}

// control runs op with the mouse under host control.
func (m *Mouse) control(op func() error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.online(true); err != nil {
		return err
	}
	defer m.online(false)
	return op()
}

func (m *Mouse) online(on bool) error {
	for range 20 {
		p := make([]byte, packetLen)
		p[0], p[5] = cmdOnline, byte(boolInt(on))
		r, err := m.exchange(p)
		if err != nil {
			return err
		}
		if r[9] == 1 { // receiver busy
			time.Sleep(10 * time.Millisecond)
			continue
		}
		if on && r[5] != 1 {
			return errors.New("the Pulsar mouse is offline; move it or click a button, then retry")
		}
		return nil
	}
	return errors.New("the Pulsar receiver stayed busy")
}

func (m *Mouse) readFlash(addr, n int) ([]byte, error) {
	out := make([]byte, n)
	for off := 0; off < n; off += 10 {
		count := min(10, n-off)
		p := make([]byte, packetLen)
		a := addr + off
		p[0], p[2], p[3], p[4] = cmdReadFlash, byte(a>>8), byte(a), byte(count)
		r, err := m.exchange(p)
		if err != nil {
			return nil, err
		}
		copy(out[off:], r[5:5+count])
	}
	return out, nil
}

func (m *Mouse) writeFlash(addr int, data []byte) error {
	for off := 0; off < len(data); off += 10 {
		chunk := data[off:min(off+10, len(data))]
		p := make([]byte, packetLen)
		a := addr + off
		p[0], p[2], p[3], p[4] = cmdWriteFlash, byte(a>>8), byte(a), byte(len(chunk))
		copy(p[5:], chunk)
		if _, err := m.exchange(p); err != nil {
			return err
		}
	}
	return nil
}

// writeByte writes a value with its paired checksum byte and confirms it.
func (m *Mouse) writeByte(addr int, v byte) error {
	if err := m.writeFlash(addr, []byte{v, 0x55 - v}); err != nil {
		return err
	}
	got, err := m.readFlash(addr, 1)
	if err != nil {
		return err
	}
	if got[0] != v {
		return errors.New("the mouse did not confirm the setting")
	}
	return nil
}

// Status is a snapshot of the mouse's settings.
type Status struct {
	DPI        int
	Polling    int
	LiftOff    int // index into LiftOffs, -1 if unknown
	Debounce   int
	MotionSync bool
	AngleSnap  bool
	Ripple     bool
	Sleep      int // firmware units, see SleepOptions
	Battery    int // %, -1 if unknown
	Charging   bool
	Firmware   string
}

// Status reads every setting.
func (m *Mouse) Status() (Status, error) {
	var st Status
	err := m.control(func() error {
		flash, err := m.readFlash(flashReportRate, flashRipple+2)
		if err != nil {
			return err
		}
		stage := int(min(flash[flashCurrentDPI], 7))
		st.DPI = m.decodeDPI(flash[flashDPIValues+stage*4:])
		st.Polling = decodePolling(flash[flashReportRate])
		st.LiftOff = slices.Index(liftOffBytes, flash[flashLiftOff])
		st.Debounce = int(flash[flashDebounce])
		st.MotionSync = flash[flashMotionSync] == 1
		st.AngleSnap = flash[flashAngleSnap] == 1
		st.Ripple = flash[flashRipple] == 1
		st.Sleep = int(flash[flashSleep])
		st.Battery = -1
		if b, err := m.queryLocked(cmdBattery); err == nil && b[5] <= 100 {
			st.Battery, st.Charging = int(b[5]), b[6] == 1
		}
		if v, err := m.queryLocked(cmdVersion); err == nil {
			st.Firmware = fmt.Sprintf("v%d.%02x", v[5], v[6])
		}
		return nil
	})
	return st, err
}

func decodePolling(v byte) int {
	if v == 0 {
		return 0
	}
	if v >= 16 {
		return int(v) / 16 * 2000
	}
	return 1000 / int(v)
}

// SetPolling sets the polling rate.
func (m *Mouse) SetPolling(hz int) error {
	if !slices.Contains(PollingRates, hz) || hz > m.MaxPolling {
		return fmt.Errorf("this connection supports up to %d Hz", m.MaxPolling)
	}
	return m.control(func() error { return m.writeByte(flashReportRate, encodePolling(hz)) })
}

// encodePolling stores rates up to 1000 Hz as a divisor of 1000 and faster
// ones as rate/125.
func encodePolling(hz int) byte {
	if hz > 1000 {
		return byte(hz / 125)
	}
	return byte(1000 / hz)
}

// DPIOptions lists every DPI the sensor accepts: the 4K receiver uses flat
// 50 DPI steps, Pulsar mice 10 DPI steps with coarser high ranges.
func (m *Mouse) DPIOptions() []int {
	var out []int
	add := func(from, to, step int) {
		for v := from; v <= to; v += step {
			out = append(out, v)
		}
	}
	if m.vgn() {
		add(50, 26000, 50)
	} else {
		add(10, 10000, 10)
		add(10050, 30000, 50)
		add(30100, 32000, 100)
	}
	return out
}

// SnapDPI rounds dpi to the nearest value the sensor accepts.
func (m *Mouse) SnapDPI(dpi int) int {
	opts := m.DPIOptions()
	i, _ := slices.BinarySearch(opts, dpi)
	if i == len(opts) || i > 0 && dpi-opts[i-1] < opts[i]-dpi {
		i--
	}
	return opts[i]
}

func (m *Mouse) decodeDPI(d []byte) int {
	lo, dup, flags, sum := d[0], d[1], d[2], d[3]
	if lo != dup || lo+dup+flags+sum != 0x55 {
		return 0
	}
	if m.vgn() {
		return (int((flags>>2)&0x03)<<8 + int(lo) + 1) * 50
	}
	raw := int(lo) + int((flags&0x0c)>>2)<<8
	dpi := (raw + 1) * 10
	if flags&0x02 != 0 {
		dpi = dpi*5 + 10000
	}
	if flags&0x01 != 0 {
		dpi *= 2
	}
	return dpi
}

func (m *Mouse) encodeDPI(dpi int) []byte {
	var raw, ex int
	switch {
	case m.vgn():
		raw = dpi/50 - 1
	case dpi >= 30100:
		raw, ex = (dpi/2-10050)/50, 0x33
	case dpi >= 10050:
		raw, ex = (dpi-10050)/50, 0x22
	default:
		raw = dpi/10 - 1
	}
	high := byte(raw>>8) & 0x03
	flags := high<<2 | high<<6 | byte(ex)
	b := []byte{byte(raw), byte(raw), flags, 0}
	b[3] = 0x55 - b[0] - b[1] - b[2]
	return b
}

// SetDPI sets the DPI of the active stage.
func (m *Mouse) SetDPI(dpi int) error {
	if m.SnapDPI(dpi) != dpi {
		return fmt.Errorf("%d DPI is not supported by this sensor", dpi)
	}
	return m.control(func() error {
		cur, err := m.readFlash(flashCurrentDPI, 1)
		if err != nil {
			return err
		}
		addr := flashDPIValues + int(min(cur[0], 7))*4
		if err := m.writeFlash(addr, m.encodeDPI(dpi)); err != nil {
			return err
		}
		got, err := m.readFlash(addr, 4)
		if err != nil {
			return err
		}
		if v := m.decodeDPI(got); v != dpi {
			return fmt.Errorf("the mouse kept %d DPI instead of %d", v, dpi)
		}
		return nil
	})
}

// SetLiftOff sets the lift-off distance (an index into LiftOffs).
func (m *Mouse) SetLiftOff(level int) error {
	if level < 0 || level >= len(liftOffBytes) {
		return errors.New("unknown lift-off distance")
	}
	return m.control(func() error { return m.writeByte(flashLiftOff, liftOffBytes[level]) })
}

// SetDebounce sets the debounce time in ms.
func (m *Mouse) SetDebounce(ms int) error {
	if ms < 0 || ms > DebounceMax {
		return fmt.Errorf("debounce must be 0-%d ms", DebounceMax)
	}
	return m.control(func() error { return m.writeByte(flashDebounce, byte(ms)) })
}

// SetSleep sets the auto-sleep timeout (a value from SleepOptions).
func (m *Mouse) SetSleep(v int) error {
	if !slices.Contains(SleepOptions, v) {
		return errors.New("unsupported sleep timeout")
	}
	return m.control(func() error {
		if err := m.writeByte(flashSleep, byte(v)); err != nil {
			return err
		}
		return m.writeByte(flashPerfTime, byte(v))
	})
}

// SetMotionSync, SetAngleSnap and SetRipple toggle sensor processing.
func (m *Mouse) SetMotionSync(on bool) error { return m.setBool(flashMotionSync, on) }
func (m *Mouse) SetAngleSnap(on bool) error  { return m.setBool(flashAngleSnap, on) }
func (m *Mouse) SetRipple(on bool) error     { return m.setBool(flashRipple, on) }

func (m *Mouse) setBool(addr int, on bool) error {
	return m.control(func() error { return m.writeByte(addr, byte(boolInt(on))) })
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
