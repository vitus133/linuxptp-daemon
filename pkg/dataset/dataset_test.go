package dataset

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func readTestData(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	testData, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(testData), nil
}

func TestParseParentDataSet(t *testing.T) {
	data, err := readTestData("test-data/parent.txt")
	if err != nil {
		t.Fatal(err)
	}

	pds, err := ParseParentDataSet(data)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "3c2c30.ffff.685900-9", pds.ParentPortIdentity)
	assert.Equal(t, 0, pds.ParentStats)
	assert.Equal(t, 0xffff, pds.ObservedParentOffsetScaledLogVariance)
	assert.Equal(t, 0x7fffffff, pds.ObservedParentClockPhaseChangeRate)
	assert.Equal(t, 128, pds.GrandmasterPriority1)
	assert.Equal(t, 165, pds.GmClockClass)
	assert.Equal(t, 0xfe, pds.GmClockAccuracy)
	assert.Equal(t, 0xffff, pds.GmOffsetScaledLogVariance)
	assert.Equal(t, 128, pds.GrandmasterPriority2)
	assert.Equal(t, "3c2c30.ffff.685900", pds.GrandmasterIdentity)

}

func TestParsePortDataSet(t *testing.T) {
	data, err := readTestData("test-data/port.txt")
	if err != nil {
		t.Fatal(err)
	}

	ports, err := ParsePortDataSet(data)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(ports))

	assert.Equal(t, "40a6b7.fffe.0e41b0-1", ports[0].portIdentity)
	assert.Equal(t, "40a6b7.fffe.0e41b0-2", ports[1].portIdentity)

	assert.Equal(t, "SLAVE", ports[0].portState)
	assert.Equal(t, "MASTER", ports[1].portState)

	assert.Equal(t, -4, ports[0].logMinDelayReqInterval)
	assert.Equal(t, -4, ports[1].logMinDelayReqInterval)

	assert.Equal(t, 0, ports[0].peerMeanPathDelay)
	assert.Equal(t, 0, ports[1].peerMeanPathDelay)

	assert.Equal(t, -3, ports[0].logAnnounceInterval)
	assert.Equal(t, -3, ports[1].logAnnounceInterval)

	assert.Equal(t, 3, ports[0].announceReceiptTimeout)
	assert.Equal(t, 3, ports[1].announceReceiptTimeout)

	assert.Equal(t, -4, ports[0].logSyncInterval)
	assert.Equal(t, -4, ports[1].logSyncInterval)

	assert.Equal(t, 1, ports[0].delayMechanism)
	assert.Equal(t, 1, ports[1].delayMechanism)

	assert.Equal(t, -4, ports[0].logMinPdelayReqInterval)
	assert.Equal(t, -4, ports[1].logMinPdelayReqInterval)

	assert.Equal(t, 2, ports[0].versionNumber)
	assert.Equal(t, 2, ports[1].versionNumber)

}

func TestParseCurrentDs(t *testing.T) {
	data, err := readTestData("test-data/current.txt")
	if err != nil {
		t.Fatal(err)
	}

	result, err := ParseCurrentDs(data)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, result.stepsRemoved)
	assert.Equal(t, 1.0, result.offsetFromMaster)
	assert.Equal(t, 354.0, result.meanPathDelay)
}
