package phcsync

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Synchronization end-state names reported by the state machine.
const (
	PTP4LEndState          = "PTP4L_END_STATE"
	ExternalPortEndState   = "EXTERNAL_PORT_END_STATE"
	PTPPortStateEndState   = "PTP_PORT_STATE_END_STATE"
	ClockClassTimeoutState = "CLOCK_CLASS_TIMEOUT_STATE"
)

// Status is the terminal outcome of a PHC synchronization attempt.
type Status string

const (
	StatusSynced    Status = "synced"
	StatusTimeout   Status = "timeout"
	StatusCritical  Status = "out-of-sync-critical"
	StatusDegraded  Status = "out-of-sync-degraded"
	StatusConfusion Status = "out-of-sync-confusion"
)

// SyncResult aggregates the last measured sync numbers from a ptp4l run.
type SyncResult struct {
	RMS        int64
	Max        int64
	ClockClass int64
	Count      int64
}

// Config carries the session inputs for one synchronization attempt.
type Config struct {
	Interface      string
	LogLevel       int
	Iterations     int
	OutputDir      string
	PTP4LArgs      []string
	PHCReadArgs    []string
	PHCWriteArgs   []string
	ClockClass     int64
	UpPeriod       time.Duration
	DetectInterval time.Duration
	MaxTaskTimeout time.Duration
}

// Task describes one synchronized action in the state machine.
type Task struct {
	Name      string
	Interface string
	Config    Config
	Process   *RunningProcess
}

// Sync runs a full synchronization attempt: ptp4l session, PHC read and
// corrected write.
func Sync(r Runner, cfg Config) (Status, error) {
	if cfg.Interface == "" {
		return StatusConfusion, errors.New("phcsync: interface is required")
	}
	var out string
	if r != nil {
		o, err := r.RunAndCapture(cfg.PTP4LArgs[0], cfg.PTP4LArgs[1:]...)
		out = o
		if err != nil {
			return StatusConfusion, fmt.Errorf("phcsync: ptp4l failed: %w\ncommand: %s\noutput:\n%s",
				err, strings.Join(cfg.PTP4LArgs, " "), out)
		}
	}
	res := parseSyncOutput(out)
	if res.RMS == 0 && res.Max == 0 {
		return StatusTimeout, errors.New("phcsync: no offset readings captured")
	}
	return applyCorrection(r, res, cfg)
}

// SyncFromOutput applies the PHC correction from already-captured ptp4l
// output (e.g. returned by RunPTP4LStream), running the phc_ctl read and
// write through the provided Runner.
func SyncFromOutput(r Runner, cfg Config, out string) (Status, error) {
	if cfg.Interface == "" {
		return StatusConfusion, errors.New("phcsync: interface is required")
	}
	res := parseSyncOutput(out)
	if res.RMS == 0 && res.Max == 0 {
		return StatusTimeout, fmt.Errorf("phcsync: no offset readings captured\noutput:\n%s", tail(out, 40))
	}
	return applyCorrection(r, res, cfg)
}

// tail returns the last n lines of s, for including process output in errors.
func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// ReSync re-reads the current PHC time without a fresh ptp4l session,
// used to refresh the clock after recovery.
func ReSync(r Runner, cfg Config) (Status, error) {
	if r == nil {
		return StatusConfusion, errors.New("phcsync: runner is required")
	}
	if _, err := r.RunAndCapture(cfg.PHCReadArgs[0], cfg.PHCReadArgs[1:]...); err != nil {
		return StatusConfusion, fmt.Errorf("phcsync: phc read failed: %w", err)
	}
	return StatusSynced, nil
}

func applyCorrection(r Runner, res SyncResult, cfg Config) (Status, error) {
	clockClass := res.ClockClass
	if clockClass == 0 {
		clockClass = cfg.ClockClass
	}
	if clockClass > 255 {
		return StatusTimeout, fmt.Errorf("phcsync: clock_class %d out of range", clockClass)
	}

	offsetNS := res.RMS
	if offsetNS > maxCorrectionNS(cfg) {
		return StatusConfusion, fmt.Errorf(
			"phcsync: measured rms offset %dns exceeds safe correction threshold %dns",
			offsetNS, maxCorrectionNS(cfg))
	}

	defaultCorr := defaultCorrection(cfg)
	if r != nil {
		args := append(append([]string{}, cfg.PHCWriteArgs...), "-n", "-C", offsetArg(offsetNS, defaultCorr), cfg.Interface)
		if _, err := r.RunAndCapture(args[0], args[1:]...); err != nil {
			return StatusConfusion, fmt.Errorf("phcsync: phc write failed: %w", err)
		}
	}
	return statusFor(res, cfg), nil
}

// statusFor classifies the aggregate sync result into a terminal status.
func statusFor(res SyncResult, cfg Config) Status {
	offsetNS := res.RMS
	switch {
	case offsetNS > maxOffsetNS(cfg):
		return StatusCritical
	case res.Count > 0 && res.Max > 2*offsetNS && offsetNS > 0:
		return StatusDegraded
	default:
		return StatusSynced
	}
}

func offsetArg(offsetNS, def int64) string {
	if offsetNS == 0 {
		offsetNS = def
	}
	if offsetNS < 0 {
		return fmt.Sprintf("-%d", -offsetNS)
	}
	return fmt.Sprintf("+%d", offsetNS)
}

func defaultCorrection(cfg Config) int64 {
	if cfg.UpPeriod > 0 {
		return int64(cfg.UpPeriod / time.Nanosecond)
	}
	return 0
}

func maxOffsetNS(cfg Config) int64 {
	const defaultMaxOffset = int64(2_000_000_000) // 2s
	if cfg.MaxTaskTimeout > 0 {
		return int64(cfg.MaxTaskTimeout / time.Nanosecond)
	}
	return defaultMaxOffset
}

func maxCorrectionNS(cfg Config) int64 {
	if cfg.UpPeriod > 0 {
		return int64(cfg.UpPeriod / time.Nanosecond)
	}
	return maxOffsetNS(cfg)
}
