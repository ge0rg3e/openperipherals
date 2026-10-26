// Command ccprobe prints media key presses from every Razer consumer-control
// collection for 15 seconds, to check what the keyboard's dial and media keys
// report (and from which collection).
package main

import (
	"fmt"
	"sync"
	"time"

	"openperipherals/internal/hid"
)

var names = map[uint16]string{
	0xe9: "Volume Up", 0xea: "Volume Down", 0xe2: "Mute",
	0xcd: "Play/Pause", 0xb5: "Next", 0xb6: "Previous", 0xb7: "Stop",
}

func main() {
	var wg sync.WaitGroup
	infos, _ := hid.Enumerate(0x1532, 0)
	deadline := time.Now().Add(15 * time.Second)
	for _, i := range infos {
		if i.UsagePage != 0x0c {
			continue
		}
		d, err := hid.Open(i)
		if err != nil {
			fmt.Printf("%04x mi=%d: %v\n", i.ProductID, i.Interface, err)
			continue
		}
		fmt.Printf("watching %04x interface %d (%s)\n", i.ProductID, i.Interface, i.Product)
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer d.Close()
			for time.Now().Before(deadline) {
				r, err := d.Read(time.Until(deadline))
				if err != nil {
					return
				}
				usages, err := d.Usages(r, 0x0c)
				fmt.Printf("%04x mi=%d: raw % x  usages %v %v\n", i.ProductID, i.Interface, r, describe(usages), errText(err))
			}
		}()
	}
	fmt.Println("turn the dial / press media keys now (15 s)…")
	wg.Wait()
}

func describe(us []uint16) []string {
	out := []string{}
	for _, u := range us {
		if n, ok := names[u]; ok {
			out = append(out, n)
		} else {
			out = append(out, fmt.Sprintf("%#x", u))
		}
	}
	return out
}

func errText(err error) string {
	if err != nil {
		return "(" + err.Error() + ")"
	}
	return ""
}
