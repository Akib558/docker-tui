package docker

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type SystemMemory struct {
	Total     uint64
	Used      uint64
	Available uint64
	Percent   float64
}

type SystemLoad struct {
	Load1  float64
	Load5  float64
	Load15 float64
}

func GetSystemMemory() SystemMemory {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return SystemMemory{}
	}
	defer f.Close()

	var total, available, free, buffers, cached uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		val, _ := strconv.ParseUint(parts[1], 10, 64)
		val *= 1024 // kB -> bytes
		switch parts[0] {
		case "MemTotal:":
			total = val
		case "MemFree:":
			free = val
		case "MemAvailable:":
			available = val
		case "Buffers:":
			buffers = val
		case "Cached:":
			cached = val
		}
	}

	if available == 0 {
		available = free + buffers + cached
	}
	used := total - available

	pct := 0.0
	if total > 0 {
		pct = float64(used) / float64(total) * 100
	}

	return SystemMemory{Total: total, Used: used, Available: available, Percent: pct}
}

func GetSystemLoad() SystemLoad {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return SystemLoad{}
	}
	parts := strings.Fields(string(data))
	if len(parts) < 3 {
		return SystemLoad{}
	}
	l1, _ := strconv.ParseFloat(parts[0], 64)
	l5, _ := strconv.ParseFloat(parts[1], 64)
	l15, _ := strconv.ParseFloat(parts[2], 64)
	return SystemLoad{Load1: l1, Load5: l5, Load15: l15}
}
