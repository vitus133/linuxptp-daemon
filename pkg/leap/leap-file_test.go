package leap

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_AddLeapEvent(t *testing.T) {
	leapFile := "testdata/leap-seconds.list"
	testFile := "/tmp/leap-seconds.list"
	leapTime := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	expirationTime := time.Date(2035, time.January, 1, 0, 0, 0, 0, time.UTC)
	currentTime := time.Date(2024, time.May, 8, 0, 0, 0, 0, time.UTC)
	leapSec := 38
	// Create the test file
	input, err := os.ReadFile(leapFile)
	assert.Equal(t, nil, err)
	err = os.WriteFile(testFile, input, 0644)
	assert.Equal(t, nil, err)
	err = AddLeapEvent(testFile, leapTime, leapSec, expirationTime, currentTime)
	assert.Equal(t, nil, err)
	// Compare files
	generated, err := os.ReadFile(testFile)
	assert.True(t, err == nil)
	desired, err := os.ReadFile("testdata/leap-seconds.list.desired")
	assert.Equal(t, nil, err)
	assert.True(t, bytes.Equal(generated, desired))
	// cleanup
	err = os.Remove(testFile)
	assert.Equal(t, nil, err)
}

func Test_GetLastLeapEventFromFile(t *testing.T) {
	leapFile := "testdata/leap-seconds.list"
	td, leap, err := GetLastLeapEventFromFile(leapFile)
	assert.Equal(t, nil, err)
	expectedLeap := time.Date(2017, time.January, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expectedLeap, *td)
	assert.Equal(t, 37, leap)
	fmt.Println(td)
}

func Test_ParseLeapFile(t *testing.T) {
	leapFile := "testdata/leap-seconds.list"
	b, err := os.ReadFile(leapFile)
	assert.Equal(t, nil, err)
	_, err = ParseLeapFile(b)
	assert.Equal(t, nil, err)
}
