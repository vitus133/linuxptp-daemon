package dataset

import (
	"bufio"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

type ParentDs struct {
	ParentPortIdentity                    string
	ParentStats                           int
	ObservedParentOffsetScaledLogVariance int
	ObservedParentClockPhaseChangeRate    int
	GrandmasterPriority1                  int
	GmClockClass                          int
	GmClockAccuracy                       int
	GmOffsetScaledLogVariance             int
	GrandmasterPriority2                  int
	GrandmasterIdentity                   string
}

type PortDs struct {
	portIdentity            string
	portState               string
	logMinDelayReqInterval  int
	peerMeanPathDelay       int
	logAnnounceInterval     int
	announceReceiptTimeout  int
	logSyncInterval         int
	delayMechanism          int
	logMinPdelayReqInterval int
	versionNumber           int
}

type CurrentDs struct {
	stepsRemoved     int
	offsetFromMaster float64
	meanPathDelay    float64
}

func parseHexToInt(input string) (int, error) {
	n := new(big.Int)
	val, ok := n.SetString(strings.Split(input, "x")[1], 16)
	if !ok {
		return 0, fmt.Errorf("failed to parse %s to int", input)
	}
	return int(val.Int64()), nil
}

func ParseParentDataSet(data string) (ParentDs, error) {
	resp := strings.Split(data, "PARENT_DATA_SET")[2]
	whiteSpaces := regexp.MustCompile(`[[:blank:]]+`)
	resp = strings.ReplaceAll(whiteSpaces.ReplaceAllString(resp, "="), "\n=", "\n")

	var pds = new(ParentDs)
	scanner := bufio.NewScanner(strings.NewReader(resp))
	for scanner.Scan() {
		kv := strings.Split(scanner.Text(), "=")
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "parentPortIdentity":
			pds.ParentPortIdentity = kv[1]
		case "parentStats":
			v, err := strconv.ParseInt(kv[1], 0, 8)
			if err != nil {
				return *pds, err
			}
			pds.ParentStats = int(v)
		case "observedParentOffsetScaledLogVariance":
			val, err := parseHexToInt(kv[1])
			if err != nil {
				return *pds, err
			}
			pds.ObservedParentOffsetScaledLogVariance = val
		case "observedParentClockPhaseChangeRate":
			val, err := parseHexToInt(kv[1])
			if err != nil {
				return *pds, err
			}
			pds.ObservedParentClockPhaseChangeRate = val
		case "grandmasterPriority1":
			v, err := strconv.ParseInt(kv[1], 0, 16)
			if err != nil {
				return *pds, err
			}
			pds.GrandmasterPriority1 = int(v)
		case "gm.ClockClass":
			v, err := strconv.ParseInt(kv[1], 0, 16)
			if err != nil {
				return *pds, err
			}
			pds.GmClockClass = int(v)
		case "gm.ClockAccuracy":
			val, err := parseHexToInt(kv[1])
			if err != nil {
				return *pds, err
			}
			pds.GmClockAccuracy = val
		case "gm.OffsetScaledLogVariance":
			val, err := parseHexToInt(kv[1])
			if err != nil {
				return *pds, err
			}
			pds.GmOffsetScaledLogVariance = val
		case "grandmasterPriority2":
			v, err := strconv.ParseInt(kv[1], 0, 16)
			if err != nil {
				return *pds, err
			}
			pds.GrandmasterPriority2 = int(v)
		case "grandmasterIdentity":
			pds.GrandmasterIdentity = kv[1]
		}
	}
	return *pds, nil

}

func ParsePortDataSet(data string) ([]PortDs, error) {
	resp := strings.Split(data, "GET PORT_DATA_SET\n")[1]

	var ports = []PortDs{}
	var portAppend = false
	var port PortDs

	scanner := bufio.NewScanner(strings.NewReader(resp))
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "RESPONSE MANAGEMENT PORT_DATA_SET") {
			if !portAppend {
				port = PortDs{}
				portAppend = true
			} else {
				ports = append(ports, port)
				port = PortDs{}
			}
			continue
		}

		whiteSpaces := regexp.MustCompile(`[[:blank:]]+`)
		kv := strings.Split(whiteSpaces.ReplaceAllString(scanner.Text(), "="), "=")

		if len(kv) != 3 {
			continue
		}
		switch kv[1] {
		case "portIdentity":
			port.portIdentity = kv[2]
		case "portState":
			port.portState = kv[2]
		case "logMinDelayReqInterval":
			v, err := strconv.ParseInt(kv[2], 0, 16)
			if err != nil {
				return ports, err
			}
			port.logMinDelayReqInterval = int(v)
		case "peerMeanPathDelay":
			v, err := strconv.ParseInt(kv[2], 0, 16)
			if err != nil {
				return ports, err
			}
			port.peerMeanPathDelay = int(v)
		case "logAnnounceInterval":
			v, err := strconv.ParseInt(kv[2], 0, 16)
			if err != nil {
				return ports, err
			}
			port.logAnnounceInterval = int(v)
		case "announceReceiptTimeout":
			v, err := strconv.ParseInt(kv[2], 0, 16)
			if err != nil {
				return ports, err
			}
			port.announceReceiptTimeout = int(v)
		case "logSyncInterval":
			v, err := strconv.ParseInt(kv[2], 0, 16)
			if err != nil {
				return ports, err
			}
			port.logSyncInterval = int(v)
		case "delayMechanism":
			v, err := strconv.ParseInt(kv[2], 0, 16)
			if err != nil {
				return ports, err
			}
			port.delayMechanism = int(v)
		case "logMinPdelayReqInterval":
			v, err := strconv.ParseInt(kv[2], 0, 16)
			if err != nil {
				return ports, err
			}
			port.logMinPdelayReqInterval = int(v)
		case "versionNumber":
			v, err := strconv.ParseInt(kv[2], 0, 16)
			if err != nil {
				return ports, err
			}
			port.versionNumber = int(v)

		}
	}
	ports = append(ports, port)
	return ports, nil
}

func ParseCurrentDs(data string) (CurrentDs, error) {
	scanner := bufio.NewScanner(strings.NewReader(data))
	var current CurrentDs
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "CURRENT_DATA_SET") {
			continue
		}
		whiteSpaces := regexp.MustCompile(`[[:blank:]]+`)
		kv := strings.Split(whiteSpaces.ReplaceAllString(scanner.Text(), "="), "=")
		if len(kv) != 3 {
			continue
		}
		switch kv[1] {
		case "stepsRemoved":
			v, err := strconv.ParseInt(kv[2], 0, 16)
			if err != nil {
				return current, err
			}
			current.stepsRemoved = int(v)

		case "offsetFromMaster":
			v, err := strconv.ParseFloat(kv[2], 64)
			if err != nil {
				return current, err
			}
			current.offsetFromMaster = v

		case "meanPathDelay":
			v, err := strconv.ParseFloat(kv[2], 64)
			if err != nil {
				return current, err
			}
			current.meanPathDelay = v

		}

	}
	return current, nil

}
