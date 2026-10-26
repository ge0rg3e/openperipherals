package keyblock

import (
	"slices"
	"sync"
	"time"

	"openperipherals/internal/hid"
)

const pageConsumer = 0x0c

// Consumer-page usages and the virtual keys Windows turns them into.
var usageVK = map[uint16]uint8{
	0xe9: 0xaf, // volume up
	0xea: 0xae, // volume down
	0xe2: 0xad, // mute
	0xcd: 0xb3, // play/pause
	0xb5: 0xb0, // next track
	0xb6: 0xb1, // previous track
	0xb7: 0xb2, // stop
}

// attributable marks the virtual keys a watcher can claim.
var attributable = func() (t [256]bool) {
	for _, vk := range usageVK {
		t[vk] = true
	}
	return t
}()

// Attributable reports whether presses of vk can be traced to a watched
// device (only consumer-page media keys can).
func Attributable(vk uint8) bool { return attributable[vk] }

// Watcher reads a device's consumer-control collections and claims each new
// media key press, so device-only blocking can tell its presses apart.
type Watcher struct {
	vid, pid uint16
	stop     chan struct{}
	wg       sync.WaitGroup
}

// Watch starts watching every consumer-control collection of vid:pid,
// reattaching when the device is replugged.
func Watch(vid, pid uint16) *Watcher {
	w := &Watcher{vid: vid, pid: pid, stop: make(chan struct{})}
	w.wg.Add(1)
	go w.run()
	return w
}

// Stop ends watching.
func (w *Watcher) Stop() {
	close(w.stop)
	w.wg.Wait()
}

func (w *Watcher) run() {
	defer w.wg.Done()
	for {
		var readers sync.WaitGroup
		infos, _ := hid.Enumerate(w.vid, w.pid)
		for _, info := range infos {
			if info.UsagePage != pageConsumer {
				continue
			}
			dev, err := hid.Open(info)
			if err != nil {
				continue
			}
			readers.Add(1)
			go func() {
				defer readers.Done()
				defer dev.Close()
				w.read(dev)
			}()
		}
		readers.Wait() // returns when stopped or every collection is gone
		select {
		case <-w.stop:
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (w *Watcher) read(dev *hid.Device) {
	var pressed []uint16
	for {
		select {
		case <-w.stop:
			return
		default:
		}
		report, err := dev.Read(250 * time.Millisecond)
		if err == hid.ErrTimeout {
			continue
		}
		if err != nil {
			return // unplugged
		}
		usages, err := dev.Usages(report, pageConsumer)
		if err != nil {
			continue // a report for another usage page
		}
		for _, u := range usages {
			if vk, ok := usageVK[u]; ok && !slices.Contains(pressed, u) {
				Claim(vk)
			}
		}
		pressed = append(pressed[:0], usages...)
	}
}
