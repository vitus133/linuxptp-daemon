package leap

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	leaphash "github.com/facebook/time/leaphash"
	"github.com/golang/glog"
	"github.com/openshift/linuxptp-daemon/pkg/ublox"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultLeapFileName    = "leap-seconds.list"
	defaultLeapFilePath    = "/usr/share/zoneinfo"
	gpsLeapToUtcLeap       = 37 - 18
	curreLsValidMask       = 0x1
	timeToLsEventValidMask = 0x2
	leapSourceGps          = 2
	leapConfigMapName      = "leap-configmap"
)

type LeapManager struct {
	// Ublox GNSS leap time indications channel
	UbloxLsInd chan ublox.TimeLs
	// Close channel
	Close chan bool
	// ts2phc path of leap-seconds.list file
	LeapFilePath string
	// client
	client    *kubernetes.Clientset
	namespace string
	// Leap file structure
	leapFile LeapFile
}

type LeapEvent struct {
	LeapTime string `json:"leapTime"`
	LeapSec  int    `json:"leapSec"`
	Comment  string `json:"comment"`
}
type LeapFile struct {
	ExpirationTime string      `json:"expirationTime"`
	UpdateTime     string      `json:"updateTime"`
	LeapEvents     []LeapEvent `json:"leapEvents"`
	Hash           string      `json:"hash"`
}

func New(kubeclient *kubernetes.Clientset, namespace string) (*LeapManager, error) {
	lm := &LeapManager{
		UbloxLsInd: make(chan ublox.TimeLs),
		Close:      make(chan bool),
		client:     kubeclient,
		namespace:  namespace,
		leapFile:   LeapFile{},
	}
	err := lm.PopulateLeapData()
	if err != nil {
		return nil, err
	}
	return lm, nil
}

func ParseLeapFile(b []byte) (*LeapFile, error) {
	var l = LeapFile{}
	lines := strings.Split(string(b), "\n")
	for i := 0; i < len(lines); i++ {
		fields := strings.Fields(lines[i])
		if strings.HasPrefix(lines[i], "#$") {
			l.UpdateTime = fields[1]
		} else if strings.HasPrefix(lines[i], "#@") {
			l.ExpirationTime = fields[1]
		} else if strings.HasPrefix(lines[i], "#h") {
			l.Hash = strings.Join(fields[1:], " ")
		} else if !strings.HasPrefix(lines[i], "#") {
			if len(fields) < 2 {
				// empty line
				continue
			}
			sec, err := strconv.ParseInt(fields[1], 10, 8)
			if err != nil {
				return nil, fmt.Errorf("failed to parse Leap seconds %s value from file: %s, %v", fields[1], defaultLeapFileName, err)
			}
			ev := LeapEvent{
				LeapTime: fields[0],
				LeapSec:  int(sec),
				Comment:  strings.Join(fields[2:], " "),
			}
			l.LeapEvents = append(l.LeapEvents, ev)
		}
	}
	return &l, nil
}

func (l *LeapManager) RenderLeapData() (*bytes.Buffer, error) {
	templateFile := "templates/leap-seconds.list.template"
	templ, err := template.ParseFiles(templateFile)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	bufWriter := bufio.NewWriter(&buf)

	err = templ.Execute(bufWriter, l.leapFile)
	if err != nil {
		return nil, err
	}
	bufWriter.Flush()
	return &buf, nil
}

func (l *LeapManager) PopulateLeapData() error {
	cm, err := l.client.CoreV1().ConfigMaps(l.namespace).Get(context.TODO(), leapConfigMapName, metav1.GetOptions{})
	nodeName := os.Getenv("NODE_NAME")
	if err != nil {
		return err
	}
	lf, found := cm.Data[nodeName]
	if !found {
		b, err := os.ReadFile(filepath.Join(defaultLeapFilePath, defaultLeapFileName))
		if err != nil {
			return err
		}
		leapData, err := ParseLeapFile(b)
		if err != nil {
			return err
		}
		l.leapFile = *leapData

		if len(cm.Data) == 0 {
			cm.Data = map[string]string{}
		}
		cm.Data[nodeName] = string(b)
		_, err = l.client.CoreV1().ConfigMaps(l.namespace).Update(context.TODO(), cm, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	} else {
		leapData, err := ParseLeapFile([]byte(lf))
		if err != nil {
			return err
		}
		l.leapFile = *leapData
	}
	return nil
}

func (l *LeapManager) SetLeapFile(leapFile string) {
	l.LeapFilePath = leapFile
	glog.Info("setting Leap file to ", leapFile)
}

func (l *LeapManager) Run() {
	glog.Info("starting Leap file manager")
	for {
		select {
		case v := <-l.UbloxLsInd:
			l.HandleLeapIndication(&v)
		case <-l.Close:
			return
		}
	}
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
	return fmt.Errorf("integrity error: %s - on Leap file, %s - computed",
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
