// Package superio reads and controls motherboard fan headers on ITE IT86xx /
// IT87xx Super I/O chips through PawnIO's LpcIO module. Register layout and
// detection follow LibreHardwareMonitor's IT87XX implementation.
package superio

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"openperipherals/internal/pawnio"
)

const (
	registerPort = 0x2e
	valuePort    = 0x2f

	chipIDReg       = 0x20
	deviceSelectReg = 0x07
	baseAddrReg     = 0x60
	configCtrlReg   = 0x02
	envControllerLD = 0x04

	addrOffset = 0x05
	dataOffset = 0x06

	configReg      = 0x00
	vendorIDReg    = 0x58
	tach16BitReg   = 0x0c
	fanMainCtrlReg = 0x13
	tempBaseReg    = 0x29
)

var (
	tachReg    = [6]byte{0x0d, 0x0e, 0x0f, 0x80, 0x82, 0x4c}
	tachExtReg = [6]byte{0x18, 0x19, 0x1a, 0x81, 0x83, 0x4d}
	pwmReg     = [6]byte{0x15, 0x16, 0x17, 0x7f, 0xa7, 0xaf}
	pwmExtReg  = [6]byte{0x63, 0x6b, 0x73, 0x7b, 0xa3, 0xab}
)

// chipSpec covers ITE chips with 16-bit tach counters and 8-bit extended PWM
// registers, which all share the register layout above.
type chipSpec struct {
	name                 string
	fans, controls, temp int
}

var chips = map[uint16]chipSpec{
	0x8613: {"ITE IT8613E", 5, 4, 4},
	0x8620: {"ITE IT8620E", 5, 5, 3},
	0x8628: {"ITE IT8628E", 6, 6, 6},
	0x8631: {"ITE IT8631E", 2, 2, 2},
	0x8638: {"ITE IT8638E", 2, 2, 2},
	0x8686: {"ITE IT8686E", 6, 5, 7},
	0x8688: {"ITE IT8688E", 6, 5, 6},
	0x8696: {"ITE IT8696E", 6, 6, 6},
	0x8721: {"ITE IT8721F", 5, 3, 3},
	0x8728: {"ITE IT8728F", 5, 3, 3},
	0x8772: {"ITE IT8772E", 5, 3, 3},
}

// Readings is one snapshot of the chip. Missing values are -1.
type Readings struct {
	FanRPM []int // per tach input; -1 if the input is disabled
	Temps  []int // °C; -1 if unconnected
	Duty   []int // per control, 0-100; -1 while the BIOS controls it
}

type saved struct {
	valid    bool
	pwm, ext byte
	mainBit  bool
}

// Chip is a detected ITE Super I/O.
type Chip struct {
	Name     string
	Controls int
	spec     chipSpec

	mu         sync.Mutex
	mod        *pawnio.Module
	bus        *pawnio.IsaBus
	addr, data uint64
	fanOff     [6]bool
	saved      [6]saved
}

