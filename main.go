package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/mem"
)

var exitCodes = map[string]int{
	"OK":       0,
	"WARNING":  1,
	"CRITICAL": 2,
	"UNKNOWN":  3,
}

type threshold struct {
	value float64
	unit  string
}

func parseThreshold(s string) (threshold, error) {
	s = strings.TrimSpace(s)
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return threshold{}, fmt.Errorf("invalid percentage: %v", err)
	}
	if val < 0 || val > 100 {
		return threshold{}, fmt.Errorf("percentage must be between 0 and 100")
	}
	return threshold{val, "%"}, nil
}

func formatPerfData(values map[string]float64, includeHuge bool) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("|TOTAL=%.0fKB;;;;", values["total"]))
	builder.WriteString(fmt.Sprintf(" USED=%.0fKB;;;;", values["used"]))
	builder.WriteString(fmt.Sprintf(" AVAILABLE=%.0fKB;;;;", values["available"]))
	builder.WriteString(fmt.Sprintf(" CACHED=%.0fKB;;;;", values["cached"]))
	builder.WriteString(fmt.Sprintf(" BUFFERS=%.0fKB;;;;", values["buffers"]))
	if includeHuge {
		builder.WriteString(fmt.Sprintf(" HUGEPAGES=%.0fKB;;;;", values["hugepages"]))
	}
	builder.WriteString(fmt.Sprintf(" SWAP_USED=%.0fKB;;;; SWAP_FREE=%.0fKB;;;;", values["swap_used"], values["swap_free"]))
	return builder.String()
}

func exitWith(status string, msg string) {
	code, ok := exitCodes[status]
	if !ok {
		fmt.Printf("UNKNOWN - Invalid status '%s'\n", status)
		os.Exit(exitCodes["UNKNOWN"])
	}
	fmt.Println(msg)
	os.Exit(code)
}

func main() {
	metric := flag.String("metric", "used", "Metric to check: 'used' or 'available'. 'free' is not supported (use 'available' instead).")
	warnStr := flag.String("w", "", "Warning threshold (percentage only, e.g. 80)")
	critStr := flag.String("c", "", "Critical threshold (percentage only, e.g. 90)")
	includeHuge := flag.Bool("huge", false, "Include hugepages in performance data")
	flag.Parse()

	if *warnStr == "" || *critStr == "" {
		exitWith("UNKNOWN", "UNKNOWN - Warning and Critical thresholds must be provided")
	}

	warn, err := parseThreshold(*warnStr)
	if err != nil {
		exitWith("UNKNOWN", fmt.Sprintf("UNKNOWN - %v", err))
	}
	crit, err := parseThreshold(*critStr)
	if err != nil {
		exitWith("UNKNOWN", fmt.Sprintf("UNKNOWN - %v", err))
	}

	if warn.value >= crit.value {
		exitWith("UNKNOWN", "UNKNOWN - WARNING must be less than CRITICAL")
	}

	vm, err := mem.VirtualMemory()
	if err != nil {
		exitWith("UNKNOWN", fmt.Sprintf("UNKNOWN - Failed to get virtual memory info: %v", err))
	}

	swap, err := mem.SwapMemory()
	if err != nil {
		swap.Used = 0
		swap.Free = 0
	}

	values := map[string]float64{
		"total":     float64(vm.Total) / 1024,
		"used":      float64(vm.Used) / 1024,
		"available": float64(vm.Available) / 1024,
		"cached":    float64(vm.Cached) / 1024,
		"buffers":   float64(vm.Buffers) / 1024,
		"hugepages": float64(vm.HugePagesTotal*vm.HugePageSize) / 1024,
		"swap_used": float64(swap.Used) / 1024,
		"swap_free": float64(swap.Free) / 1024,
	}

	m := strings.ToLower(*metric)
	if m != "used" && m != "available" {
		exitWith("UNKNOWN", fmt.Sprintf("UNKNOWN - Invalid metric '%s'. Only 'used' and 'available' are supported.", m))
	}

	currentVal := values[m]
	total := values["total"]
	percent := (currentVal / total) * 100
	perf := formatPerfData(values, *includeHuge)

	switch m {
	case "used":
		if percent > crit.value {
			exitWith("CRITICAL", fmt.Sprintf("CRITICAL - %.1f%% (%.0f kB) used memory%s", percent, currentVal, perf))
		} else if percent > warn.value {
			exitWith("WARNING", fmt.Sprintf("WARNING - %.1f%% (%.0f kB) used memory%s", percent, currentVal, perf))
		}
	case "available":
		if percent < crit.value {
			exitWith("CRITICAL", fmt.Sprintf("CRITICAL - %.1f%% (%.0f kB) available memory%s", percent, currentVal, perf))
		} else if percent < warn.value {
			exitWith("WARNING", fmt.Sprintf("WARNING - %.1f%% (%.0f kB) available memory%s", percent, currentVal, perf))
		}
	}

	exitWith("OK", fmt.Sprintf("OK - %.1f%% (%.0f kB) %s memory%s", percent, currentVal, m, perf))
}
