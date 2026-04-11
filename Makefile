APP := reumask
STATIC_APP := $(APP)-static
BUILD_DIR := $(CURDIR)/build
BIN_DIR := $(BUILD_DIR)/bin
HOST_BIN_DIR := $(BIN_DIR)/host
RELEASE_BIN_DIR := $(BIN_DIR)/release
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
	$(RELEASE_BIN_DIR)/$(STATIC_APP)-macos-aarch64 \
	$(RELEASE_BIN_DIR)/$(STATIC_APP)-linux-aarch64 \
	$(RELEASE_BIN_DIR)/$(STATIC_APP)-linux-x86_64

export GOCACHE
export GOMODCACHE
export GOPATH
export GOTMPDIR
export GOTELEMETRYDIR
export GOENV
export GOFLAGS
export GOTELEMETRY=off

.PHONY: all build static test release install run clean

all: build

build: $(BIN_DIR)/$(APP)

static: $(BIN_DIR)/$(STATIC_APP)

release: $(RELEASE_BINS)

install: $(HOST_BIN_DIR)/$(APP)
	if [ "$$(id -u)" -eq 0 ]; then \
		install -d "/usr/local/bin"; \
		install -m 0755 "$(HOST_BIN_DIR)/$(APP)" "/usr/local/bin/$(APP)"; \
	else \
		install -d "$$HOME/.local/bin"; \
		ln -sfn "$(HOST_BIN_DIR)/$(APP)" "$$HOME/.local/bin/$(APP)"; \
	fi

test: go.mod main.go main_test.go
	mkdir -p "$(BIN_DIR)" "$(GOCACHE)" "$(GOMODCACHE)" "$(GOPATH)" "$(GOTMPDIR)" "$(GOTELEMETRYDIR)"
	go test ./...

$(BIN_DIR)/$(APP): $(HOST_BIN_DIR)/$(APP)

$(BIN_DIR)/$(STATIC_APP): $(HOST_BIN_DIR)/$(STATIC_APP)

$(HOST_BIN_DIR)/$(APP): go.mod main.go
	mkdir -p "$(HOST_BIN_DIR)" "$(GOCACHE)" "$(GOMODCACHE)" "$(GOPATH)" "$(GOTMPDIR)" "$(GOTELEMETRYDIR)"
	go build -o "$@" .

$(HOST_BIN_DIR)/$(STATIC_APP): go.mod main.go
	mkdir -p "$(HOST_BIN_DIR)" "$(GOCACHE)" "$(GOMODCACHE)" "$(GOPATH)" "$(GOTMPDIR)" "$(GOTELEMETRYDIR)"
	CGO_ENABLED=0 GOOS="$(GOOS)" GOARCH="$(GOARCH)" go build -trimpath -ldflags='-s -w' -o "$@" .

$(RELEASE_BIN_DIR)/$(STATIC_APP)-macos-aarch64: go.mod main.go
	mkdir -p "$(RELEASE_BIN_DIR)" "$(GOCACHE)" "$(GOMODCACHE)" "$(GOPATH)" "$(GOTMPDIR)" "$(GOTELEMETRYDIR)"
	CGO_ENABLED=0 GOOS="darwin" GOARCH="arm64" go build -trimpath -ldflags='-s -w' -o "$@" .

$(RELEASE_BIN_DIR)/$(STATIC_APP)-linux-aarch64: go.mod main.go
	mkdir -p "$(RELEASE_BIN_DIR)" "$(GOCACHE)" "$(GOMODCACHE)" "$(GOPATH)" "$(GOTMPDIR)" "$(GOTELEMETRYDIR)"
	CGO_ENABLED=0 GOOS="linux" GOARCH="arm64" go build -trimpath -ldflags='-s -w' -o "$@" .

$(RELEASE_BIN_DIR)/$(STATIC_APP)-linux-x86_64: go.mod main.go
	mkdir -p "$(RELEASE_BIN_DIR)" "$(GOCACHE)" "$(GOMODCACHE)" "$(GOPATH)" "$(GOTMPDIR)" "$(GOTELEMETRYDIR)"
	CGO_ENABLED=0 GOOS="linux" GOARCH="amd64" go build -trimpath -ldflags='-s -w' -o "$@" .

run: build
	"$(HOST_BIN_DIR)/$(APP)"

clean:
	chmod -R u+w "$(BUILD_DIR)" 2>/dev/null || true
	rm -rf "$(BUILD_DIR)"
