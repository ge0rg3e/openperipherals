package razer

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"openperipherals/internal/hid"
)

// MinDPI is the lowest DPI the mice accept.
const MinDPI = 100

// LiftOffs are the tracking distances, indexed by the byte the mouse uses.
var LiftOffs = []string{"Low", "Medium", "High"}

// SleepOptions are the auto-sleep timeouts in seconds (1 to 15 minutes, as
// in Synapse).
var SleepOptions = func() []int {
	out := make([]int, 15)
	for i := range out {
		out[i] = (i + 1) * 60
	}
	return out
}()

// Mouse is an open Razer mouse.
type Mouse struct {
	Model    MouseModel
	Firmware string
	c        *conn
}

// MouseSupported reports whether pid is a known mouse.
func MouseSupported(pid uint16) bool { _, ok := mouseModels[pid]; return ok }

// OpenMouseFrom opens the first supported mouse among enumerated
// collections. The control channel is usually the plain mouse collection;
// some revisions carry it on a vendor collection instead.
func OpenMouseFrom(infos []hid.Info) (*Mouse, error) {
	tried := map[uint16]bool{}
	for _, info := range infos {
		m, ok := mouseModels[info.ProductID]
		if info.VendorID != VendorID || !ok || tried[info.ProductID] {
			continue
		}
		tried[info.ProductID] = true
		m.PID = info.ProductID
		cands := controlCandidates(infos, m.PID)
		slices.SortStableFunc(cands, func(a, b hid.Info) int {
			mouse := func(i hid.Info) int { return boolInt(i.UsagePage == 0x01 && i.Usage == 0x02) }
			return mouse(b) - mouse(a)
		})
		polls := 25
		if m.Wireless {
			polls = 100 // the receiver relays each command over the air
		}
		c, resp, err := openControl(cands, m.TransactionID, polls, true, newReport(0x00, 0x81, 0x02))
		if err != nil {
			continue
		}
		return &Mouse{Model: m, c: c, Firmware: fmt.Sprintf("v%d.%d", resp[8], resp[9])}, nil
	}
	return nil, errors.New("no supported Razer mouse found")
}

// Close releases the device.
func (m *Mouse) Close() error { return m.c.dev.Close() }

// DPI reads the current DPI.
func (m *Mouse) DPI() (x, y int, err error) {
	r := newReport(0x04, 0x85, 0x07)
	r.args[0] = m.Model.DPIStore
	resp, err := m.c.do(r)
	if err != nil {
		return 0, 0, err
	}
	a := resp[8:]
	return int(a[1])<<8 | int(a[2]), int(a[3])<<8 | int(a[4]), nil
}

// SetDPI sets the DPI and confirms it by reading it back.
func (m *Mouse) SetDPI(x, y int) error {
	for _, v := range []int{x, y} {
		if v < MinDPI || v > m.Model.MaxDPI {
			return fmt.Errorf("DPI must be between %d and %d", MinDPI, m.Model.MaxDPI)
		}
	}
	if _, err := m.c.do(dpiReport(x, y, m.Model.DPIStore)); err != nil {
		return err
	}
	gx, gy, err := m.DPI()
	if err != nil {
		return err
	}
	if gx != x || gy != y {
		return fmt.Errorf("the mouse kept %d DPI instead of %d", gx, x)
	}
	return nil
}

// PollingRate reads the polling rate in Hz. Wired mice answer only the
// legacy command and receivers only the extended one, so the expected one
// is asked first.
func (m *Mouse) PollingRate() (int, error) {
	legacy := func() (int, error) {
		resp, err := m.c.do(newReport(0x00, 0x85, 0x01))
		if err != nil {
			return 0, err
		}
		if resp[8] == 0 {
			return 0, errors.New("razer: unknown polling rate")
		}
		return 1000 / int(resp[8]), nil
	}
	extended := func() (int, error) {
		resp, err := m.c.do(newReport(0x00, 0xc0, 0x02))
		if err != nil {
			return 0, err
		}
		if resp[9] == 0 {
			return 0, errors.New("razer: unknown polling rate")
		}
		return 8000 / int(resp[9]), nil
	}
	first, second := legacy, extended
	if m.Model.HighRatePolling {
		first, second = extended, legacy
	}
	if hz, err := first(); err == nil {
		return hz, nil
	}
	return second()
}

