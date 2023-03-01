package daemon

import (
	"strings"

	"github.com/golang/glog"
	v1 "github.com/openshift/ptp-operator/api/v1"
	ini "gopkg.in/ini.v1"
)

const tTscOnlyDirective = "masterOnly"

// isSingleTTsc checks ptp profile for a condition of a single T-TSC
func isSingleTTsc(profile v1.PtpProfile) bool {
	if profile.Ptp4lConf == nil {
		glog.Info("No ptp4lConf in profile ", profile.Name)
		return false
	}
	confData := string(*profile.Ptp4lConf)
	cfg, err := ini.LoadSources(ini.LoadOptions{KeyValueDelimiters: " "}, []byte(confData))
	if err != nil {
		glog.Error("failed to load ptp4lConf from ", profile.Name, ", error ", err)
		return false
	}
	sections := cfg.Sections()
	tBcOnlyCount := 0
	interfaceCount := 0
	for _, section := range sections {
		if strings.Contains(section.Name(), "global") ||
			strings.Contains(section.Name(), "unicast_master_table") ||
			section.Name() == "DEFAULT" || section.Name() == "global" {
			continue
		}
		interfaceCount += 1

		tBcOnlyFlag, err := section.Key(tTscOnlyDirective).Int()
		if err != nil {
			// In HA mode each interface section must have the "masterOnly" directive
			return false
		}
		tBcOnlyCount += tBcOnlyFlag
	}

	return interfaceCount-tBcOnlyCount == 1
}

// isHighAvailability does input verification to determine the operational mode.
//
// The operational mode can be either standard availability, or high availability.
func (dn *Daemon) isHighAvailability() bool {
	phc2sysCount := 0
	ptp4lCount := 0
	for _, profile := range dn.ptpUpdate.NodeProfiles {
		if profile.Ptp4lOpts != nil && profile.Phc2sysOpts != nil {
			glog.Info("Operating in the standard availability mode")
			return false
		}
		if profile.Phc2sysOpts != nil {
			phc2sysCount += 1
		}
		if profile.Ptp4lOpts != nil {
			// TODO: Commenting this out to allow for ordinary clocks HA testing
			// if !isSingleTTsc(profile) {
			// 	glog.Infof("Operating in the standard availability mode: %v", profile)
			// 	return false
			// }
			ptp4lCount += 1
		}
	}
	if phc2sysCount == 1 && ptp4lCount >= 1 {
		glog.Info("Operating in the high availability mode")
		return true
	}
	return false
}
