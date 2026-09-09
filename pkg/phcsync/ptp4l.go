package phcsync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// defaultPTP4LConfig is written to the log directory on first run so ptp4l
// has a config to open when none is provided.
const defaultPTP4LConfig = `# phc-sync free-running measurement session
[global]
logSyncInterval -4
network_transport L2
`

// PTP4LArgs returns the arguments for a free-running measurement session on
// the given PTP interface, creating the log/config directory and a minimal
// default.cfg when it does not exist.
func PTP4LArgs(iface, logDir string) []string {
	_ = os.MkdirAll(logDir, os.ModePerm)
	dir := filepath.Join(logDir, iface)
	_ = os.MkdirAll(dir, os.ModePerm)
	cfgPath := filepath.Join(logDir, "default.cfg")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		_ = os.WriteFile(cfgPath, []byte(defaultPTP4LConfig), 0o644)
	}
	return []string{
		"ptp4l", "-f", cfgPath,
		"-i", iface, "-m", "-l", "7", "-s",
		"-p", "/dev/ptp0",
	}
}

// PTP4LConfig carries the ptp4l session inputs for a task.
type PTP4LConfig struct {
	Interface  string
	LogDir     string
	Iterations int
	Logger     func(format string, args ...any)
}

// RunPTP4LStream starts a ptp4l session, streaming every output line to
// cfg.Logger in real time, and returns the captured output. The session ends
// after cfg.Iterations offset measurements (when > 0), on a detected
// end-state line, or when no activity is seen for MaxTaskTimeout.
func RunPTP4LStream(cfg PTP4LConfig, maxTaskTimeout time.Duration) (out string, retErr error) {
	if cfg.Interface == "" {
		return "", errors.New("phcsync: ptp4l interface is required")
	}
	args := PTP4LArgs(cfg.Interface, cfg.LogDir)
	if cfg.Logger != nil {
		cfg.Logger("command: ptp4l %s", strings.Join(args[1:], " "))
	}
	proc, err := StartRunning(args[0], args[1:]...)
	if err != nil {
		return "", fmt.Errorf("phcsync: start ptp4l: %w", err)
	}
	defer func() {
		if err := proc.Stop(); err != nil && retErr == nil {
			retErr = fmt.Errorf("phcsync: stop ptp4l: %w", err)
		}
	}()

	sc := proc.Scanner()
	first := time.Now()
	last := first
	measurements := 0
	var lines []string
	for sc.Scan() {
		line := sc.Text()
		lines = append(lines, line)
		if line != "" {
			last = time.Now()
			if cfg.Logger != nil {
				cfg.Logger("ptp4l: %s", line)
			}
			if isMeasurementLine(line) {
				measurements++
				if cfg.Iterations > 0 && measurements >= cfg.Iterations {
					break
				}
			}
		}
		// endState detection mirrors the daemon's ptp4l state-machine watch.
		if _, ok := endState(line); ok {
			break
		}
		if maxTaskTimeout > 0 && time.Since(last) > maxTaskTimeout {
			return strings.Join(lines, "\n"), fmt.Errorf("phcsync: ptp4l idle for %s: %w", maxTaskTimeout, ErrPTP4LTimeout)
		}
		if maxTaskTimeout > 0 && time.Since(first) > maxTaskTimeout {
			return strings.Join(lines, "\n"), fmt.Errorf("phcsync: ptp4l session exceeded %s", maxTaskTimeout)
		}
	}
	return strings.Join(lines, "\n"), sc.Err()
}

// ErrPTP4LTimeout is returned when a ptp4l session produces no measurement
// activity within the task timeout.
var ErrPTP4LTimeout = errors.New("ptp4l measurement timeout")
