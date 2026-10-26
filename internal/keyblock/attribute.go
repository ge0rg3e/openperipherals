package keyblock

import (
	"sync/atomic"
	"time"
)

const (
	// claimWait is how long the hook waits for the device watcher to claim a
	// press; HID reports and hook events for one press arrive within a few ms.
	claimWait = 30 * time.Millisecond
	// claimTTL discards claims the hook never consumed.
	claimTTL = 150 * time.Millisecond
	// repeatWindow treats a key-down shortly after a blocked one as
	// auto-repeat of the same press.
	repeatWindow = 600 * time.Millisecond
)

var deviceOnly atomic.Bool

type claim struct {
	n  atomic.Int32
	at atomic.Int64 // unix nanos of the latest claim
}

var (
	claims [256]claim
	held   [256]atomic.Int64 // unix nanos of the latest blocked key-down, 0 when released
)

// SetDeviceOnly switches between blocking keys from every device (false) and,
// for keys a device watcher can report (media keys), blocking only presses it
// claimed (true). Other keys can't be attributed and are always blocked.
func SetDeviceOnly(on bool) { deviceOnly.Store(on) }

// Claim records that the watched device just pressed vk.
func Claim(vk uint8) {
	c := &claims[vk]
	now := time.Now().UnixNano()
	if now-c.at.Load() > int64(claimTTL) {
		c.n.Store(0)
	}
	c.n.Add(1)
	c.at.Store(now)
}

func take(vk uint8) bool {
	c := &claims[vk]
	if time.Now().UnixNano()-c.at.Load() > int64(claimTTL) {
		return false
	}
	for {
		n := c.n.Load()
		if n <= 0 {
			return false
		}
		if c.n.CompareAndSwap(n, n-1) {
			return true
		}
	}
}

// shouldBlock runs on the hook thread for every blocked key event.
func shouldBlock(vk uint8, up bool) bool {
	if !deviceOnly.Load() || !Attributable(vk) {
		return true
	}
	now := time.Now().UnixNano()
	if up {
		return held[vk].Swap(0) != 0
	}
	if h := held[vk].Load(); h != 0 && now-h < int64(repeatWindow) {
		held[vk].Store(now)
		return true
	}
	for deadline := time.Now().Add(claimWait); ; {
		if take(vk) {
			held[vk].Store(time.Now().UnixNano())
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(time.Millisecond)
	}
}
