package phcsync

import (
	"fmt"
	"strings"
	"testing"
)

func TestLoggingRunnerStreamsLines(t *testing.T) {
	var logged []string
	lr := LoggingRunner{
		Inner: OSRunner{},
		Log: func(format string, args ...any) {
			logged = append(logged, fmt.Sprintf(format, args...))
		},
	}
	out, err := lr.RunAndCapture("printf", "first\nsecond\n")
	if err != nil {
		t.Fatalf("RunAndCapture: %v", err)
	}
	if !strings.Contains(out, "first\nsecond\n") {
		t.Errorf("captured output missing lines: %q", out)
	}
	want := []string{"command: printf first\nsecond\n", "printf: first", "printf: second"}
	for _, w := range want {
		found := false
		for _, l := range logged {
			if l == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing log line %q; got %v", w, logged)
		}
	}
	if len(logged) < 3 {
		t.Errorf("expected at least 3 log lines, got %d", len(logged))
	}
}

func TestIsMeasurementLine(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{"ptp4l[1.2]: sync rms     321 max    512 clock_class  159", true},
		{"ptp4l[1.2]: master offset          0 s2 freq   -1 path delay       123", true},
		{"ptp4l[1.2]: master offset         -7 s2 freq   -1 path delay       123", true},
		{"ptp4l[1.2]: port 1 became MASTER", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isMeasurementLine(c.line); got != c.want {
			t.Errorf("isMeasurementLine(%q) = %v, want %v", c.line, got, c.want)
		}
	}
}

func TestTail(t *testing.T) {
	s := "a\nb\nc\n"
	if got := tail(s, 10); got != "a\nb\nc" {
		t.Errorf("tail(10) = %q", got)
	}
	if got := tail(s, 1); got != "c" {
		t.Errorf("tail(1) = %q", got)
	}
}
