// Package corsair drives Corsair NXP-era mice (SABRE RGB) over the 64-byte
// vendor protocol documented by ckb-next and OpenRGB.
package corsair

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"openperipherals/internal/hid"
)

const (
	VendorID = 0x1b1c

	cmdSet = 0x07
	cmdGet = 0x0e

	fieldIdent    = 0x01
	fieldLighting = 0x05
	fieldMouse    = 0x13
	fieldMColor   = 0x22

	mouseDPI     = 0x02
	mouseDPIMask = 0x05
	mouseHWColor = 0x10
	mouseDPIProf = 0xd0

	lightingHardware = 0x01
	lightingSoftware = 0x02

	// Byte 3 of a mouse field packet selects live state (0) or the saved
	// hardware profile (1).
	live  = 0x00
	saved = 0x01

	ioTimeout = 500 * time.Millisecond
)

// StageCount is the number of DPI stages the firmware stores. Stage 0 is the
// sniper stage on mice that have a sniper button.
const StageCount = 6

// Zone IDs used by the live colour packet.
const (
	ZoneUnderglow = iota
	ZoneLogo
	ZoneDPI
	ZoneWheel
	ZoneCount
)

// ZoneNames are the user-facing names, indexed by zone.
var ZoneNames = [ZoneCount]string{"Underglow", "Logo", "DPI indicator", "Scroll wheel"}

// Model describes a supported mouse.
type Model struct {
	PID    uint16
	Name   string
	MaxDPI int
}

// Models lists the supported mice.
var Models = []Model{
	{PID: 0x1b2f, Name: "Corsair SABRE RGB", MaxDPI: 10000},
	{PID: 0x1b32, Name: "Corsair SABRE RGB Optical", MaxDPI: 6400},
	{PID: 0x1b14, Name: "Corsair SABRE RGB Optical", MaxDPI: 6400},
	{PID: 0x1b19, Name: "Corsair SABRE RGB Laser", MaxDPI: 8200},
}

// RGB is an 8-bit colour.
type RGB struct{ R, G, B uint8 }

// Stage is one DPI stage.
type Stage struct {
	DPI     int
	Enabled bool
	Color   RGB // DPI indicator colour while this stage is active
}

// DPIConfig is the full DPI configuration.
type DPIConfig struct {
	Stages  [StageCount]Stage
	Current int
}

// Mouse is an open SABRE.
type Mouse struct {
	Model    Model
	Firmware string
	dev      *hid.Device
	reply    *hid.Device
	getMu    sync.Mutex
}

// Open finds the first supported mouse. Commands go out on the interface 1
// vendor collection; replies come back on the interface 0 vendor collection.
func Open() (*Mouse, error) {
	infos, err := hid.Enumerate(VendorID, 0)
	if err != nil {
		return nil, err
	}
	return OpenFrom(infos)
}

// Supported reports whether pid is a known mouse.
func Supported(pid uint16) bool {
	for _, m := range Models {
		if m.PID == pid {
			return true
		}
	}
	return false
}

// OpenFrom opens the first supported mouse among enumerated collections.
func OpenFrom(infos []hid.Info) (*Mouse, error) {
	for _, m := range Models {
		var ctl, in *hid.Info
		for i := range infos {
			info := &infos[i]
			if info.VendorID != VendorID || info.ProductID != m.PID || info.UsagePage != 0xffc2 {
				continue
			}
			switch {
			case info.OutputLen >= 65:
				ctl = info
			case info.InputLen >= 64 && info.Interface == 0:
				in = info
			}
		}
		if ctl == nil || in == nil {
			continue
		}
		dev, err := hid.Open(*ctl)
		if err != nil {
			return nil, err
		}
		reply, err := hid.Open(*in)
		if err != nil {
			dev.Close()
			return nil, err
		}
		ms := &Mouse{Model: m, dev: dev, reply: reply}
		resp, err := ms.get(fieldIdent)
		if err != nil {
			ms.Close()
			return nil, fmt.Errorf("corsair: mouse not responding: %w", err)
		}
		ms.Firmware = fmt.Sprintf("%x.%02x", resp[9], resp[8])
		return ms, nil
	}
	return nil, errors.New("no supported Corsair mouse found")
}

// Close releases the device.
func (m *Mouse) Close() error {
	m.reply.Close()
	return m.dev.Close()
}

func (m *Mouse) send(pkt ...byte) error {
	buf := make([]byte, 65)
	copy(buf[1:], pkt)
	return m.dev.Write(buf, ioTimeout)
}

// get issues a read command and returns the 64-byte reply whose header
// echoes the request, skipping unrelated input reports.
func (m *Mouse) get(pkt ...byte) ([]byte, error) {
	m.getMu.Lock()
	defer m.getMu.Unlock()
	req := append([]byte{cmdGet}, pkt...)
	if err := m.send(req...); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(ioTimeout)
	for time.Now().Before(deadline) {
		buf, err := m.reply.Read(time.Until(deadline))
		if err != nil {
			return nil, err
		}
		if len(buf) >= 12 && matches(buf, req) {
			return buf, nil
		}
	}
	return nil, hid.ErrTimeout
}

func matches(resp, req []byte) bool {
	n := min(len(req), 4)
	for i := range n {
		if resp[i] != req[i] {
			return false
		}
	}
	return true
}

