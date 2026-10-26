// Command probe dumps HID collections and reads (never writes) device state.
package main

import (
	"fmt"

	"openperipherals/internal/corsair"
	"openperipherals/internal/hid"
	"openperipherals/internal/logitech"
	"openperipherals/internal/pulsar"
	"openperipherals/internal/razer"
	"openperipherals/internal/redragon"
)

func main() {
	vendors := map[uint16]bool{
		razer.VendorID: true, corsair.VendorID: true, logitech.VendorID: true,
		redragon.VendorID: true, pulsar.VendorID: true, pulsar.VGNVendorID: true,
	}
	infos, err := hid.EnumerateFunc(func(vid, _ uint16) bool { return vendors[vid] })
	if err != nil {
		fmt.Println("enumerate:", err)
	}
	for _, i := range infos {
		fmt.Printf("%04x:%04x mi=%d page=%04x usage=%04x in=%d out=%d feat=%d %q\n",
			i.VendorID, i.ProductID, i.Interface, i.UsagePage, i.Usage, i.InputLen, i.OutputLen, i.FeatureLen, i.Product)
	}

	if kb, err := razer.OpenFrom(infos); err != nil {
		fmt.Println("razer keyboard:", err)
	} else {
		b, err := kb.Brightness()
		fmt.Printf("razer keyboard: %s serial=%q fw=%s brightness=%d err=%v\n", kb.Model.Name, kb.Serial, kb.Firmware, b, err)
		kb.Close()
	}
	if m, err := razer.OpenMouseFrom(infos); err != nil {
		fmt.Println("razer mouse:", err)
	} else {
		x, y, err := m.DPI()
		hz, perr := m.PollingRate()
		fmt.Printf("razer mouse: %s fw=%s dpi=%dx%d (%v) polling=%d (%v)\n", m.Model.Name, m.Firmware, x, y, err, hz, perr)
		m.Close()
	}
	if m, err := corsair.OpenFrom(infos); err != nil {
		fmt.Println("corsair:", err)
	} else {
		fmt.Printf("corsair: %s fw=%s\n", m.Model.Name, m.Firmware)
		cfg, err := m.DPI()
		fmt.Printf("  dpi: %+v err=%v\n", cfg, err)
		z, err := m.Colors()
		fmt.Printf("  colors: %+v err=%v\n", z, err)
		m.Close()
	}
	if m, err := pulsar.OpenFrom(infos); err != nil {
		fmt.Println("pulsar:", err)
	} else {
		st, err := m.Status()
		fmt.Printf("pulsar: %s wired=%v max=%d Hz status=%+v err=%v\n", m.Name, m.Wired, m.MaxPolling, st, err)
		m.Close()
	}
	// Logitech and Redragon keyboards are write-only: opening is all a
	// read-only probe can do.
	if kb, err := logitech.OpenFrom(infos); err != nil {
		fmt.Println("logitech:", err)
	} else {
		fmt.Printf("logitech: %s\n", kb.Model.Name)
		kb.Close()
	}
	if kb, err := redragon.OpenFrom(infos); err != nil {
		fmt.Println("redragon:", err)
	} else {
		fmt.Printf("redragon: %s\n", kb.Name)
		kb.Close()
	}
}
