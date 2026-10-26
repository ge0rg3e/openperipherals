// Package redragon drives Redragon keyboards built on the EVision firmware
// (K550, K552, K556, K582, K587, K589, K512), using the 64-byte report 0x04
// documented by OpenRGB's EVisionKeyboardController. The keyboards never
// report their lighting back.
package redragon

import (
	"errors"
	"slices"
	"strings"
	"time"

	"openperipherals/internal/hid"
)

const (
	VendorID   = 0x0c45 // shared by many Sonix-based devices
	usagePage  = 0xff1c // lighting collection
	reportID   = 0x04
	reportSize = 64
	ioTimeout  = 500 * time.Millisecond

	cmdSetParameter = 0x06
	paramMode       = 0x00
)

// EVision mode bytes.
const (
	modeWaveLong  = 0x02
	modeSpectrum  = 0x04
	modeBreathing = 0x05
	modeStatic    = 0x06
	modeReactive  = 0x07
	modeStarlight = 0x0a
)

// Model describes a supported product ID. Several keyboards share one, so
// the name the device reports is preferred.
type Model struct {
	PID    uint16
	Name   string
	Layout string
}

// Models lists the supported product IDs.
var Models = []Model{
	{0x5004, "Redragon K556 / K587 / K582 / K589", "full"},
	{0x5104, "Redragon K552 Kumara", "tkl"},
	{0x5204, "Redragon K550 Yama", "full"},
	{0x672e, "Redragon K512", "full"},
}

// Effects lists the effects the firmware renders.
var Effects = []string{"off", "static", "spectrum", "wave", "breathing", "reactive", "starlight"}

// Supported reports whether pid is a known keyboard.
func Supported(pid uint16) bool {
	return slices.ContainsFunc(Models, func(m Model) bool { return m.PID == pid })
}

// RGB is an 8-bit colour.
type RGB struct{ R, G, B uint8 }

// EffectParams configures an effect.
type EffectParams struct {
	Effect     string
	Color      RGB
	Speed      int    // 1 (fast) to 4 (slow)
	Direction  string // wave: "left" or "right"
	Brightness int    // 0-4
}

// Keyboard is an open Redragon keyboard.
type Keyboard struct {
	Model Model
	Name  string
	dev   *hid.Device
}

// OpenFrom opens the lighting collection of the first supported keyboard,
// falling back to any collection that takes the 64-byte report.
func OpenFrom(infos []hid.Info) (*Keyboard, error) {
	infos = slices.Clone(infos)
	slices.SortStableFunc(infos, func(a, b hid.Info) int {
		return boolInt(b.UsagePage == usagePage) - boolInt(a.UsagePage == usagePage)
	})
	for _, m := range Models {
		for _, info := range infos {
			if info.VendorID != VendorID || info.ProductID != m.PID || info.OutputLen != reportSize {
				continue
			}
			dev, err := hid.Open(info)
			if err != nil {
				continue
			}
			name := strings.TrimSpace(info.Product)
			if name == "" {
				name = m.Name
			}
			return &Keyboard{Model: m, Name: name, dev: dev}, nil
		}
	}
	return nil, errors.New("no supported Redragon keyboard found")
}

// Close releases the device.
func (k *Keyboard) Close() error { return k.dev.Close() }

// SetEffect applies a hardware effect in one mode packet, which also carries
// the brightness.
func (k *Keyboard) SetEffect(p EffectParams) error {
	buf := modePacket(p)
	// A write can be rejected right after a reconnect; retry once.
	err := k.dev.Write(buf, ioTimeout)
	if err != nil {
		time.Sleep(25 * time.Millisecond)
		err = k.dev.Write(buf, ioTimeout)
	}
	return err
}

func modePacket(p EffectParams) []byte {
	mode, random := byte(modeStatic), false
	switch p.Effect {
	case "spectrum":
		mode = modeSpectrum
	case "wave":
		mode, random = modeWaveLong, true
	case "breathing":
		mode = modeBreathing
	case "reactive":
		mode = modeReactive
	case "starlight":
		mode = modeStarlight
	}
	speed := byte(0x03) // normal
	switch p.Effect {
	case "wave", "breathing", "reactive", "starlight":
		speed = speedByte(p.Speed)
	}
	brightness := byte(min(max(p.Brightness, 0), 4))
	if p.Effect == "off" {
		brightness = 0 // no off mode: static at brightness 0
	}
	dir := byte(0x01)
	if p.Direction == "left" {
		dir = 0x00
	}
	data := []byte{mode, brightness, speed, dir, byte(boolInt(random)), p.Color.R, p.Color.G, p.Color.B}

	buf := make([]byte, reportSize)
	buf[0] = reportID
	buf[3] = cmdSetParameter
	buf[4] = byte(len(data))
	buf[5] = paramMode
	copy(buf[8:], data)
	var sum int
	for _, b := range buf[3:] {
		sum += int(b)
	}
	buf[1], buf[2] = byte(sum), byte(sum>>8)
	return buf
}

// speedByte maps the 1-4 speed level to EVision's 0 (fastest) - 5 (slowest).
func speedByte(level int) byte {
	switch level {
	case 1:
		return 0x00
	case 2:
		return 0x02
	case 4:
		return 0x05
	}
	return 0x03
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
