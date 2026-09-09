package phcsync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PTP4LArgs returns the arguments for a free-running measurement session on
// the given PTP interface.
func PTP4LArgs(iface, logDir string) []string {
	dir := filepath.Join(logDir, iface)
	_ = os.MkdirAll(dir, os.ModePerm)
	return []string{
		"ptp4l", "-f", filepath.Join(logDir, "default.cfg"),
		"-i", iface, "-m", "-l", "7", "-s",
		"-p", "/dev/ptp0",
	}
}

// PTP4LConfig carries the ptp4l session inputs for a task.
type PTP4LConfig struct {
	Interface string
	LogDir    string
	Out       string
	Logger    func(format string, args ...any)
}

// RunPTP4LStream starts a ptp4l session and consumes its log output until the
// first silence between measurement lines exceeds MaxTaskTimeout.
func RunPTP4LStream(cfg PTP4LConfig, maxTaskTimeout time.Duration) error {
	if cfg.Interface == "" {
		return errors.New("phcsync: ptp4l interface is required")
	}
	proc, err := StartRunning(PTP4LArgs(cfg.Interface, cfg.LogDir)[0], PTP4LArgs(cfg.Interface, cfg.LogDir)[1:]...)
	if err != nil {
		return fmt.Errorf("phcsync: start ptp4l: %w", err)
	}
	defer proc.Stop()

	sc := proc.Scanner()
	first := time.Now()
	last := first
	for sc.Scan() {
		line := sc.Text()
		if line != "" {
			last = time.Now()
			if cfg.Logger != nil {
				cfg.Logger("%s", line)
			}
		}
		if line != "" {
			last = time.Now()
		}
		// endState detection mirrors the daemon's ptp4l state-machine watch.
		if _, ok := endState(line); ok {
			break
		}
		if maxTaskTimeout > 0 && time.Since(last) > maxTaskTimeout {
			return fmt.Errorf("phcsync: ptp4l idle for %s: %w", maxTaskTimeout, ErrPTP4LTimeout)
		}
		if maxTaskTimeout > 0 && time.Since(first) > maxTaskTimeout {
			return fmt.Errorf("phcsync: ptp4l session exceeded %s", maxTaskTimeout)
		}
	}
	return nil
}

// ErrPTP4LTimeout is returned when a ptp4l session produces no measurement
// activity within the task timeout.
var ErrPTP4LTimeout = errors.New("ptp4l measurement timeout")
