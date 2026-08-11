package mac

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// cgsessionLock is the fixed-path screen lock helper shipped with macOS. It
// locks the screen without sudo and takes no user input.
const cgsessionLock = "/System/Library/CoreServices/Menu Extras/User.menu/Contents/Resources/CGSession"

// Lock locks the screen (PRD §10).
func Lock(ctx context.Context, r Runner) error {
	return r.Run(ctx, cgsessionLock, "-suspend")
}

// Sleep puts the Mac to sleep (PRD §11). pmset sleepnow needs no privileges.
func Sleep(ctx context.Context, r Runner) error {
	return r.Run(ctx, "pmset", "sleepnow")
}

// Status carries a concise snapshot of system health (PRD §14).
type Status struct {
	Uptime   string  `json:"uptime"`
	CPU      float64 `json:"cpu_percent"`
	Memory   float64 `json:"memory_percent"`
	Battery  int     `json:"battery_percent"`
	Charging bool    `json:"charging"`
	Disk     float64 `json:"disk_percent"`
}

// SystemStatus reads system health through read-only commands. Every read
// maps to a small pure parser so the parsing is unit-testable (PRD §25).
func SystemStatus(ctx context.Context, r Runner) (Status, error) {
	var st Status

	boot, err := r.Output(ctx, "sysctl", "-n", "kern.boottime")
	if err != nil {
		return st, fmt.Errorf("uptime: %w", err)
	}
	uptime, err := parseUptime(boot)
	if err != nil {
		return st, fmt.Errorf("uptime: %w", err)
	}
	st.Uptime = formatUptime(uptime)

	top, err := r.Output(ctx, "top", "-l", "1", "-n", "0")
	if err != nil {
		return st, fmt.Errorf("cpu: %w", err)
	}
	st.CPU, err = parseCPUUsage(top)
	if err != nil {
		return st, fmt.Errorf("cpu: %w", err)
	}

	memSizeRaw, err := r.Output(ctx, "sysctl", "-n", "hw.memsize")
	if err != nil {
		return st, fmt.Errorf("memory: %w", err)
	}
	memSize, err := strconv.ParseInt(strings.TrimSpace(memSizeRaw), 10, 64)
	if err != nil {
		return st, fmt.Errorf("memory: %w", err)
	}
	vmstat, err := r.Output(ctx, "vm_stat")
	if err != nil {
		return st, fmt.Errorf("memory: %w", err)
	}
	st.Memory, err = parseMemoryUsed(vmstat, memSize)
	if err != nil {
		return st, fmt.Errorf("memory: %w", err)
	}

	batt, err := r.Output(ctx, "pmset", "-g", "batt")
	if err != nil {
		return st, fmt.Errorf("battery: %w", err)
	}
	percent, charging, err := parseBattery(batt)
	if err != nil {
		return st, fmt.Errorf("battery: %w", err)
	}
	st.Battery, st.Charging = percent, charging

	df, err := r.Output(ctx, "df", "-k", "/")
	if err != nil {
		return st, fmt.Errorf("disk: %w", err)
	}
	st.Disk, err = parseDiskUsage(df)
	if err != nil {
		return st, fmt.Errorf("disk: %w", err)
	}

	return st, nil
}

// parseUptime reads the boot timestamp from `sysctl -n kern.boottime`.
// Newer macOS emits "{ sec = 1784093334, usec = 462503 }", older releases
// "{ sec = 1784093334 }"; both end the seconds value at a delimiter.
func parseUptime(out string) (time.Duration, error) {
	idx := strings.Index(out, "sec = ")
	if idx < 0 {
		return 0, fmt.Errorf("unexpected boottime output %q", out)
	}
	rest := out[idx+len("sec = "):]
	end := strings.IndexAny(rest, " ,}")
	if end < 0 {
		return 0, fmt.Errorf("unexpected boottime output %q", out)
	}
	secs, err := strconv.ParseInt(strings.TrimSpace(rest[:end]), 10, 64)
	if err != nil {
		return 0, err
	}
	return time.Since(time.Unix(secs, 0)), nil
}

