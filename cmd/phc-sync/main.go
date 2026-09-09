package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang/glog"

	"github.com/k8snetworkplumbingwg/linuxptp-daemon/pkg/phcsync"
)

var (
	version   = "0.1.0"
	GitCommit string
)

func versionString() string {
	if GitCommit != "" {
		return version + " (" + GitCommit + ")"
	}
	return version
}

func main() {
	fs := flag.NewFlagSet("phc-sync", flag.ExitOnError)
	iface := fs.String("interface", "", "PTP interface to synchronize")
	iterations := fs.Int("iterations", 3, "number of measurement iterations to run")
	logDir := fs.String("log-dir", "/var/log/ptp", "directory for ptp4l configuration and logs")
	clockDevice := fs.String("ptp-device", "/dev/ptp0", "PHC clock device used by phc_ctl")
	upPeriod := fs.Duration("up-period", 10*time.Minute, "maximum actionable holdover period")
	maxTimeout := fs.Duration("max-timeout", 10*time.Minute, "task timeout for the ptp4l session")
	showVersion := fs.Bool("version", false, "print version and exit")
	_ = flag.CommandLine.Parse(nil)
	if err := fs.Parse(os.Args[1:]); err != nil {
		glog.Fatalf("phc-sync: parse flags: %v", err)
	}
	if *showVersion {
		fmt.Printf("phc-sync %s\n", versionString())
		return
	}
	if *iface == "" {
		fmt.Fprintln(os.Stderr, "phc-sync: -interface is required")
		os.Exit(1)
	}

	cfg := phcsync.Config{
		Interface:      *iface,
		Iterations:     *iterations,
		OutputDir:      *logDir,
		PTP4LArgs:      phcsync.PTP4LArgs(*iface, *logDir),
		PHCReadArgs:    phcsync.PHCReadArgs(*clockDevice),
		PHCWriteArgs:   phcsync.PHCWriteArgs(*clockDevice, ""),
		UpPeriod:       *upPeriod,
		MaxTaskTimeout: *maxTimeout,
	}

	logf := glog.Infof
	out, err := phcsync.RunPTP4LStream(phcsync.PTP4LConfig{
		Interface:  *iface,
		LogDir:     *logDir,
		Iterations: *iterations,
		Logger:     logf,
	}, *maxTimeout)
	if err != nil {
		glog.Warningf("phc-sync: ptp4l session: %v", err)
	}

	runner := phcsync.LoggingRunner{Inner: phcsync.OSRunner{}, Log: logf}
	status, err := phcsync.SyncFromOutput(runner, cfg, out)
	if err != nil {
		glog.Fatalf("phc-sync: synchronization failed: %v", err)
	}
	glog.Infof("phc-sync: status %s", status)
}
