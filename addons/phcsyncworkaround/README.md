# PHC Sync Workaround Plugin

The plugin adjusts the PHC used by the time-receiver ports in a telecom
boundary-clock profile before normal daemon processing continues. A time
receiver port is a PTP interface section with `masterOnly 0` in the profile's
`ptp4lConf`.

## Enable the plugin

Register the plugin using `PtpOperatorConfig.spec.plugins`. Keep every other
plugin the cluster needs because specifying this map replaces the operator's
default plugin list.

```yaml
apiVersion: ptp.openshift.io/v1
kind: PtpOperatorConfig
metadata:
  name: default
  namespace: openshift-ptp
spec:
  daemonNodeSelector:
    node-role.kubernetes.io/master: ""
  ptpEventConfig:
    apiVersion: "2.0"
    enableEventPublisher: true
  plugins:
    e810: {}
    e825: {}
    e830: {}
    ntpfailover: {}
    phc-sync-workaround: {}
```

Select it in the relevant `PtpConfig` profile. Add these entries to the
profile's existing `plugins` map:

```yaml
spec:
  profile:
    - name: 01-bc-tr
      plugins:
        phc-sync-workaround: {}
        e825:
          devices:
            - eno8703
          settings:
            LocalMaxHoldoverOffSet: 500
            LocalHoldoverTimeout: 0
            MaxInSpecOffset: 100
        e830:
          devices:
            - enp108s0f0
            - enp110s0f0
```

The existing profile must still define its `ptp4lConf`, including the relevant
TR interface sections with `masterOnly 0`.

## Plugin behavior

During profile application, the plugin finds every TR interface configured with
`masterOnly 0`. It generates a free-running `ptp4l` config from the profile's
telecom settings and runs `ptp4l` with a repeated `-i <interface>` argument for
each TR port. Profile application blocks while it collects 16 offset samples
with non-zero path delay. The plugin averages the offsets, reads the
associated PHC, and sets it to the corrected time. Only upstream ports sharing the
same PHC are supported.

You may optionally set `timeout` under `phc-sync-workaround`, using a Go
duration such as `15s` or `2m`; it limits only the `ptp4l` measurement. By
default, measurement runs without a timeout and waits until enough valid samples
are collected.
