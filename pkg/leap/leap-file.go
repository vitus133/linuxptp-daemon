package leap

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	leaphash "github.com/facebook/time/leaphash"
	"github.com/golang/glog"
	"github.com/openshift/linuxptp-daemon/pkg/ublox"
)

const (
	userLeapFileName       = "leap-seconds.list"
	gpsLeapToUtcLeap       = 37 - 18
	curreLsValidMask       = 0x1
	timeToLsEventValidMask = 0x2
	leapSourceGps          = 2
)

type LeapManager struct {
	// Ublox GNSS leap time indications channel
	UbloxLsInd chan ublox.TimeLs
	// Close channel
	Close chan bool
	// ts2phc path of leap-seconds.list file
	LeapFilePath string
	// Path of base64 encoded user-provided leap-seconds.list file
	UserLeapFilePath string
	// LeapFileUpdateFromGpsEnabled - allow updating system leap file
	// from GNSS TIMELS indication.
	// Enabled by default and becomes disabled if leap-seconds.list file
	// is created by user (through a configmap)
	// It becomes re-enabled again when the user-provided leap-seconds.list is deleted
	LeapFileUpdateFromGpsEnabled bool
}

func New(leapUserConfigDir string) *LeapManager {
	return &LeapManager{
		UbloxLsInd:                   make(chan ublox.TimeLs),
		UserLeapFilePath:             leapUserConfigDir,
		Close:                        make(chan bool),
		LeapFileUpdateFromGpsEnabled: true,
	}
}

func (l *LeapManager) SetLeapFile(leapFile string) {
	l.LeapFilePath = leapFile
	glog.Info("setting Leap file to ", leapFile)
}

func (l *LeapManager) Run() {
	glog.Info("starting Leap file manager")
	tickerPull := time.NewTicker(30 * time.Second)
	defer tickerPull.Stop()
	for {
		select {
		case v := <-l.UbloxLsInd:
			l.HandleLeapIndication(&v)
		case <-tickerPull.C:
			glog.Info("Leap file check ticker")
			if equal, uf := l.checkLeapFileDiff(); !equal && uf != nil {
				glog.Info("user Leap file exists, valid and different from the system leap file: replacing the system leap file")
				err := os.WriteFile(l.LeapFilePath, *uf, 0644)
				if err != nil {
					glog.Info("failed to update system Leap file")
				}
				l.LeapFileUpdateFromGpsEnabled = false
			}
		case <-l.Close:
			return
		}
	}
}

// checkLeapFileDiff checks whether the user-provided leap seconds
// file exists, valid and different from the system leap-seconds file
func (l *LeapManager) checkLeapFileDiff() (bool, *[]byte) {
	userFile := filepath.Join(l.UserLeapFilePath, userLeapFileName)
	if _, err := os.Stat(userFile); err != nil {
		glog.Info("no user Leap file provided")
		l.LeapFileUpdateFromGpsEnabled = true
		return false, nil
	}
	if err := CheckLeapFileIntegrity(userFile); err != nil {
		glog.Info("Leap file integrity check error: ", err)
		return false, nil
	}
	if _, err := os.Stat(l.LeapFilePath); err != nil {
		glog.Info("system Leap file path is not yet initialized")
		return false, nil
	}
	equal, uf, err := l.leapFilesEqual()
	if err != nil {
		glog.Info("failed to compare Leap files: ", err)
		return false, nil
	}
	return equal, uf
}

func (l *LeapManager) leapFilesEqual() (bool, *[]byte, error) {
	userFile := filepath.Join(l.UserLeapFilePath, userLeapFileName)
	uf, err := os.ReadFile(userFile)
	if err != nil {
		return false, nil, err
	}
	sf, err := os.ReadFile(l.LeapFilePath)
	if err != nil {
		return false, nil, err
	}
	equal := bytes.Equal(uf, sf)
	return equal, &uf, nil
}

func GetLastLeapEventFromFile(fp string) (*time.Time, int, error) {
	b, err := os.ReadFile(fp)
	if err != nil {
		return nil, 0, err
	}
	// TODO: Add integrity check
	lines := strings.Split(string(b), "\n")
	var leapSection = false
	for i := 0; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], "#") {
			leapSection = true
		}
		if leapSection && strings.HasPrefix(lines[i], "#") {
			// line i-1 contains the last leap
			event := strings.Fields(lines[i-1])
			if len(event) < 2 {
				return nil, 0, fmt.Errorf("failed to get last Leap event from file %s: %v", fp, err)
			}
			delta, err := time.ParseDuration(event[0] + "s")
			if err != nil {
				return nil, 0, fmt.Errorf("failed to convert last Leap event duration %s: %v", event[0], err)
			}

			startTime := time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)
			leapEventTime := startTime.Add(delta)

			leapSec, err := strconv.Atoi(event[1])
			if err != nil {
				return nil, 0, fmt.Errorf("failed to convert Leap seconds to int %s: %v", event[1], err)
			}
			return &leapEventTime, leapSec, nil
		}
	}
	return nil, 0, fmt.Errorf("can't find last Leap event in file %s: %v", fp, err)
}

