package mac

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"
)

var errTest = errors.New("test error")

func TestLock(t *testing.T) {
	fake := newFakeRunner()
	if err := Lock(context.Background(), fake); err != nil {
		t.Fatalf("Lock error = %v", err)
	}
	if !fake.ran(cgsessionLock, "-suspend") {
		t.Errorf("lock helper was not invoked; calls = %v", fake.calls)
	}
}

func TestSleep(t *testing.T) {
	fake := newFakeRunner()
	if err := Sleep(context.Background(), fake); err != nil {
		t.Fatalf("Sleep error = %v", err)
	}
	if !fake.ran("pmset", "sleepnow") {
		t.Errorf("pmset sleepnow was not invoked; calls = %v", fake.calls)
	}
}

func TestSystemStatus(t *testing.T) {
	fake := newFakeRunner().
		withOutput("sysctl -n kern.boottime", "{ sec = 1752345600 } 123456789 0\n").
		withOutput("top -l 1 -n 0", "Processes: 387 total; 2 running; 385 sleeping\nCPU usage: 5.26% user, 3.51% sys, 91.22% idle\n").
		withOutput("sysctl -n hw.memsize", "17179869184\n").
		withOutput("vm_stat", "Mach Virtual Memory Statistics: (page size of 4096 bytes)\nPages free: 12345.\nPages active: 2345678.\nPages inactive: 3456789.\nPages speculative: 0.\nPages wired down: 345678.\nPages occupied by compressor: 456789.\n").
		withOutput("pmset -g batt", "Now drawing from 'AC Power'\n -InternalBattery-0 (id=1234567)\t100%; charged; 0:00 remaining present: true\n").
		withOutput("df -k /", "Filesystem   1024-blocks     Used Available Capacity iused ifree %iused  Mounted on\n/dev/disk3s1 245110784 135202900 108834076    56% 123456 789012 14%   /\n")

	st, err := SystemStatus(context.Background(), fake)
	if err != nil {
		t.Fatalf("SystemStatus error = %v", err)
	}

	if st.Uptime == "" {
		t.Error("Uptime is empty")
	}
	if want := 5.26 + 3.51; st.CPU != want {
		t.Errorf("CPU = %v, want %v", st.CPU, want)
	}
	wantMem := float64((2345678+345678+456789)*4096) / 17179869184 * 100
	if st.Memory != wantMem {
		t.Errorf("Memory = %v, want %v", st.Memory, wantMem)
	}
	if st.Battery != 100 {
		t.Errorf("Battery = %d, want 100", st.Battery)
	}
	if !st.Charging {
		t.Error("Charging = false, want true (charged on AC)")
	}
	if st.Disk != 56 {
		t.Errorf("Disk = %v, want 56", st.Disk)
	}
}

func TestSystemStatusBatteryOnPower(t *testing.T) {
	fake := newFakeRunner().
		withOutput("sysctl -n kern.boottime", "{ sec = 1752345600 } 0 0\n").
		withOutput("top -l 1 -n 0", "CPU usage: 0.00% user, 0.00% sys, 100.00% idle\n").
		withOutput("sysctl -n hw.memsize", "17179869184\n").
		withOutput("vm_stat", "Mach Virtual Memory Statistics: (page size of 4096 bytes)\nPages active: 1.\nPages wired down: 1.\nPages occupied by compressor: 1.\n").
		withOutput("pmset -g batt", "Now drawing from 'Battery Power'\n -InternalBattery-0 (id=1234567)\t74%; discharging; 4:02 remaining present: true\n").
		withOutput("df -k /", "/dev/disk3s1 245110784 135202900 108834076    56% 123456 789012 14%   /\n")

	st, err := SystemStatus(context.Background(), fake)
	if err != nil {
		t.Fatalf("SystemStatus error = %v", err)
	}
	if st.Battery != 74 || st.Charging {
		t.Errorf("Battery = %d, Charging = %v; want 74, false", st.Battery, st.Charging)
	}
}