// Detect looks for a supported ITE chip at the primary Super I/O slot.
func Detect() (*Chip, error) {
	mod, err := pawnio.Load(pawnio.LpcIO)
	if err != nil {
		return nil, err
	}
	bus, err := pawnio.OpenIsaBus()
	if err != nil {
		mod.Close()
		return nil, err
	}
	c := &Chip{mod: mod, bus: bus}
	if err := c.detect(); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

func (c *Chip) detect() error {
	if err := c.bus.Lock(); err != nil {
		return err
	}
	defer c.bus.Unlock()

	if _, err := c.mod.Execute("ioctl_select_slot", []uint64{0}, 0); err != nil {
		return err
	}
	for _, b := range []uint64{0x87, 0x01, 0x55, 0x55} { // enter config mode
		if err := c.outb(registerPort, b); err != nil {
			return err
		}
	}
	id := uint16(c.sioIn(chipIDReg))<<8 | uint16(c.sioIn(chipIDReg+1))
	spec, ok := chips[id]
	if !ok {
		c.exitConfig()
		if id == 0 || id == 0xffff {
			return errors.New("no ITE Super I/O chip found")
		}
		return fmt.Errorf("unsupported Super I/O chip %#04x", id)
	}
	if _, err := c.mod.Execute("ioctl_find_bars", nil, 0); err != nil {
		c.exitConfig()
		return err
	}
	c.sioOut(deviceSelectReg, envControllerLD)
	base := c.sioWord(baseAddrReg)
	time.Sleep(time.Millisecond)
	verify := c.sioWord(baseAddrReg)
	c.exitConfig()
	if base != verify || base < 0x100 || base&0xf007 != 0 {
		return fmt.Errorf("%s: invalid base address %#x", spec.name, base)
	}
	c.addr, c.data = uint64(base+addrOffset), uint64(base+dataOffset)

	vendor, ok1 := c.read(vendorIDReg)
	cfg, ok2 := c.read(configReg)
	if !ok1 || !ok2 || (vendor != 0x90 && vendor != 0x7f) || cfg&0x10 == 0 {
		return fmt.Errorf("%s: environment controller not responding", spec.name)
	}
	modes, _ := c.read(tach16BitReg)
	c.fanOff[3] = modes&(1<<4) == 0
	c.fanOff[4] = modes&(1<<5) == 0
	c.fanOff[5] = modes&(1<<2) == 0

	c.Name, c.Controls, c.spec = spec.name, spec.controls, spec
	return nil
}

// Close hands every fan back to the BIOS and releases the driver.
func (c *Chip) Close() {
	if c.addr != 0 {
		for i := range c.spec.controls {
			c.Restore(i)
		}
	}
	c.bus.Close()
	c.mod.Close()
}

// Read takes a snapshot of fan speeds, temperatures and duty cycles.
func (c *Chip) Read() (Readings, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.bus.Lock(); err != nil {
		return Readings{}, err
	}
	defer c.bus.Unlock()

	r := Readings{
		FanRPM: make([]int, c.spec.fans),
		Temps:  make([]int, c.spec.temp),
		Duty:   make([]int, c.spec.controls),
	}
	for i := range r.FanRPM {
		r.FanRPM[i] = -1
		if c.fanOff[i] {
			continue
		}
		lo, ok1 := c.read(tachReg[i])
		hi, ok2 := c.read(tachExtReg[i])
		if !ok1 || !ok2 {
			continue
		}
		switch v := int(lo) | int(hi)<<8; {
		case v <= 0x3f:
		case v == 0xffff:
			r.FanRPM[i] = 0
		default:
			r.FanRPM[i] = 1350000 / (v * 2)
		}
	}
	for i := range r.Temps {
		r.Temps[i] = -1
		if v, ok := c.read(tempBaseReg + byte(i)); ok && int8(v) > 0 && int8(v) < 127 {
			r.Temps[i] = int(int8(v))
		}
	}
	for i := range r.Duty {
		r.Duty[i] = -1
		if v, ok := c.read(pwmReg[i]); ok && v&0x80 == 0 {
			if ext, ok := c.read(pwmExtReg[i]); ok {
				r.Duty[i] = (int(ext)*100 + 127) / 255
			}
		}
	}
	return r, nil
}

// SetDuty takes control of a fan header from the BIOS and sets its duty
// cycle (0-100 %).
func (c *Chip) SetDuty(i, percent int) error {
	if i < 0 || i >= c.spec.controls {
		return fmt.Errorf("fan control %d out of range", i)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.bus.Lock(); err != nil {
		return err
	}
	defer c.bus.Unlock()

	s := &c.saved[i]
	if !s.valid {
		s.pwm, _ = c.read(pwmReg[i])
		s.ext, _ = c.read(pwmExtReg[i])
		if i < 3 {
			main, _ := c.read(fanMainCtrlReg)
			s.mainBit = main&(1<<i) != 0
		}
		s.valid = true
	}
	if i < 3 && !s.mainBit {
		main, _ := c.read(fanMainCtrlReg)
		c.write(fanMainCtrlReg, main|1<<i)
	}
	c.write(pwmReg[i], s.pwm&0x7f) // clear the automatic (SmartGuardian) bit
	c.write(pwmExtReg[i], byte(min(max(percent, 0), 100)*255/100))
	return nil
}

// Restore returns a fan header to the BIOS settings saved before SetDuty.
func (c *Chip) Restore(i int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := &c.saved[i]
	if !s.valid {
		return nil
	}
	if err := c.bus.Lock(); err != nil {
		return err
	}
	defer c.bus.Unlock()
	c.write(pwmReg[i], s.pwm)
	if i < 3 {
		main, _ := c.read(fanMainCtrlReg)
		if (main&(1<<i) != 0) != s.mainBit {
			c.write(fanMainCtrlReg, main^1<<i)
		}
	}
	c.write(pwmExtReg[i], s.ext)
	s.valid = false
	return nil
}

// Port-level helpers. Errors from the driver surface as invalid reads, which
// callers already handle.

func (c *Chip) outb(port, v uint64) error {
	_, err := c.mod.Execute("ioctl_pio_outb", []uint64{port, v}, 0)
	return err
}

func (c *Chip) inb(port uint64) byte {
	out, err := c.mod.Execute("ioctl_pio_inb", []uint64{port}, 1)
	if err != nil {
		return 0xff
	}
	return byte(out[0])
}

func (c *Chip) sioIn(reg uint64) byte {
	out, err := c.mod.Execute("ioctl_superio_inb", []uint64{reg}, 1)
	if err != nil {
		return 0xff
	}
	return byte(out[0])
}

func (c *Chip) sioWord(reg uint64) uint16 {
	return uint16(c.sioIn(reg))<<8 | uint16(c.sioIn(reg+1))
}

func (c *Chip) sioOut(reg, v uint64) {
	c.mod.Execute("ioctl_superio_outb", []uint64{reg, v}, 0)
}

func (c *Chip) exitConfig() {
	c.outb(registerPort, configCtrlReg)
	c.outb(valuePort, 0x02)
}

func (c *Chip) read(reg byte) (byte, bool) {
	c.outb(c.addr, uint64(reg))
	v := c.inb(c.data)
	return v, c.inb(c.addr) == reg
}

func (c *Chip) write(reg, v byte) {
	c.outb(c.addr, uint64(reg))
	c.outb(c.data, uint64(v))
	c.inb(c.addr)
}

// Labels returns header and sensor names for boards whose wiring is known,
// or nil maps to fall back on generic names.
func Labels(board string) (fans, temps map[int]string) {
	b := strings.ToUpper(strings.TrimSpace(board))
	switch {
	case strings.HasPrefix(b, "B450M DS3H"), strings.HasPrefix(b, "B450M S2H"), strings.HasPrefix(b, "B450M H"):
		return map[int]string{0: "CPU Fan", 1: "System Fan"},
			map[int]string{0: "System", 1: "Chipset", 2: "CPU", 4: "VRM MOS", 5: "VSoC MOS"}
	}
	return nil, nil
}
