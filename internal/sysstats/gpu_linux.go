package sysstats

import (
	"context"
	"encoding/csv"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func gpuStats() []GPU {
	var gpus []GPU
	cards, _ := filepath.Glob("/sys/class/drm/card[0-9]*/device")
	for _, device := range cards {
		usage, ok := gpuNumber(device, "gpu_busy_percent")
		if !ok || usage > 100 {
			continue
		}
		used, _ := gpuNumber(device, "mem_info_vram_used")
		total, _ := gpuNumber(device, "mem_info_vram_total")
		name := "AMD GPU (" + filepath.Base(filepath.Dir(device)) + ")"
		gpus = append(gpus, GPU{Name: name, Usage: int(usage), VRAMUsed: used, VRAMTotal: total})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
	defer cancel()
	raw, err := exec.CommandContext(ctx, "nvidia-smi", "--query-gpu=name,utilization.gpu,memory.used,memory.total", "--format=csv,noheader,nounits").Output()
	if err != nil {
		return gpus
	}
	rows, err := csv.NewReader(strings.NewReader(string(raw))).ReadAll()
	if err != nil {
		return gpus
	}
	for _, row := range rows {
		if len(row) != 4 {
			continue
		}
		usage, err := strconv.ParseUint(strings.TrimSpace(row[1]), 10, 8)
		if err != nil || usage > 100 {
			continue
		}
		used, _ := strconv.ParseUint(strings.TrimSpace(row[2]), 10, 44)
		total, _ := strconv.ParseUint(strings.TrimSpace(row[3]), 10, 44)
		gpus = append(gpus, GPU{Name: strings.TrimSpace(row[0]), Usage: int(usage), VRAMUsed: used << 20, VRAMTotal: total << 20})
	}
	return gpus
}

func gpuNumber(device, name string) (uint64, bool) {
	data, err := os.ReadFile(filepath.Join(device, name))
	if err != nil {
		return 0, false
	}
	n, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	return n, err == nil
}
