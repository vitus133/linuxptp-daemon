# phc-sync build checklist

## Build correctness
- [x] `go build ./...` compiles without errors
- [x] `go build ./cmd/phc-sync/` produces a binary and is linked into the image
- [x] `go vet ./pkg/phcsync/ ./cmd/phc-sync/` passes

## Formatting
- [x] `gofmt -l pkg/phcsync/ cmd/phc-sync/` reports no files
- [x] `goimports -l pkg/phcsync/ cmd/phc-sync/` reports no files (or gofmt-clean)

## Docker / container build (release-5.x base image `registry.ci.openshift.org/ocp/4.18:base-rhel9`)
- [x] `.dockerignore` excludes an accidentally vendored `vendor/` directory from the build context
- [x] multi-stage Dockerfile output references the phc-sync binary, not the daemon
- [x] image builds with the added `.dockerignore` in place

## Runtime smoke test
- [ ] `phc-sync -interface <iface>` reports a non-empty status and exits cleanly
- [ ] `phc-sync -version` prints the version constant