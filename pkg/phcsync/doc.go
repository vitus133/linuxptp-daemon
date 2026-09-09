// Package phcsync implements the PHC time synchronization utility.
//
// It runs a short ptp4l free-running measurement session on an upstream PTP
// port, reads the current PHC time with phc_ctl, computes a corrected time
// that nulls the measured offset and writes it back with phc_ctl.
package phcsync