// SetPollingRate sets the polling rate and confirms it.
func (m *Mouse) SetPollingRate(hz int) error {
	if !slices.Contains(m.Model.PollingRates, hz) {
		return fmt.Errorf("this mouse does not support %d Hz", hz)
	}
	if _, err := m.c.do(pollingReport(hz, m.Model.HighRatePolling)); err != nil {
		return err
	}
	if m.Model.Wireless {
		// The link reconfigures; replies are garbage until it settles.
		time.Sleep(150 * time.Millisecond)
	}
	got, err := m.PollingRate()
	if err != nil {
		return err
	}
	if got != hz {
		return fmt.Errorf("the mouse kept %d Hz instead of %d Hz", got, hz)
	}
	return nil
}

// LiftOff reads the tracking distance (an index into LiftOffs).
func (m *Mouse) LiftOff() (int, error) {
	if !m.Model.LiftOff {
		return 0, ErrUnsupported
	}
	resp, err := m.c.do(newReport(0x0b, 0x85, 0x05))
	if err != nil {
		return 0, err
	}
	if int(resp[8+2]) >= len(LiftOffs) {
		return 0, errors.New("razer: unknown tracking distance")
	}
	return int(resp[8+2]), nil
}

// SetLiftOff sets the tracking distance and confirms it.
func (m *Mouse) SetLiftOff(level int) error {
	if !m.Model.LiftOff || level < 0 || level >= len(LiftOffs) {
		return ErrUnsupported
	}
	if _, err := m.c.do(liftOffReport(level)); err != nil {
		return err
	}
	if got, err := m.LiftOff(); err != nil || got != level {
		return fmt.Errorf("the mouse did not confirm the tracking distance")
	}
	return nil
}

// Sleep reads the auto-sleep timeout in seconds (wireless mice).
func (m *Mouse) Sleep() (int, error) {
	if !m.Model.Battery {
		return 0, ErrUnsupported
	}
	resp, err := m.c.do(newReport(0x07, 0x83, 0x02))
	if err != nil {
		return 0, err
	}
	return int(resp[8])<<8 | int(resp[9]), nil
}

// SetSleep sets the auto-sleep timeout in seconds and confirms it.
func (m *Mouse) SetSleep(seconds int) error {
	if !m.Model.Battery || !slices.Contains(SleepOptions, seconds) {
		return errors.New("auto sleep must be between 1 and 15 minutes")
	}
	if _, err := m.c.do(sleepReport(seconds)); err != nil {
		return err
	}
	if got, err := m.Sleep(); err != nil || got != seconds {
		return errors.New("the mouse did not confirm the sleep timeout")
	}
	return nil
}

// Battery reads the charge (0-100 %).
func (m *Mouse) Battery() (percent int, charging bool, err error) {
	if !m.Model.Battery {
		return 0, false, ErrUnsupported
	}
	return battery(m.c, m.c.tid)
}

// SetEffect writes an underglow effect (off, static, spectrum, reactive or
// breathing). The mouse stores it but can't report it back.
func (m *Mouse) SetEffect(p EffectParams) error {
	_, err := m.c.do(mouseEffect(p))
	return err
}

func mouseEffect(p EffectParams) *report {
	switch p.Effect {
	case EffectOff, EffectStatic, EffectSpectrum, EffectReactive, EffectBreathing:
	case EffectWave, EffectWheel:
		p.Effect = EffectSpectrum
	default:
		p.Effect = EffectStatic
	}
	r, _ := extendedEffect(p, ledLogo, false)
	return r
}

func dpiReport(x, y int, store byte) *report {
	r := newReport(0x04, 0x05, 0x07)
	r.args[0] = store
	r.args[1], r.args[2], r.args[3], r.args[4] = byte(x>>8), byte(x), byte(y>>8), byte(y)
	return r
}

// pollingReport encodes the rate as a divisor of 1000 (legacy) or 8000
// (extended, HyperPolling).
func pollingReport(hz int, extended bool) *report {
	if extended {
		r := newReport(0x00, 0x40, 0x02)
		r.args[1] = byte(8000 / hz)
		return r
	}
	r := newReport(0x00, 0x05, 0x01)
	r.args[0] = byte(1000 / hz)
	return r
}

func liftOffReport(level int) *report {
	r := newReport(0x0b, 0x0b, 0x04)
	r.args[1], r.args[2], r.args[3] = 0x04, 0x01, byte(level) // smart tracking: distance
	return r
}

func sleepReport(seconds int) *report {
	r := newReport(0x07, 0x03, 0x02)
	r.args[0], r.args[1] = byte(seconds>>8), byte(seconds)
	return r
}
