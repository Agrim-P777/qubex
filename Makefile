GO := $(if $(wildcard .tools/go/bin/go),$(CURDIR)/.tools/go/bin/go,go)
export PATH := $(CURDIR)/.tools:$(CURDIR)/.tools/go/bin:$(PATH)
export GOPATH := $(CURDIR)/.cache/gopath
export GOCACHE := $(CURDIR)/.cache/go-build
VERSION ?= dev

.PHONY: build test check pty bench release
build:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags '-X main.version=$(VERSION)' -o bin/qbx ./cmd/qbx
test:
	$(GO) test ./...
check:
	$(GO) vet ./...
	$(GO) test -race ./...
pty: build
	python3 scripts/pty_smoke.py bin/qbx
bench:
	$(GO) build -o bin/qbx-bench ./tools/bench
	bin/qbx-bench -mib 5
	bin/qbx-bench -mib 50
	bin/qbx-bench -mib 500
release:
	VERSION=$(VERSION) bash scripts/release.sh