// SetLiveColors switches lighting to software control and shows colours
// immediately. They are lost when the mouse loses power or returns to
// hardware mode.
func (m *Mouse) SetLiveColors(zones [ZoneCount]RGB) error {
	if err := m.send(cmdSet, fieldLighting, lightingSoftware, 0x00, 0x01); err != nil {
		return err
	}
	pkt := []byte{cmdSet, fieldMColor, ZoneCount, 0x01}
	for i, c := range zones {
		pkt = append(pkt, byte(i+1), c.R, c.G, c.B)
	}
	return m.send(pkt...)
}

// LiveFrame updates the colours shown in software mode without re-sending
// the mode switch; call SetLiveColors once first. Used to stream animations.
func (m *Mouse) LiveFrame(zones [ZoneCount]RGB) error {
	pkt := []byte{cmdSet, fieldMColor, ZoneCount, 0x01}
	for i, c := range zones {
		pkt = append(pkt, byte(i+1), c.R, c.G, c.B)
	}
	return m.send(pkt...)
}

// HardwareMode hands lighting back to the firmware, which shows the colours
// saved in the hardware profile.
func (m *Mouse) HardwareMode() error {
	return m.send(cmdSet, fieldLighting, lightingHardware, 0x00, 0x01)
}

// SaveColors stores the zone colours in the mouse's hardware profile and
// hands lighting back to the firmware, so they persist without the app. The
// DPI indicator follows the per-stage colours in the DPI config.
func (m *Mouse) SaveColors(zones [ZoneCount]RGB) error {
	slot := byte(mouseHWColor)
	for i, c := range zones {
		if i == ZoneDPI {
			continue
		}
		if err := m.send(cmdSet, fieldMouse, slot, saved, c.R, c.G, c.B); err != nil {
			return err
		}
		slot++
	}
	// Already in hardware mode the firmware keeps showing the old colours,
	// so bounce through software mode to make it reload the saved ones.
	if err := m.send(cmdSet, fieldLighting, lightingSoftware, 0x00, 0x01); err != nil {
		return err
	}
	return m.send(cmdSet, fieldLighting, lightingHardware, 0x00, 0x01)
}

// StoreColors writes the zone colours to the hardware profile only, without
// changing what is shown. They take effect in hardware mode (e.g. after
// the app quits, or at power-up).
func (m *Mouse) StoreColors(zones [ZoneCount]RGB) error {
	slot := byte(mouseHWColor)
	for i, c := range zones {
		if i == ZoneDPI {
			continue
		}
		if err := m.send(cmdSet, fieldMouse, slot, saved, c.R, c.G, c.B); err != nil {
			return err
		}
		slot++
	}
	return nil
}

// Colors reads the zone colours saved in the hardware profile. The DPI zone
// is left zero; its colour lives in each DPI stage.
func (m *Mouse) Colors() ([ZoneCount]RGB, error) {
	var zones [ZoneCount]RGB
	slot := byte(mouseHWColor)
	for i := range zones {
		if i == ZoneDPI {
			continue
		}
		resp, err := m.get(fieldMouse, slot, saved)
		if err != nil {
			return zones, err
		}
		zones[i] = RGB{resp[4], resp[5], resp[6]}
		slot++
	}
	return zones, nil
}

// CurrentStage reads the active DPI stage as the mouse is running now
// (it changes when the DPI button is pressed).
func (m *Mouse) CurrentStage() (int, error) {
	resp, err := m.get(fieldMouse, mouseDPI, live)
	if err != nil {
		return 0, err
	}
	return int(resp[4]), nil
}

// DPI reads the saved DPI configuration.
func (m *Mouse) DPI() (DPIConfig, error) {
	var cfg DPIConfig
	mask, err := m.get(fieldMouse, mouseDPIMask, saved)
	if err != nil {
		return cfg, err
	}
	cur, err := m.get(fieldMouse, mouseDPI, saved)
	if err != nil {
		return cfg, err
	}
	cfg.Current = int(cur[4])
	if cfg.Current >= StageCount {
		cfg.Current = 1
	}
	for i := range StageCount {
		resp, err := m.get(fieldMouse, mouseDPIProf|byte(i), saved)
		if err != nil {
			return cfg, err
		}
		cfg.Stages[i] = Stage{
			DPI:     int(resp[5]) | int(resp[6])<<8,
			Enabled: mask[4]&(1<<i) != 0,
			Color:   RGB{resp[9], resp[10], resp[11]},
		}
	}
	return cfg, nil
}

// SetDPI applies the DPI configuration live and saves it to the hardware
// profile.
func (m *Mouse) SetDPI(cfg DPIConfig) error {
	var mask byte
	for i := range cfg.Stages {
		s := &cfg.Stages[i]
		s.DPI = min(max(s.DPI, 50), m.Model.MaxDPI) / 50 * 50
		if s.Enabled {
			mask |= 1 << i
		}
	}
	if cfg.Current < 0 || cfg.Current >= StageCount || !cfg.Stages[cfg.Current].Enabled {
		return errors.New("corsair: active DPI stage must be enabled")
	}
	for _, store := range []byte{saved, live} {
		for i, s := range cfg.Stages {
			lo, hi := byte(s.DPI), byte(s.DPI>>8)
			if err := m.send(cmdSet, fieldMouse, mouseDPIProf|byte(i), store,
				0x00, lo, hi, lo, hi, s.Color.R, s.Color.G, s.Color.B); err != nil {
				return err
			}
		}
		if err := m.send(cmdSet, fieldMouse, mouseDPIMask, store, mask); err != nil {
			return err
		}
		if err := m.send(cmdSet, fieldMouse, mouseDPI, store, byte(cfg.Current)); err != nil {
			return err
		}
	}
	return nil
}