func formatUptime(d time.Duration) string {
	d = d.Truncate(time.Minute)
	hours := int(d.Hours())
	switch {
	case hours >= 24:
		return fmt.Sprintf("%dd%dh", hours/24, hours%24)
	case hours >= 1:
		return fmt.Sprintf("%dh%dm", hours, int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
}

// parseCPUUsage reads the used CPU share from `top -l 1 -n 0`:
// "CPU usage: 5.26% user, 3.51% sys, 91.22% idle".
func parseCPUUsage(out string) (float64, error) {
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "CPU usage:") {
			continue
		}
		var user, sys, idle float64
		if _, err := fmt.Sscanf(line, "CPU usage: %f%% user, %f%% sys, %f%% idle", &user, &sys, &idle); err != nil {
			return 0, fmt.Errorf("parse cpu line %q: %w", line, err)
		}
		return user + sys, nil
	}
	return 0, fmt.Errorf("no CPU usage line in %q", out)
}

// parseMemoryUsed computes the used memory share from `vm_stat` output:
// "Mach Virtual Memory Statistics: (page size of 4096 bytes)" followed by
// "Pages active: N.", "Pages wired down: N.", "Pages occupied by compressor: N.".
// Used memory is the sum of active, wired and compressed pages.
func parseMemoryUsed(out string, totalBytes int64) (float64, error) {
	pageSize := int64(4096)
	if idx := strings.Index(out, "page size of "); idx >= 0 {
		if n, err := strconv.ParseInt(out[idx+len("page size of "):], 10, 64); err == nil {
			pageSize = n
		}
	}

	var used int64
	for _, field := range []string{"Pages active", "Pages wired down", "Pages occupied by compressor"} {
		n, ok := parsePageField(out, field)
		if !ok {
			return 0, fmt.Errorf("missing %q in vm_stat output", field)
		}
		used += n
	}

	if totalBytes <= 0 {
		return 0, fmt.Errorf("invalid memory size %d", totalBytes)
	}
	return float64(used*pageSize) / float64(totalBytes) * 100, nil
}

func parsePageField(out, field string) (int64, bool) {
	idx := strings.Index(out, field)
	if idx < 0 {
		return 0, false
	}
	rest := out[idx+len(field):]
	if !strings.HasPrefix(rest, ":") {
		return 0, false
	}
	rest = rest[1:]
	end := strings.IndexByte(rest, '.')
	if end < 0 {
		end = len(rest)
	}
	n, err := strconv.ParseInt(strings.TrimSpace(rest[:end]), 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// parseBattery reads charge percent and state from `pmset -g batt`:
// "Now drawing from 'AC Power'" and
// "-InternalBattery-0 (id=1234567)	100%; charged; 0:00 remaining present: true".
func parseBattery(out string) (percent int, charging bool, err error) {
	for _, line := range strings.Split(out, "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "internalbattery") || strings.Contains(lower, "smart battery") {
			if idx := strings.Index(lower, "%"); idx >= 0 {
				start := idx - 3
				if start < 0 {
					start = 0
				}
				p, err := strconv.Atoi(strings.TrimSpace(line[start:idx]))
				if err != nil {
					return 0, false, fmt.Errorf("parse battery percent in %q: %w", line, err)
				}
				// "discharging" contains "charging" as a substring, so the
				// state must be matched most-specific-first.
				charging = false
				switch {
				case strings.Contains(lower, "discharging"):
					charging = false
				case strings.Contains(lower, "charging"):
					charging = true
				case strings.Contains(lower, "charged"):
					charging = true
				}
				return p, charging, nil
			}
		}
	}
	return 0, false, fmt.Errorf("no battery line in %q", out)
}

// parseDiskUsage reads the used share of the root volume from `df -k /`:
// "Filesystem   1024-blocks     Used Available Capacity ... Mounted on" and
// "/dev/disk3s1 245110784 135202900 108834076    56% ... /".
func parseDiskUsage(out string) (float64, error) {
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 9 || fields[len(fields)-1] != "/" {
			continue
		}
		cap := strings.TrimSuffix(fields[len(fields)-5], "%")
		pct, err := strconv.ParseFloat(cap, 64)
		if err != nil {
			return 0, fmt.Errorf("parse capacity %q: %w", cap, err)
		}
		return pct, nil
	}
	return 0, fmt.Errorf("no root volume line in %q", out)
}
