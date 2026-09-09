# phc-sync build checklist

## Build correctness
- [x] `go build ./...` compiles without errors
- [x] `go build ./cmd/phc-sync/` produces a binary and is linked into the image
- [x] `go vet ./pkg/phcsync/ ./cmd/phc-sync/` passes

## Formatting
- [x] `gofmt -l pkg/phcsync/ cmd/phc-sync/` reports no files
- [x] `goimports -l pkg/phcsync/ cmd/phc-sync/` reports no files (or gofmt-clean)

## Docker / container build
- [x] no `.dockerignore` is added: `hack/build.sh` builds with `--mod=vendor` and runs `git rev-list -1 HEAD`, so the build context must keep the committed `vendor/` and `.git/`
- [x] multi-stage Dockerfile output references the phc-sync binary, not the daemon

## Runtime smoke test
- [ ] `phc-sync -interface <iface>` reports a non-empty status and exits cleanly
- [ ] `phc-sync -version` prints the version constant