// Package razer drives Razer keyboards and mice that use the 90-byte feature
// report protocol (reverse-engineered by OpenRazer). Keyboards use either the
// classic matrix (class 0x03) or the extended matrix (class 0x0f).
package razer

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"openperipherals/internal/hid"
)

const (
	VendorID  = 0x1532
	reportLen = 90

	statusBusy        = 0x01
	statusOK          = 0x02
	statusUnsupported = 0x05

	noStore  = 0x00
	varStore = 0x01 // persist to on-device storage

	ledMacro     = 0x07
	ledGame      = 0x08
	ledLogo      = 0x04
	ledBacklight = 0x05
)

// Effect identifies a hardware lighting effect.
type Effect string

const (
	EffectOff       Effect = "off"
	EffectStatic    Effect = "static"
	EffectSpectrum  Effect = "spectrum"
	EffectWave      Effect = "wave"
	EffectWheel     Effect = "wheel"
	EffectBreathing Effect = "breathing"
	EffectReactive  Effect = "reactive"
	EffectStarlight Effect = "starlight"
)

// RGB is an 8-bit colour.
type RGB struct{ R, G, B uint8 }

// EffectParams configures an effect. Unused fields are ignored.
type EffectParams struct {
	Effect    Effect
	Color     RGB
	Color2    RGB
	Mode      string // breathing/starlight: "single", "dual", "random"
	Speed     int    // reactive 1-4, starlight 1-3
	Direction string // wave, wheel: "left" or "right"
}

type report struct {
	class, id, size byte
	args            [80]byte
}

func newReport(class, id, size byte) *report {
	return &report{class: class, id: id, size: size}
}

func putRGB(dst []byte, c RGB) { dst[0], dst[1], dst[2] = c.R, c.G, c.B }

func clamp(v, lo, hi int) byte { return byte(min(max(v, lo), hi)) }

// conn is one open control collection. mu holds a command's write and its
// reply read together, so concurrent callers never read each other's
// replies from the device's single reply buffer.
type conn struct {
	dev   *hid.Device
	tid   byte
	polls int  // reply polls (2 ms apart) per attempt
	crc   bool // reject replies with a bad checksum
	mu    sync.Mutex
}

func (c *conn) pack(r *report, tid byte) []byte {
	b := make([]byte, reportLen+1) // [0] is the HID report ID
	p := b[1:]
	p[1] = tid
	p[5] = r.size
	p[6] = r.class
	p[7] = r.id
	copy(p[8:], r.args[:r.size])
	p[88] = checksum(p)
	return b
}

func checksum(p []byte) byte {
	var crc byte
	for i := 2; i < 88; i++ {
		crc ^= p[i]
	}
	return crc
}

// do sends a command and returns the 90-byte response payload.
func (c *conn) do(r *report) ([]byte, error) { return c.doTID(r, c.tid) }

func (c *conn) doTID(r *report, tid byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.exchange(r, tid)
}

// ErrUnsupported is returned for commands the device reports it doesn't
// implement.
var ErrUnsupported = errors.New("razer: command not supported by this device")

// exchange writes a command and waits for its reply. The reply buffer can
// also hold replies to other software (e.g. Synapse); if ours doesn't show
// up in time the command is resent. Caller holds c.mu.
func (c *conn) exchange(r *report, tid byte) ([]byte, error) {
	pkt := c.pack(r, tid)
	var last byte
	for range 4 {
		if err := c.dev.SetFeature(pkt); err != nil {
			return nil, err
		}
		for range c.polls {
			time.Sleep(2 * time.Millisecond)
			buf, err := c.dev.GetFeature(0)
			if err != nil {
				return nil, err
			}
			resp := buf[1:]
			if len(resp) < reportLen {
				continue
			}
			last = resp[0]
			if last == statusBusy || resp[1] != tid || resp[6] != r.class || resp[7] != r.id {
				continue // busy, or not our reply yet
			}
			if c.crc && resp[88] != checksum(resp) {
				continue // corrupt; wireless links do this while reconfiguring
			}
			switch last {
			case statusOK:
				return resp, nil
			case statusUnsupported:
				return nil, fmt.Errorf("%w (%02x:%02x)", ErrUnsupported, r.class, r.id)
			}
			return nil, fmt.Errorf("razer: command %02x:%02x failed (status %#x)", r.class, r.id, last)
		}
	}
	return nil, fmt.Errorf("razer: no reply to command %02x:%02x (status %#x)", r.class, r.id, last)
}

