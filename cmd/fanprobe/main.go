// Command fanprobe reads (never writes) fan speeds, duty cycles and
// temperatures from the Super I/O chip and NVIDIA GPUs.
package main

import (
	"fmt"

	"openperipherals/internal/nvgpu"
	"openperipherals/internal/superio"
)

func main() {
	gpus, err := nvgpu.Enumerate()
	fmt.Println("nvidia:", len(gpus), err)
	for _, g := range gpus {
		t, terr := g.Temperature()
		rpm, rerr := g.RPM()
		duty, derr := g.Duty()
		fmt.Printf("  %s coolers=%d min=%v temp=%d(%v) rpm=%d(%v) duty=%v(%v)\n", g.Name, g.Coolers, g.MinDuty, t, terr, rpm, rerr, duty, derr)
	}
	chip, err := superio.Detect()
	if err != nil {
		fmt.Println("superio:", err)
		return
	}
	r, err := chip.Read()
	fmt.Printf("superio: %s controls=%d\n  rpm=%v\n  temps=%v\n  duty=%v err=%v\n", chip.Name, chip.Controls, r.FanRPM, r.Temps, r.Duty, err)
	chip.Close()
}
