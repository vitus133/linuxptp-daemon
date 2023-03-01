package bmca

import (
	"sort"

	"golang.org/x/exp/slices"
)

type BmcaPort struct {
	GmClockClass int
	// gmClockAccuracy           int
	// gmOffsetScaledLogVariance int
	// gmPriority2               int
	// gmClockIdentity           int
	// senderPortIdentity        int
	// stepsRemoved              int
	// receiverPortIdentity      int
	// receiverPortNumber        int
	// localPriority             int
	ConfigName string
}

const (
	GmClockClass              = "GmClockClass"
	GmClockAccuracy           = "GmClockAccuracy"
	GmOffsetScaledLogVariance = "GmOffsetScaledLogVariance"
	GmPriority2               = "GmPriority2"
	GmClockIdentity           = "GmClockIdentity"
	SenderPortIdentity        = "SenderPortIdentity"
	StepsRemoved              = "StepsRemoved"
	ReceiverPortIdentity      = "ReceiverPortIdentity"
	ReceiverPortNumber        = "ReceiverPortNumber"
	LocalPriority             = "LocalPriority"
)

func SortAndFilter(ports []BmcaPort, sortBy string) []BmcaPort {
	var idx int
	switch sortBy {
	case GmClockClass:
		sort.Slice(ports, func(i, j int) bool {
			return ports[i].GmClockClass < ports[j].GmClockClass
		})
		idx = slices.IndexFunc(ports, func(port BmcaPort) bool {
			return port.GmClockClass > ports[0].GmClockClass
		})
	}
	if idx != -1 {
		ports = slices.Delete(ports, idx, len(ports))
	}
	return ports
}