// openControl tries each candidate collection in order and returns the
// first that answers probe.
func openControl(cands []hid.Info, tid byte, polls int, crc bool, probe *report) (*conn, []byte, error) {
	for _, info := range cands {
		dev, err := hid.Open(info)
		if err != nil {
			continue
		}
		c := &conn{dev: dev, tid: tid, polls: polls, crc: crc}
		resp, err := c.do(probe)
		if err != nil {
			dev.Close()
			continue
		}
		return c, resp, nil
	}
	return nil, nil, errors.New("razer: device not responding")
}

// Keyboard is an open Razer keyboard.
type Keyboard struct {
	Model    Model
	Serial   string
	Firmware string
	c        *conn
}

// Supported reports whether pid is a known keyboard.
func Supported(pid uint16) bool { _, ok := keyboardModel(pid); return ok }

// Open finds the first supported keyboard.
func Open() (*Keyboard, error) {
	infos, err := hid.Enumerate(VendorID, 0)
	if err != nil {
		return nil, err
	}
	return OpenFrom(infos)
}

// OpenFrom opens the first supported keyboard among enumerated collections,
// pinning the collection that answers commands.
func OpenFrom(infos []hid.Info) (*Keyboard, error) {
	tried := map[uint16]bool{}
	for _, info := range infos {
		m, ok := keyboardModel(info.ProductID)
		if info.VendorID != VendorID || !ok || tried[m.PID] {
			continue
		}
		tried[m.PID] = true
		// The interface OpenRazer addresses comes first; the others are
		// fallbacks for boards whose firmware moved it.
		cands := controlCandidates(infos, m.PID)
		slices.SortStableFunc(cands, func(a, b hid.Info) int {
			return boolInt(b.Interface == m.Interface) - boolInt(a.Interface == m.Interface)
		})
		polls := 10
		if m.Wireless {
			polls = 50
		}
		c, resp, err := openControl(cands, m.TransactionID, polls, false, newReport(0x00, 0x82, 0x16))
		if err != nil {
			continue
		}
		kb := &Keyboard{Model: m, c: c, Serial: ascii(resp[8 : 8+0x16])}
		if fw, err := c.do(newReport(0x00, 0x81, 0x02)); err == nil {
			kb.Firmware = fmt.Sprintf("v%d.%d", fw[8], fw[9])
		}
		return kb, nil
	}
	return nil, errors.New("no supported Razer keyboard found")
}

