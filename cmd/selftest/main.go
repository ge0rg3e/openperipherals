// Command selftest writes back each device's current settings unchanged and
// verifies the readback, exercising the write paths without visible changes.
package main

import (
	"fmt"
	"os"

	"openperipherals/internal/corsair"
	"openperipherals/internal/razer"
)

func check(what string, err error) {
	if err != nil {
		fmt.Println("FAIL", what, err)
		os.Exit(1)
	}
	fmt.Println("ok  ", what)
}

func main() {
	m, err := corsair.Open()
	check("corsair open", err)
	before, err := m.DPI()
	check("corsair read dpi", err)
	check("corsair write dpi", m.SetDPI(before))
	after, err := m.DPI()
	check("corsair reread dpi", err)
	if before != after {
		check("corsair dpi roundtrip", fmt.Errorf("\n %+v\n %+v", before, after))
	}
	zones, err := m.Colors()
	check("corsair read colors", err)
	check("corsair save colors", m.SaveColors(zones))
	z2, err := m.Colors()
	check("corsair reread colors", err)
	if zones != z2 {
		check("corsair color roundtrip", fmt.Errorf("%v != %v", zones, z2))
	}
	m.Close()

	kb, err := razer.Open()
	check("razer open", err)
	b, err := kb.Brightness()
	check("razer read brightness", err)
	check("razer write brightness", kb.SetBrightness(b))
	b2, err := kb.Brightness()
	check(fmt.Sprintf("razer brightness roundtrip %d -> %d", b, b2), err)
	kb.Close()
}
