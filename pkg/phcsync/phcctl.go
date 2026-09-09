package phcsync

import "fmt"

// PHCReadArgs returns the phc_ctl invocation that reads the current PHC time
// into the kernel clock.
func PHCReadArgs(clockDevice string) []string {
	if clockDevice == "" {
		clockDevice = "/dev/ptp0"
	}
	return []string{"phc_ctl", clockDevice, "get"}
}

// PHCWriteArgs returns the phc_ctl invocation that applies a corrected time to
// the PHC clock.
func PHCWriteArgs(clockDevice, correction string) []string {
	if clockDevice == "" {
		clockDevice = "/dev/ptp0"
	}
	return []string{"phc_ctl", clockDevice, "set", "-n", "-C", correction}
}

// PHCCtlError wraps a phc_ctl failure.
type PHCCtlError struct {
	Args []string
	Out  string
	err  error
}

func (e *PHCCtlError) Error() string {
	return fmt.Sprintf("phcsync: phc_ctl %v failed: %v", e.Args, e.err)
}

func (e *PHCCtlError) Unwrap() error { return e.err }
