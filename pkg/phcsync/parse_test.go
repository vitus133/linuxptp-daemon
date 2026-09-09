package phcsync

import "testing"

const samplePTP4LLog = `ptp4l[123.456]: master offset          0 s2 freq   -1 path delay       123
ptp4l[123.457]: master offset         87 s2 freq   -1 path delay       123
ptp4l[123.458]: sync rms     321 max    512 clock_class  159
`

func TestParseSyncOutput(t *testing.T) {
	res := parseSyncOutput(samplePTP4LLog)
	if res.RMS != 321 {
		t.Errorf("RMS = %d, want 321", res.RMS)
	}
	if res.Max != 512 {
		t.Errorf("Max = %d, want 512", res.Max)
	}
	if res.ClockClass != 159 {
		t.Errorf("ClockClass = %d, want 159", res.ClockClass)
	}
	if res.Count != 1 {
		t.Errorf("Count = %d, want 1", res.Count)
	}
}

func TestParseSyncOutputEmpty(t *testing.T) {
	res := parseSyncOutput("no measurements here\n")
	if res.RMS != 0 || res.Max != 0 || res.Count != 0 {
		t.Errorf("expected zero result, got %+v", res)
	}
}

func TestEndStateDetection(t *testing.T) {
	cases := []struct {
		line  string
		check string
		want  bool
	}{
		{"Kernel ptp clock changed", PTP4LEndState, true},
		{"Kernel ptp clock appeared", PTP4LEndState, true},
		{"port 1 became SLAVE", PTPPortStateEndState, true},
		{"port 1 became MASTER", PTPPortStateEndState, true},
		{"master offset      0 s2 freq   -1 path delay       123", "", false},
	}
	for _, c := range cases {
		if state, ok := endState(c.line); ok && state != c.check {
			t.Errorf("endState(%q) = %q, want %q", c.line, state, c.check)
		} else if !ok && c.want {
			t.Errorf("endState(%q) = false, want true", c.line)
		}
	}
}

func TestTimeoutStateDetection(t *testing.T) {
	if state, ok := timeoutState("clock_class 255"); !ok || state != ClockClassTimeoutState {
		t.Errorf("timeoutState(clock_class 255) = %q, %v; want %s, true", state, ok, ClockClassTimeoutState)
	}
	if _, ok := timeoutState("clock_class 159"); ok {
		t.Error("timeoutState(clock_class 159) should be false")
	}
}

func TestStatusFor(t *testing.T) {
	cfg := Config{MaxTaskTimeout: 2_000_000_000}
	if s := statusFor(SyncResult{RMS: 100, Max: 150}, cfg); s != StatusSynced {
		t.Errorf("got %s, want %s", s, StatusSynced)
	}
	if s := statusFor(SyncResult{RMS: 150, Max: 200}, Config{MaxTaskTimeout: 100}); s != StatusCritical {
		t.Errorf("got %s, want %s", s, StatusCritical)
	}
	if s := statusFor(SyncResult{RMS: 50, Max: 200, Count: 3}, cfg); s != StatusDegraded {
		t.Errorf("got %s, want %s", s, StatusDegraded)
	}
}

func TestOffsetArg(t *testing.T) {
	if got := offsetArg(123, 0); got != "+123" {
		t.Errorf("offsetArg(123) = %q, want +123", got)
	}
	if got := offsetArg(-123, 0); got != "-123" {
		t.Errorf("offsetArg(-123) = %q, want -123", got)
	}
	if got := offsetArg(0, 42); got != "+42" {
		t.Errorf("offsetArg(0, 42) = %q, want +42", got)
	}
}

type fakeRunner struct {
	keepRunning bool
	err         error
}

func (f fakeRunner) RunAndCapture(name string, args ...string) (string, error) {
	return "captured: " + name, f.err
}

func (f fakeRunner) StartRunning(name string, args ...string) (*RunningProcess, error) {
	return nil, nil
}

func TestSyncRequiresInterface(t *testing.T) {
	if _, err := Sync(fakeRunner{}, Config{}); err == nil {
		t.Fatal("expected error for missing interface")
	}
}

func TestSyncRunnerFailure(t *testing.T) {
	r := fakeRunner{err: errFake}
	cfg := Config{
		Interface: "eth0",
		PTP4LArgs: []string{"ptp4l", "-i", "eth0"},
	}
	if _, err := Sync(r, cfg); err == nil {
		t.Fatal("expected error from runner")
	}
}

var errFake = newFakeErr()

type fakeErr struct{ msg string }

func newFakeErr() error { return fakeErr{msg: "boom"} }
func (e fakeErr) Error() string {
	return e.msg
}