func CheckLeapFileIntegrity(filePath string) error {
	var hashOnFile string
	b, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")

	// going backwards from the end of file
	for i := len(lines) - 1; i >= 0; i-- {
		// line starts from the hash sign
		if strings.HasPrefix(lines[i], "#h") {
			hashOnFile = strings.Join(strings.Fields(lines[i])[1:], " ")
			break
		}
	}
	hash := leaphash.Compute(string(b))
	if strings.Compare(hash, hashOnFile) == 0 {
		return nil
	}
	return fmt.Errorf("Leap file integrity error: %s - on file, %s - computed",
		hashOnFile, hash)
}

// AddLeapEvent appends a leap event to the leap-file.list
func AddLeapEvent(filepath string, leapTime time.Time,
	leapSec int, expirationTime time.Time, currentTime time.Time) error {
	startTime := time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)
	leap := int(leapTime.Sub(startTime).Seconds())
	modified := int(currentTime.Sub(startTime).Seconds())
	expired := int(expirationTime.Sub(startTime).Seconds())
	//fmt.Printf("leap %d, mod %d, exp %d", leap, modified, expired)
	b, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")

	out := []string{}
	var leapSection = false
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "#$") {
			// "#$" - last modification time
			lines[i] = strings.Join([]string{"#$", fmt.Sprintf("%d", modified)}, "\t")
		} else if strings.HasPrefix(lines[i], "#@") {
			// "#@" - the expiration time of the file
			lines[i] = strings.Join([]string{"#@", fmt.Sprintf("%d", expired)}, "\t")
		}

		if !strings.HasPrefix(lines[i], "#") {
			leapSection = true
		}
		if leapSection && strings.HasPrefix(lines[i], "#") {
			// Insert the new leap event
			out = append(out, strings.Join(
				[]string{fmt.Sprintf("%d", leap),
					"    ",
					fmt.Sprintf("%d", leapSec),
					"    ",
					"#", fmt.Sprintf("%v %v %v",
						leapTime.Day(), leapTime.Month().String()[:3], leapTime.Year())}, " "))
			leapSection = false
		}
		if strings.HasPrefix(lines[i], "#h") {
			// "#h" - the hash
			// Compute the hash and add it to the end of file
			s := strings.Join(out, "\n")
			hash := leaphash.Compute(s)
			lines[i] = strings.Join([]string{"#h", hash}, "\t")
		}
		out = append(out, lines[i])
	}
	return os.WriteFile(filepath, []byte(strings.Join(out, "\n")), 0644)
}

// HandleLeapIndication handles NAV-TIMELS indication
// and updates the leapseconds.list file
// If leap event is closer than 12 hours in the future,
// GRANDMASTER_SETTINGS_NP dataset will be updated with
// the up to date leap second information

func (l *LeapManager) HandleLeapIndication(data *ublox.TimeLs) {
	glog.Infof("Leap indication: %+v", data)
	if data.SrcOfCurrLs != leapSourceGps {
		glog.Info("Discarding Leap event not opriginating from GPS")
		return
	}
	if !l.LeapFileUpdateFromGpsEnabled {
		glog.Info("Automatic Leap file update is disabled - user override found")
		return
	}
	_, leapSecOnFile, err := GetLastLeapEventFromFile(l.LeapFilePath)
	if err != nil {
		glog.Error("Leap file error: ", err)
		return
	}

	validCurrLs := (data.Valid & curreLsValidMask) > 0
	validTimeToLsEvent := (data.Valid & timeToLsEventValidMask) > 0
	expirationTime := time.Now().UTC().Add(time.Hour * 87660)
	currentTime := time.Now().UTC()

	if validTimeToLsEvent && validCurrLs {
		leapSec := int(data.CurrLs) + gpsLeapToUtcLeap + int(data.LsChange)
		if data.LsChange != 0 && data.TimeToLsEvent > -1 && data.TimeToLsEvent < 12*3600 {
			// TODO: Run PMC command to announce the leap downstream
			glog.Info("Leap event is within ", data.TimeToLsEvent, " sec.")
		}
		if leapSec != leapSecOnFile {
			// File update is needed
			glog.Infof("Leap Seconds on file outdated: %d on file, %d + %d + %d in GNSS data",
				leapSecOnFile, int(data.CurrLs), gpsLeapToUtcLeap, int(data.LsChange))
			startTime := time.Date(1980, time.January, 6, 0, 0, 0, 0, time.UTC)
			deltaHours, err := time.ParseDuration(fmt.Sprintf("%dh",
				data.DateOfLsGpsWn*7*24+uint(data.DateOfLsGpsDn)*24))
			if err != nil {
				glog.Error("Leap:", err)
				return
			}
			leapTime := startTime.Add(deltaHours)
			err = AddLeapEvent(l.LeapFilePath, leapTime, leapSec, expirationTime, currentTime)
			if err != nil {
				glog.Error("Leap :", err)
				return
			}
		}
	}
}
