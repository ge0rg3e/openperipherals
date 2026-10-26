// Package logitech drives Logitech G RGB keyboards over their 20-byte (0x11)
// output report, as documented by OpenRGB. The keyboards
// never report their lighting back, so everything here is write-only.
package logitech

import (
	"errors"
	"time"

	"openperipherals/internal/hid"
)

const (
	VendorID = 0x046d

	report20 = 0x11

	ioTimeout = 500 * time.Millisecond
)

// Family selects the packet layout a keyboard speaks.
type Family int

const (
	G213      Family = iota
	RomerG           // G512, G610, G810, G910, G Pro
	G910Spark        // G910 Orion Spark
	G815             // G813, G815
	G915             // G915 family (wired)
)

// Effect identifies a lighting effect.
type Effect string

const (
	EffectOff       Effect = "off"
	EffectStatic    Effect = "static"
	EffectSpectrum  Effect = "spectrum"
	EffectWave      Effect = "wave"
	EffectBreathing Effect = "breathing"
	EffectReactive  Effect = "reactive"
)

// Model describes a supported keyboard.
type Model struct {
	PID     uint16
	Name    string
	Family  Family
	Layout  string // "full" or "tkl"
	Effects []Effect
}

var (
	basic    = []Effect{EffectOff, EffectStatic, EffectSpectrum, EffectBreathing}
	withWave = []Effect{EffectOff, EffectStatic, EffectSpectrum, EffectWave, EffectBreathing}
	g915FX   = []Effect{EffectOff, EffectStatic, EffectSpectrum, EffectWave, EffectBreathing, EffectReactive}
)

// Models lists the supported keyboards.
var Models = []Model{
	{0xc336, "Logitech G213 Prodigy", G213, "full", withWave},
	{0xc342, "Logitech G512 Carbon", RomerG, "full", basic},
	{0xc33c, "Logitech G512 RGB", RomerG, "full", basic},
	{0xc333, "Logitech G610 Orion", RomerG, "full", basic},
	{0xc338, "Logitech G610 Orion", RomerG, "full", basic},
	{0xc331, "Logitech G810 Orion Spectrum", RomerG, "full", basic},
	{0xc337, "Logitech G810 Orion Spectrum", RomerG, "full", basic},
	{0xc32b, "Logitech G910 Orion Spark", G910Spark, "full", basic},
	{0xc335, "Logitech G910 Orion Spectrum", RomerG, "full", basic},
	{0xc339, "Logitech G Pro Keyboard", RomerG, "full", basic},
	{0xc232, "Logitech G813 RGB", G815, "full", basic},
	{0xc33f, "Logitech G815 RGB", G815, "full", basic},
	{0xc33e, "Logitech G915 Lightspeed", G915, "full", g915FX},
	{0xc343, "Logitech G915 TKL Lightspeed", G915, "tkl", g915FX},
}

// Supported reports whether pid is a known keyboard.
func Supported(pid uint16) bool { _, ok := model(pid); return ok }

func model(pid uint16) (Model, bool) {
	for _, m := range Models {
		if m.PID == pid {
			return m, true
		}
	}
	return Model{}, false
}

// RGB is an 8-bit colour.
type RGB struct{ R, G, B uint8 }

// EffectParams configures an effect.
type EffectParams struct {
	Effect    Effect
	Color     RGB
	Speed     int    // 1 (fast) to 4 (slow)
	Direction string // G213 wave: "left" or "right"
}

// Keyboard is an open Logitech keyboard. Effects go out on the collection
// that owns the 20-byte report 0x11 (the 64-byte 0x12 one only carries
// per-key frames).
type Keyboard struct {
	Model Model
	dev   *hid.Device
}

// OpenFrom opens the first supported keyboard among enumerated collections.
func OpenFrom(infos []hid.Info) (*Keyboard, error) {
	for _, info := range infos {
		m, ok := model(info.ProductID)
		if info.VendorID != VendorID || !ok || info.OutputLen != 20 {
			continue
		}
		if dev, err := hid.Open(info); err == nil {
			return &Keyboard{Model: m, dev: dev}, nil
		}
	}
	return nil, errors.New("no supported Logitech keyboard found")
}

// Close releases the device.
func (k *Keyboard) Close() error { return k.dev.Close() }

