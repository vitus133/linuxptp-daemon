package daemon

import (
	"errors"
	"os"
	"testing"

	v1 "github.com/openshift/ptp-operator/api/v1"
)

func readProfiles(nodeProfile string) ([]v1.PtpProfile, error) {
	if _, err := os.Stat(nodeProfile); err != nil {
		return []v1.PtpProfile{}, err
	}
	nodeProfilesJson, err := os.ReadFile(nodeProfile)
	if err != nil {
		return []v1.PtpProfile{}, err
	}
	nodeProfiles, ok := tryToLoadConfig(nodeProfilesJson)
	if !ok {
		return []v1.PtpProfile{}, errors.New("error loading node profiles")
	}
	return nodeProfiles, nil
}

func TestIsSingleTTsc(t *testing.T) {
	nodeProfiles, err := readProfiles("test-data/ha-valid.json")
	if err != nil {
		t.Fatal(err)
	}

	expected := []bool{false, true, true}
	for i, want := range expected {
		result := isSingleTTsc(nodeProfiles[i])
		if want != result {
			t.Fatalf("isSingleTTsc(nodeProfiles[0]) must be %t, but it's %t", want, result)
		}
	}
}

func TestIsHighAvailability(t *testing.T) {
	var daemon Daemon
	var confUpdate LinuxPTPConfUpdate
	nodeProfiles, err := readProfiles("test-data/ha-valid.json")
	if err != nil {
		t.Fatal(err)
	}
	confUpdate.NodeProfiles = nodeProfiles
	daemon.ptpUpdate = &confUpdate
	want := true

	result := daemon.isHighAvailability()

	if want != result {
		t.Fatalf("isHighAvailability() of ha-valid.json profile must be %t, but it's %t", want, result)
	}

}
