package phcsync

import "testing"

func TestDiscoverNetDevice(t *testing.T) {
	dev, err := DiscoverNetDevice(nil, DiscoveryConfig{ScanDir: t.TempDir()})
	if err == nil && dev != nil {
		t.Fatalf("expected error scanning empty dir, got device %+v", dev)
	}
}

func TestDriverFromScan(t *testing.T) {
	if got := driverFromScan("/sys/class/net/eth0\nebtables\n"); got != "eth0" {
		t.Errorf("driverFromScan = %q, want eth0", got)
	}
	if got := driverFromScan("\n\n"); got != "" {
		t.Errorf("driverFromScan = %q, want empty", got)
	}
}

func TestSyncNoRunner(t *testing.T) {
	cfg := Config{
		Interface:   "eth0",
		PTP4LArgs:   []string{"ptp4l", "-i", "eth0"},
		PHCReadArgs: []string{"phc_ctl", "/dev/ptp0", "get"},
	}
	if _, err := Sync(nil, cfg); err == nil {
		t.Fatal("expected error calling Sync with nil runner")
	}
}