// send writes reports in order. The firmware can reject a write right after
// a reconnect and drops reports sent back to back, so each is retried once
// and followed by a short gap.
func (k *Keyboard) send(reports ...[]byte) error {
	for _, r := range reports {
		if err := k.dev.Write(r, ioTimeout); err != nil {
			time.Sleep(25 * time.Millisecond)
			if err := k.dev.Write(r, ioTimeout); err != nil {
				return err
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	return nil
}

// Hardware mode bytes.
const (
	modeOff       = 0x00
	modeStatic    = 0x01
	modeBreathing = 0x02
	modeCycle     = 0x03
	modeWave      = 0x04
	modeRipple    = 0x05

	g915LogoStatic    = 0x01
	g915LogoCycle     = 0x02
	g915LogoBreathing = 0x03
)

// mode maps an effect to the hardware mode, falling back to the closest
// effect a family has.
func (f Family) mode(e Effect) byte {
	switch e {
	case EffectOff:
		return modeOff
	case EffectSpectrum, "wheel", "starlight":
		return modeCycle
	case EffectWave:
		if f == G213 || f == G915 {
			return modeWave
		}
		return modeCycle
	case EffectReactive:
		if f == G915 {
			return modeRipple
		}
		return modeCycle
	case EffectBreathing:
		return modeBreathing
	}
	return modeStatic
}

// speedValue maps the 1-4 speed level to the firmware's period value.
func speedValue(level int) int {
	switch level {
	case 1:
		return 10
	case 3:
		return 128
	case 4:
		return 200
	}
	return 50
}

func pkt20(b ...byte) []byte {
	out := make([]byte, 20)
	out[0] = report20
	copy(out[1:], b)
	return out
}

// modeReport is the shared zone mode packet of the G213, RomerG and G815
// families. feature/cmd select the family.
func modeReport(feature, cmd, zone, mode byte, speed int, c RGB) []byte {
	b := pkt20(0xff, feature, cmd, zone, mode, c.R, c.G, c.B)
	v := speed * 100
	switch mode {
	case modeCycle:
		b[11], b[12], b[13] = byte(v>>8), byte(v), 0x64
	case modeBreathing:
		b[9], b[10], b[12] = byte(v>>8), byte(v), 0x64
	}
	return b
}

// SetEffect applies a hardware effect.
func (k *Keyboard) SetEffect(p EffectParams) error {
	return k.send(k.Model.Family.effectReports(p)...)
}

// effectReports builds the reports that apply an effect.
func (f Family) effectReports(p EffectParams) [][]byte {
	speed := speedValue(p.Speed)
	mode := f.mode(p.Effect)
	c := p.Color
	switch f {
	case G213:
		if mode == modeOff {
			var reps [][]byte
			for zone := byte(1); zone <= 5; zone++ {
				reps = append(reps, pkt20(0xff, 0x0c, 0x3a, zone, 0x01, 0, 0, 0, 0x02))
			}
			return reps
		}
		b := modeReport(0x0c, 0x3c, 0x00, mode, speed, c)
		if mode == modeWave {
			v := speed * 100
			b[11], b[12], b[15] = byte(v>>8), byte(v), byte(v>>8)
			b[13] = 0x01
			if p.Direction == "left" {
				b[13] = 0x06
			}
		}
		return [][]byte{b}
	case G915:
		return g915Effect(mode, speed, c)
	case G815:
		return [][]byte{modeReport(0x0d, 0x3d, 0x00, mode, speed, c), modeReport(0x0d, 0x3d, 0x01, mode, speed, c)}
	}
	feature, cmd, commitFeature, commitCmd := byte(0x0d), byte(0x3d), byte(0x0c), byte(0x5d)
	if f == G910Spark {
		feature, cmd, commitFeature, commitCmd = 0x10, 0x3b, 0x0f, 0x5f
	}
	return [][]byte{
		modeReport(feature, cmd, 0x00, mode, speed, c), // keys
		modeReport(feature, cmd, 0x01, mode, speed, c), // logo
		pkt20(0xff, commitFeature, commitCmd),
	}
}

// g915Effect builds the G915 sequence. The logo has its own, smaller set of
// modes: wave and cycle share "cycle", and ripple shows it static.
func g915Effect(mode byte, speed int, c RGB) [][]byte {
	logo := byte(g915LogoStatic)
	switch mode {
	case modeOff, modeStatic:
		logo = mode
	case modeBreathing:
		logo = g915LogoBreathing
	case modeCycle, modeWave:
		logo = g915LogoCycle
	}
	reps := [][]byte{pkt20(0xff, 0x0e, 0x3e), pkt20(0xff, 0x0e, 0x1e)}
	if mode != modeOff {
		reps = append(reps, g915Mode(0x01, mode, logo, speed, c))
	}
	return append(reps, g915Mode(0x00, logo, logo, speed, c))
}

func g915Mode(zone, mode, logo byte, speed int, c RGB) []byte {
	const keys, logoZone = 0x01, 0x00
	b := pkt20(0xff, 0x09, 0x1e, zone, mode, c.R, c.G, c.B)
	v := speed * 100
	if mode == modeRipple {
		v = speed
	}
	switch {
	case mode == modeStatic:
		b[9] = 0x02
	case mode == modeBreathing && zone == keys || logo == g915LogoBreathing && zone == logoZone:
		b[9], b[10] = byte(v>>8), byte(v)
	case mode == modeCycle && zone == keys || logo == g915LogoCycle && zone == logoZone:
		b[11], b[12] = byte(v>>8), byte(v)
	case mode == modeWave:
		b[12], b[13], b[15] = byte(v), 0x01, byte(v>>8)
	case mode == modeRipple:
		b[11] = byte(v)
	}
	b[16] = 0x01
	return b
}