func TestSystemStatusReadFailure(t *testing.T) {
	fake := newFakeRunner().
		withOutput("sysctl -n kern.boottime", "{ sec = 1752345600 } 0 0\n").
		withError("top -l 1 -n 0", errTest)

	if _, err := SystemStatus(context.Background(), fake); err == nil {
		t.Error("SystemStatus: want error when a read fails")
	}
}

func TestParseUptime(t *testing.T) {
	start := time.Now()
	sec := strconv.FormatInt(start.Unix(), 10)
	for _, out := range []string{
		"{ sec = " + sec + ", usec = 462503 } Wed Jul 15 10:58:54 2026\n",
		"{ sec = " + sec + " } 123456 0\n",
	} {
		d, err := parseUptime(out)
		if err != nil {
			t.Fatalf("parseUptime(%q) error = %v", out, err)
		}
		if d < 0 || d > time.Minute {
			t.Errorf("parseUptime(%q) = %v, want a small positive duration", out, d)
		}
	}

	if _, err := parseUptime("garbage"); err == nil {
		t.Error("parseUptime: want error on garbage input")
	}
}

func TestFormatUptime(t *testing.T) {
	for _, tt := range []struct {
		in   time.Duration
		want string
	}{
		{25 * time.Hour, "1d1h"},
		{90 * time.Minute, "1h30m"},
		{45 * time.Minute, "45m"},
		{2 * time.Hour, "2h0m"},
	} {
		if got := formatUptime(tt.in); got != tt.want {
			t.Errorf("formatUptime(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseCPUUsage(t *testing.T) {
	got, err := parseCPUUsage("Processes: 387 total\nCPU usage: 12.5% user, 3.2% sys, 84.3% idle\n")
	if err != nil {
		t.Fatalf("parseCPUUsage error = %v", err)
	}
	if got != 15.7 {
		t.Errorf("parseCPUUsage = %v, want 15.7", got)
	}
	if _, err := parseCPUUsage("no cpu line here"); err == nil {
		t.Error("parseCPUUsage: want error on missing line")
	}
}

func TestParseMemoryUsed(t *testing.T) {
	out := "Mach Virtual Memory Statistics: (page size of 4096 bytes)\nPages active: 100.\nPages wired down: 50.\nPages occupied by compressor: 25.\n"
	got, err := parseMemoryUsed(out, 4096*400)
	if err != nil {
		t.Fatalf("parseMemoryUsed error = %v", err)
	}
	// used = 175 pages of 4096 bytes out of 400 pages total.
	if got != 43.75 {
		t.Errorf("parseMemoryUsed = %v, want 43.75", got)
	}
	if _, err := parseMemoryUsed("Pages active: 1.\n", 100); err == nil {
		t.Error("parseMemoryUsed: want error on missing fields")
	}
}

func TestParseBattery(t *testing.T) {
	percent, charging, err := parseBattery("Now drawing from 'AC Power'\n -InternalBattery-0 (id=1234567)\t100%; charged; 0:00 remaining present: true\n")
	if err != nil {
		t.Fatalf("parseBattery error = %v", err)
	}
	if percent != 100 || !charging {
		t.Errorf("parseBattery = %d, %v; want 100, true", percent, charging)
	}

	percent, charging, err = parseBattery("-InternalBattery-0 (id=1234567)\t74%; discharging; 4:02 remaining present: true\n")
	if err != nil {
		t.Fatalf("parseBattery error = %v", err)
	}
	if percent != 74 || charging {
		t.Errorf("parseBattery = %d, %v; want 74, false", percent, charging)
	}

	if _, _, err := parseBattery("no battery"); err == nil {
		t.Error("parseBattery: want error on missing line")
	}
}

func TestParseDiskUsage(t *testing.T) {
	out := "Filesystem   1024-blocks     Used Available Capacity iused ifree %iused  Mounted on\n/dev/disk3s1 245110784 135202900 108834076    56% 123456 789012 14%   /\n"
	got, err := parseDiskUsage(out)
	if err != nil {
		t.Fatalf("parseDiskUsage error = %v", err)
	}
	if got != 56 {
		t.Errorf("parseDiskUsage = %v, want 56", got)
	}
	if _, err := parseDiskUsage("nothing"); err == nil {
		t.Error("parseDiskUsage: want error on missing root volume")
	}
}