// controlCandidates lists the collections of pid that carry the 90-byte
// feature report.
func controlCandidates(infos []hid.Info, pid uint16) []hid.Info {
	var out []hid.Info
	for _, i := range infos {
		if i.VendorID == VendorID && i.ProductID == pid && i.FeatureLen == reportLen+1 {
			out = append(out, i)
		}
	}
	return out
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Close releases the device.
func (k *Keyboard) Close() error { return k.c.dev.Close() }

func extended(size, store, led, effect byte) *report {
	r := newReport(0x0f, 0x02, size)
	r.args[0] = store
	r.args[1] = led
	r.args[2] = effect
	return r
}

func classic(size, effect byte) *report {
	r := newReport(0x03, 0x0a, size)
	r.args[0] = effect
	return r
}

// extendedEffect builds an extended-matrix effect for one LED. Keyboards and
// mice share the layout.
func extendedEffect(p EffectParams, led byte, wheel bool) (*report, error) {
	var r *report
	switch p.Effect {
	case EffectOff:
		r = extended(0x06, varStore, led, 0x00)
	case EffectStatic:
		r = extended(0x09, varStore, led, 0x01)
		r.args[5] = 0x01
		putRGB(r.args[6:], p.Color)
	case EffectSpectrum:
		r = extended(0x06, varStore, led, 0x03)
	case EffectWave:
		r = extended(0x06, varStore, led, 0x04)
		// Right is 1 everywhere. OpenRazer lists left as 0 for some older
		// boards, but they clamp it; their firmware wants 2 as well.
		r.args[3] = 0x01
		if p.Direction == "left" {
			r.args[3] = 0x02
		}
		r.args[4] = 0x28
	case EffectWheel:
		if !wheel {
			// Only the BlackWidow V4 family has the colour wheel; others get
			// the closest equivalent.
			return extendedEffect(EffectParams{Effect: EffectSpectrum}, led, wheel)
		}
		r = extended(0x06, varStore, led, 0x0a)
		r.args[3] = 0x01
		if p.Direction == "left" {
			r.args[3] = 0x02
		}
		r.args[4] = 0x28
	case EffectBreathing:
		switch p.Mode {
		case "random":
			r = extended(0x06, varStore, led, 0x02)
		case "dual":
			r = extended(0x0c, varStore, led, 0x02)
			r.args[3], r.args[5] = 0x02, 0x02
			putRGB(r.args[6:], p.Color)
			putRGB(r.args[9:], p.Color2)
		default:
			r = extended(0x09, varStore, led, 0x02)
			r.args[3], r.args[5] = 0x01, 0x01
			putRGB(r.args[6:], p.Color)
		}
	case EffectReactive:
		r = extended(0x09, varStore, led, 0x05)
		r.args[4] = clamp(p.Speed, 1, 4)
		r.args[5] = 0x01
		putRGB(r.args[6:], p.Color)
	case EffectStarlight:
		switch p.Mode {
		case "random":
			r = extended(0x06, varStore, led, 0x07)
		case "dual":
			r = extended(0x0c, varStore, led, 0x07)
			r.args[5] = 0x02
			putRGB(r.args[6:], p.Color)
			putRGB(r.args[9:], p.Color2)
		default:
			r = extended(0x09, varStore, led, 0x07)
			r.args[5] = 0x01
			putRGB(r.args[6:], p.Color)
		}
		r.args[4] = clamp(p.Speed, 1, 3)
	default:
		return nil, fmt.Errorf("razer: unknown effect %q", p.Effect)
	}
	return r, nil
}

func classicEffect(p EffectParams) (*report, error) {
	var r *report
	switch p.Effect {
	case EffectOff:
		r = classic(0x01, 0x00)
	case EffectStatic:
		r = classic(0x04, 0x06)
		putRGB(r.args[1:], p.Color)
	case EffectSpectrum, EffectWheel:
		r = classic(0x01, 0x04)
	case EffectWave:
		r = classic(0x02, 0x01)
		r.args[1] = 0x01
		if p.Direction == "left" {
			r.args[1] = 0x02
		}
	case EffectBreathing:
		r = classic(0x08, 0x03)
		switch p.Mode {
		case "random":
			r.args[1] = 0x03
		case "dual":
			r.args[1] = 0x02
			putRGB(r.args[2:], p.Color)
			putRGB(r.args[5:], p.Color2)
		default:
			r.args[1] = 0x01
			putRGB(r.args[2:], p.Color)
		}
	case EffectReactive:
		r = classic(0x05, 0x02)
		r.args[1] = clamp(p.Speed, 1, 4)
		putRGB(r.args[2:], p.Color)
	case EffectStarlight:
		switch p.Mode {
		case "random":
			r = classic(0x02, 0x19)
			r.args[1] = 0x03
		case "dual":
			r = classic(0x0c, 0x19)
			r.args[1] = 0x02
			putRGB(r.args[3:], p.Color)
			putRGB(r.args[6:], p.Color2)
		default:
			r = classic(0x09, 0x19)
			r.args[1] = 0x01
			putRGB(r.args[3:], p.Color)
		}
		r.args[2] = clamp(p.Speed, 1, 3)
	default:
		return nil, fmt.Errorf("razer: unknown effect %q", p.Effect)
	}
	return r, nil
}

// SetEffect applies a hardware effect, stored on the keyboard so it survives
// the app closing and power cycles.
func (k *Keyboard) SetEffect(p EffectParams) error {
	var r *report
	var err error
	if k.Model.Extended {
		r, err = extendedEffect(p, ledBacklight, k.Model.Wheel)
	} else {
		r, err = classicEffect(p)
	}
	if err != nil {
		return err
	}
	_, err = k.c.do(r)
	return err
}

// SetBrightness sets the backlight brightness (0-255).
func (k *Keyboard) SetBrightness(v uint8) error {
	_, err := k.c.do(k.Model.brightnessReport(true, v))
	return err
}

// Brightness reads the backlight brightness (0-255).
func (k *Keyboard) Brightness() (uint8, error) {
	resp, err := k.c.do(k.Model.brightnessReport(false, 0))
	if err != nil {
		return 0, err
	}
	return resp[8+2], nil
}

func (m Model) brightnessReport(set bool, v uint8) *report {
	var r *report
	switch {
	case m.Extended && set:
		r = newReport(0x0f, 0x04, 0x03)
	case m.Extended:
		r = newReport(0x0f, 0x84, 0x03)
	case set:
		r = newReport(0x03, 0x03, 0x03)
	default:
		r = newReport(0x03, 0x83, 0x03)
	}
	r.args[0], r.args[1], r.args[2] = varStore, ledBacklight, v
	return r
}

func (k *Keyboard) ledState(led byte) (bool, error) {
	r := newReport(0x03, 0x80, 0x03)
	r.args[0], r.args[1] = varStore, led
	resp, err := k.c.do(r)
	if err != nil {
		return false, err
	}
	return resp[8+2] == 1, nil
}

func (k *Keyboard) setLEDState(led byte, on bool) error {
	r := newReport(0x03, 0x00, 0x03)
	r.args[0], r.args[1], r.args[2] = varStore, led, byte(boolInt(on))
	_, err := k.c.do(r)
	return err
}

// GameMode reads whether game mode (Windows key disabled) is on.
func (k *Keyboard) GameMode() (bool, error) { return k.ledState(ledGame) }

// SetGameMode turns game mode on or off; it is stored on the keyboard.
func (k *Keyboard) SetGameMode(on bool) error { return k.setLEDState(ledGame, on) }

// MacroLEDs reads whether the macro key backlight is on.
func (k *Keyboard) MacroLEDs() (bool, error) { return k.ledState(ledMacro) }

// SetMacroLEDs turns the macro key backlight on or off.
func (k *Keyboard) SetMacroLEDs(on bool) error { return k.setLEDState(ledMacro, on) }

// Battery reads the charge (0-100 %) of a wireless keyboard.
func (k *Keyboard) Battery() (percent int, charging bool, err error) {
	if !k.Model.Battery {
		return 0, false, ErrUnsupported
	}
	return battery(k.c, k.Model.BatteryTID)
}

func battery(c *conn, tid byte) (int, bool, error) {
	level, err := c.doTID(newReport(0x07, 0x80, 0x02), tid)
	if err != nil {
		return 0, false, err
	}
	status, err := c.doTID(newReport(0x07, 0x84, 0x02), tid)
	return int(level[8+1]) * 100 / 255, err == nil && status[8+1] == 1, nil
}

// StartCustom switches the keyboard to host-driven custom frames (not
// stored, so a power cycle returns to the saved effect).
func (k *Keyboard) StartCustom() error {
	if !k.Model.Custom {
		return ErrUnsupported
	}
	_, err := k.c.do(k.Model.customReport())
	return err
}

func (m Model) customReport() *report {
	if !m.Extended {
		r := classic(0x02, 0x05)
		r.args[1] = noStore
		return r
	}
	r := newReport(0x0f, 0x02, 0x0c)
	r.args[2] = 0x08 // custom frame effect, NOSTORE / all LEDs
	return r
}

// WriteFrame sends one full custom frame, colour(row, col) for every LED.
// Each packet waits for the keyboard's acknowledgement (about 2 ms) so none
// is dropped while the firmware is busy.
func (k *Keyboard) WriteFrame(colour func(row, col int) RGB) error {
	k.c.mu.Lock()
	defer k.c.mu.Unlock()
	for row := range k.Model.Rows {
		for _, r := range k.Model.rowReports(row, colour) {
			if _, err := k.c.exchange(r, k.c.tid); err != nil {
				return err
			}
		}
	}
	return nil
}

// rowReports builds one matrix row; rows wider than 22 LEDs go out in
// chunks.
func (m Model) rowReports(row int, colour func(row, col int) RGB) []*report {
	var out []*report
	for start := 0; start < m.Cols; start += 22 {
		stop := min(start+21, m.Cols-1)
		var r *report
		off := 5
		if m.Extended {
			r = newReport(0x0f, 0x03, byte(max(0x47, (stop-start+1)*3+5)))
			r.args[2], r.args[3], r.args[4] = byte(row), byte(start), byte(stop)
		} else {
			r = newReport(0x03, 0x0b, 0x46)
			r.args[0], r.args[1], r.args[2], r.args[3] = 0xff, byte(row), byte(start), byte(stop)
			off = 4
		}
		for c := start; c <= stop; c++ {
			putRGB(r.args[off+(c-start)*3:], colour(row, c))
		}
		out = append(out, r)
	}
	return out
}

func ascii(b []byte) string {
	out := make([]byte, 0, len(b))
	for _, c := range b {
		if c >= 32 && c <= 126 {
			out = append(out, c)
		}
	}
	return string(out)
}
