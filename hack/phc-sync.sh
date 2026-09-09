#!/bin/bash
# phc-sync.sh — live diagnostic for PHC time synchronization.
#
# Prints every command before it runs and streams its output in real time.
# Nothing is gated on measurements: ptp4l runs for [capture-seconds] and then
# stops, so the script always returns.
#
# Usage:
#   phc-sync.sh <interface> [capture-seconds]
#
# Environment:
#   LOG_DIR     (default /var/log/ptp)  path for ptp4l config and logs
#   PTP_DEVICE  (default /dev/ptp0)     PHC device for phc_ctl

IFACE="${1:?usage: phc-sync.sh <interface> [capture-seconds]}"
SECS="${2:-5}"
LOG_DIR="${LOG_DIR:-/var/log/ptp}"
DEV="${PTP_DEVICE:-/dev/ptp0}"

# run prints the exact command, then executes it.
run() {
  printf '+ %s\n' "$*"
  "$@"
}

mkdir -p "${LOG_DIR}/${IFACE}"

CFG="${LOG_DIR}/default.cfg"
if [ ! -f "${CFG}" ]; then
  printf '\nwriting %s\n' "${CFG}"
  printf '[global]\nlogSyncInterval -4\nnetwork_transport L2\n' > "${CFG}"
fi

printf '\n== ptp4l: capturing %ss of live output ==\n' "${SECS}"
run timeout --foreground "${SECS}" ptp4l -f "${CFG}" -i "${IFACE}" -m -l 7 -s -p "${DEV}"

printf '\n== phc_ctl: reading the PHC clock ==\n'
run phc_ctl "${DEV}" get

printf '\ndone.\n'