package phcsyncworkaround

import (
	"context"
	"strings"
	"testing"
	"time"

	ptpv1 "github.com/k8snetworkplumbingwg/ptp-operator/api/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestMeasurementTimeout(t *testing.T) {
	tests := []struct {
		name    string
		profile *ptpv1.PtpProfile
		want    time.Duration
		wantErr bool
	}{
		{name: "nil profile", want: 0},
		{name: "no plugins", profile: &ptpv1.PtpProfile{}, want: 0},
		{name: "unset", profile: profileWithPlugin(`{}`), want: 0},
		{name: "configured", profile: profileWithPlugin(`{"timeout":"15s"}`), want: 15 * time.Second},
		{name: "invalid", profile: profileWithPlugin(`{"timeout":"bad"}`), wantErr: true},
		{name: "negative", profile: profileWithPlugin(`{"timeout":"-1s"}`), wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := measurementTimeout(test.profile)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected timeout parsing error, got %s", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("measurementTimeout() error: %v", err)
			}
			if got != test.want {
				t.Fatalf("measurementTimeout() = %s, want %s", got, test.want)
			}
		})
	}
}

func profileWithPlugin(raw string) *ptpv1.PtpProfile {
	return &ptpv1.PtpProfile{Plugins: map[string]*apiextensions.JSON{
		pluginName: {Raw: []byte(raw)},
	}}
}

func TestNew(t *testing.T) {
	got, data := New(pluginName)
	if got == nil || data == nil || got.Name != pluginName || got.OnPTPConfigChange == nil {
		t.Fatalf("unexpected plugin registration: plugin=%+v data=%v", got, data)
	}
	if got, data := New("other"); got != nil || data != nil {
		t.Fatalf("expected wrong plugin name to be rejected, got plugin=%v data=%v", got, data)
	}
}

func TestProfileTimeReceiverPortsAndConfig(t *testing.T) {
	conf := "[global]\ndomainNumber 24\ndataset_comparison G.8275.x\n" +
		"[eno1]\nmasterOnly 0\n" +
		"[eno2]\nmasterOnly 0\n" +
		"[eno3]\nmasterOnly 1\n"
	profile := &ptpv1.PtpProfile{Ptp4lConf: &conf}
	ports := upstreamPortsFromPtpProfile(profile)
	if want := []string{"eno1", "eno2"}; !equalStrings(ports, want) {
		t.Fatalf("upstream ports = %v, want %v", ports, want)
	}
	got, err := renderMeasurementConfig(profile, ports)
	if err != nil {
		t.Fatalf("renderMeasurementConfig() error: %v", err)
	}
	for _, want := range []string{"domainNumber 24", "dataset_comparison G.8275.x", "[eno1]", "[eno2]", "masterOnly 0"} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered config missing %q:\n%s", want, got)
		}
	}
	if strings.Count(got, "masterOnly 0") != 2 {
		t.Errorf("expected both TR sections in rendered config:\n%s", got)
	}
}

func TestUpstreamPortsFromPtpProfile(t *testing.T) {
	tests := []struct {
		name   string
		config *string
		want   []string
	}{
		{name: "nil profile config"},
		{name: "global section is not an interface", config: stringPointer("[global]\nmasterOnly 0\n")},
		{
			name:   "one upstream port",
			config: stringPointer("[global]\ndomainNumber 24\n[eno1]\nmasterOnly 0\n[eno2]\nmasterOnly 1\n"),
			want:   []string{"eno1"},
		},
		{
			name:   "multiple upstream ports preserve order",
			config: stringPointer("[eno2]\nmasterOnly 0\n[eno1]\nmasterOnly 0\n"),
			want:   []string{"eno2", "eno1"},
		},
		{
			name:   "non-interface sections are ignored",
			config: stringPointer("[nmea]\nmasterOnly 0\n[unicast_master_table]\nmasterOnly 0\n[eno1]\nmasterOnly 0\n"),
			want:   []string{"eno1"},
		},
		{
			name:   "unrecognized ptp4l section is treated as a port section",
			config: stringPointer("[unicast]\nmasterOnly 0\n"),
			want:   []string{"unicast"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile := &ptpv1.PtpProfile{Ptp4lConf: test.config}
			got := upstreamPortsFromPtpProfile(profile)
			if !equalStrings(got, test.want) {
				t.Fatalf("upstreamPortsFromPtpProfile() = %v, want %v", got, test.want)
			}
		})
	}
	if got := upstreamPortsFromPtpProfile(nil); len(got) != 0 {
		t.Fatalf("nil profile returned upstream ports: %v", got)
	}
}

func stringPointer(value string) *string {
	return &value
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestParseSampleRequiresNonZeroPathDelay(t *testing.T) {
	tests := []struct {
		line       string
		wantOffset int64
		wantValid  bool
	}{
		{line: "ptp4l[544.425]: master offset 0 s0 freq +48 path delay 80", wantValid: true},
		{line: "ptp4l[544.487]: master offset 0 s0 freq -16 path delay 0"},
		{line: "ptp4l[544.487]: master offset 0 s0 freq -16 path delay +0"},
		{line: "ptp4l[1.0]: [ptp4l.1.config:6] eno1 master offset -19 s0 freq +1 path delay 83", wantOffset: -19, wantValid: true},
	}
	for _, test := range tests {
		got, valid := parseSample(test.line)
		if valid != test.wantValid || valid && got != test.wantOffset {
			t.Errorf("parseSample(%q) = (%d, %v), want (%d, %v)", test.line, got, valid, test.wantOffset, test.wantValid)
		}
	}
}

func TestCommandOutputUntilDrainsOutput(t *testing.T) {
	out, err := commandOutputUntil(context.Background(), nil, "sh", "-c", "printf 'first\\nlast\\n'")
	if err != nil {
		t.Fatalf("commandOutputUntil() error: %v", err)
	}
	if out != "first\nlast\n" {
		t.Fatalf("captured output %q, want both lines", out)
	}
}

func TestCorrectedTime(t *testing.T) {
	const phc = int64(1_788_982_998_578_115_938)
	got, err := correctedTime(phc, 318_984_375_136)
	if err != nil {
		t.Fatalf("correctedTime() error: %v", err)
	}
	if want := phc - 318_984_375_136; got != want {
		t.Fatalf("correctedTime() = %d, want %d", got, want)
	}
}

func TestPHCSecondsToNS(t *testing.T) {
	got, err := phcSecondsToNS("1788982998.578115938")
	if err != nil {
		t.Fatalf("phcSecondsToNS() error: %v", err)
	}
	if got != 1_788_982_998_578_115_938 {
		t.Fatalf("phcSecondsToNS() = %d", got)
	}
}

func TestFormatNS(t *testing.T) {
	if got, want := formatNS(1_788_982_998_578_115_938), "1788982998.578115938"; got != want {
		t.Fatalf("formatNS() = %q, want %q", got, want)
	}
	if got := len(formatNS(1_788_982_998_578_115_938)); got != 20 {
		t.Fatalf("unexpected seconds format width: %d", got)
	}
}
