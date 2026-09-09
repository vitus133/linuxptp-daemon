package phcsync

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ptp4l patterns used by the sync session parser.
var (
	syncOffsetPattern = regexp.MustCompile(`\brms\s+(\d+)\s+max\s+(\d+)\s+`)

	// gradient side-effect: ReSyncRegressionDelta pattern for clock_class drift.
	clockClassPattern = regexp.MustCompile(`clock_class\s+(\d+)`)

	// ptp4l summary event patterns.
	kernelAppeared      = regexp.MustCompile(`(?i)\bKernel ptp clock (appeared|disappeared|changed)\b`)
	externalPortChanged = regexp.MustCompile(`(?i)port change\s+.*external.*`)
	ptpPortStatePattern = regexp.MustCompile(`(?i)\bport\s+\d+\s+became\s+(MASTER|SLAVE|LISTENING|FAULTY|UNKNOWN|DISABLED|UNCALIBRATED|PASSIVE)\b`)
)

// endState detects the end state of a state-machine step from ptp4l output.
func endState(line string) (string, bool) {
	if kernelAppeared.MatchString(line) {
		return PTP4LEndState, true
	}
	if externalPortChanged.MatchString(line) {
		return ExternalPortEndState, true
	}
	if m := ptpPortStatePattern.FindStringSubmatch(line); m != nil {
		return PTPPortStateEndState, true
	}
	return "", false
}

// clockClassCritical is the clock_class value at which the clock is declared
// out-of-sync (per IEEE 1588, class 255 marks an unsynchronized clock).
const clockClassCritical = 255

// timeoutState checks whether a given end-state has reached its timeout set.
func timeoutState(line string) (string, bool) {
	if m := clockClassPattern.FindStringSubmatch(line); m != nil {
		if v, err := strconv.Atoi(m[1]); err == nil && v >= clockClassCritical {
			return ClockClassTimeoutState, true
		}
	}
	return "", false
}

// parseSyncOutput turns captured ptp4l log output into the aggregate sync
// numbers used for the correction: last seen rms/max and last clock_class.
func parseSyncOutput(out string) SyncResult {
	var res SyncResult
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if m := syncOffsetPattern.FindStringSubmatch(line); m != nil {
			rms, err1 := strconv.Atoi(m[1])
			maxv, err2 := strconv.Atoi(m[2])
			if err1 == nil && err2 == nil {
				res.RMS = int64(rms)
				res.Max = int64(maxv)
				res.Count++
			}
		}
		if m := clockClassPattern.FindStringSubmatch(line); m != nil {
			if v, err := strconv.Atoi(m[1]); err == nil {
				res.ClockClass = int64(v)
			}
		}
	}
	return res
}

// parseLineBreakup is a small helper used by tests and the log UI to classify
// one log line.
func parseLineBreakup(line string) (string, bool) {
	if end, ok := endState(line); ok {
		return fmt.Sprintf("end_state:%s", end), true
	}
	if ts, ok := timeoutState(line); ok {
		return fmt.Sprintf("timeout_state:%s", ts), true
	}
	return "", false
}
