package phcsync

import (
	"fmt"
	"os"
	"path/filepath"
)

// NetDevice describes a discovered network device that can synchronize.
type NetDevice struct {
	Name         string
	Driver       string
	MACAddress   string
	PtpInterface bool
}

// DiscoveryConfig limits how many devices are scanned.
type DiscoveryConfig struct {
	MaxDevices int
	ScanDir    string
}

// DiscoverNetDevice scans the system for PTP-capable network interfaces.
func DiscoverNetDevice(r Runner, cfg DiscoveryConfig) (*NetDevice, error) {
	if cfg.ScanDir == "" {
		cfg.ScanDir = "/sys/class/net"
	}
	entries, err := os.ReadDir(cfg.ScanDir)
	if err != nil {
		return nil, fmt.Errorf("phcsync: scan %s: %w", cfg.ScanDir, err)
	}

	if cfg.MaxDevices <= 0 {
		cfg.MaxDevices = len(entries)
	}

	var out string
	if r != nil {
		if out, err = r.RunAndCapture("bash", "-c", "ls -d /sys/class/net/*"); err != nil {
			return nil, fmt.Errorf("phcsync: device scan failed: %w", err)
		}
	}

	var driver string
	if out != "" {
		driver = driverFromScan(out)
	}

	for i, e := range entries {
		if i >= cfg.MaxDevices {
			break
		}
		if e.IsDir() {
			name := e.Name()
			d := &NetDevice{
				Name:         name,
				Driver:       driver,
				MACAddress:   macAddress(r, cfg.ScanDir, name),
				PtpInterface: true,
			}
			return d, nil
		}
	}
	return nil, fmt.Errorf("phcsync: no network device found under %s", cfg.ScanDir)
}

// driverFromScan extracts the first driver name from an ls listing.
func driverFromScan(out string) string {
	for _, line := range splitLines(out) {
		if line != "" {
			return filepath.Base(line)
		}
	}
	return ""
}

func macAddress(r Runner, scanDir, name string) string {
	if r == nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(scanDir, name, "address"))
	if err != nil {
		return ""
	}
	return string(data)
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return out
}
