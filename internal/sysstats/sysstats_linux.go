package sysstats

import (
	"os"
	"strconv"
	"strings"
	"sync"
)

type Sampler struct {
	mu          sync.Mutex
	total, idle uint64
}

func (s *Sampler) Sample() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()

	st := Stats{CPU: -1}
	if data, err := os.ReadFile("/proc/stat"); err == nil {
		line, _, _ := strings.Cut(string(data), "\n")
		total, idle := cpuTimes(line)
		if total > s.total && s.total != 0 && idle >= s.idle && idle-s.idle <= total-s.total {
			st.CPU = int(100 * ((total - s.total) - (idle - s.idle)) / (total - s.total))
		}
		s.total, s.idle = total, idle
	}
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		st.RAMUsed, st.RAMTotal = memoryTimes(string(data))
	}
	st.GPUs = gpuStats()
	return st
}

func cpuTimes(line string) (total, idle uint64) {
	f := strings.Fields(line)
	if len(f) < 5 || f[0] != "cpu" {
		return 0, 0
	}
	// guest and guest_nice are already included in user and nice.
	for i, field := range f[1:min(len(f), 9)] {
		n, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return 0, 0
		}
		total += n
		if i == 3 || i == 4 {
			idle += n
		}
	}
	return total, idle
}

func memoryTimes(data string) (used, total uint64) {
	var available uint64
	var haveAvailable bool
	for line := range strings.SplitSeq(data, "\n") {
		key, value, _ := strings.Cut(line, ":")
		if key != "MemTotal" && key != "MemAvailable" {
			continue
		}
		fields := strings.Fields(value)
		if len(fields) != 2 || fields[1] != "kB" {
			continue
		}
		n, err := strconv.ParseUint(fields[0], 10, 54)
		if err != nil {
			continue
		}
		if key == "MemTotal" {
			total = n * 1024
		} else {
			available, haveAvailable = n*1024, true
		}
	}
	if haveAvailable && available <= total {
		used = total - available
	}
	return used, total
}
