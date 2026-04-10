APP := reumask
STATIC_APP := $(APP)-static
BUILD_DIR := $(CURDIR)/build
BIN_DIR := $(BUILD_DIR)/bin
GOCACHE := $(BUILD_DIR)/gocache
GOMODCACHE := $(BUILD_DIR)/gomodcache
GOPATH := $(BUILD_DIR)/gopath
GOTMPDIR := $(BUILD_DIR)/tmp
GOTELEMETRYDIR := $(BUILD_DIR)/telemetry
GOENV := off
GOFLAGS := -modcacherw
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
RELEASE_BINS := \
	$(BIN_DIR)/$(STATIC_APP)-macos-aarch64 \
	$(BIN_DIR)/$(STATIC_APP)-linux-aarch64 \
	$(BIN_DIR)/$(STATIC_APP)-linux-x86_64

export GOCACHE
export GOMODCACHE
export GOPATH
export GOTMPDIR
export GOTELEMETRYDIR
export GOENV
export GOFLAGS
export GOTELEMETRY=off

.PHONY: all build static test release run clean

all: build

build: $(BIN_DIR)/$(APP)

static: $(BIN_DIR)/$(STATIC_APP)

release: $(RELEASE_BINS)

test: go.mod main.go main_test.go
	mkdir -p "$(BIN_DIR)" "$(GOCACHE)" "$(GOMODCACHE)" "$(GOPATH)" "$(GOTMPDIR)" "$(GOTELEMETRYDIR)"
	go test ./...

$(BIN_DIR)/$(APP): go.mod main.go
	mkdir -p "$(BIN_DIR)" "$(GOCACHE)" "$(GOMODCACHE)" "$(GOPATH)" "$(GOTMPDIR)" "$(GOTELEMETRYDIR)"
	go build -o "$(BIN_DIR)/$(APP)" .

$(BIN_DIR)/$(STATIC_APP): go.mod main.go
	mkdir -p "$(BIN_DIR)" "$(GOCACHE)" "$(GOMODCACHE)" "$(GOPATH)" "$(GOTMPDIR)" "$(GOTELEMETRYDIR)"
	CGO_ENABLED=0 GOOS="$(GOOS)" GOARCH="$(GOARCH)" go build -trimpath -ldflags='-s -w' -o "$(BIN_DIR)/$(STATIC_APP)" .

$(BIN_DIR)/$(STATIC_APP)-macos-aarch64: go.mod main.go
	mkdir -p "$(BIN_DIR)" "$(GOCACHE)" "$(GOMODCACHE)" "$(GOPATH)" "$(GOTMPDIR)" "$(GOTELEMETRYDIR)"
	CGO_ENABLED=0 GOOS="darwin" GOARCH="arm64" go build -trimpath -ldflags='-s -w' -o "$@" .

$(BIN_DIR)/$(STATIC_APP)-linux-aarch64: go.mod main.go
	mkdir -p "$(BIN_DIR)" "$(GOCACHE)" "$(GOMODCACHE)" "$(GOPATH)" "$(GOTMPDIR)" "$(GOTELEMETRYDIR)"
	CGO_ENABLED=0 GOOS="linux" GOARCH="arm64" go build -trimpath -ldflags='-s -w' -o "$@" .

$(BIN_DIR)/$(STATIC_APP)-linux-x86_64: go.mod main.go
	mkdir -p "$(BIN_DIR)" "$(GOCACHE)" "$(GOMODCACHE)" "$(GOPATH)" "$(GOTMPDIR)" "$(GOTELEMETRYDIR)"
	CGO_ENABLED=0 GOOS="linux" GOARCH="amd64" go build -trimpath -ldflags='-s -w' -o "$@" .

run: build
	"$(BIN_DIR)/$(APP)"

clean:
	chmod -R u+w "$(BUILD_DIR)" 2>/dev/null || true
	rm -rf "$(BUILD_DIR)"
