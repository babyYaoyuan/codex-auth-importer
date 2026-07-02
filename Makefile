PLUGIN_ID ?= codex-auth-importer
VERSION ?= $(shell tr -d '[:space:]' < VERSION)
GO ?= go
GOOS ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)
HOST_GOOS ?= $(shell $(GO) env GOHOSTOS)
HOST_GOARCH ?= $(shell $(GO) env GOHOSTARCH)
CC ?= $(shell $(GO) env CC)
MANAGEMENT_KEY ?=
LD_FLAGS := -s -w -X main.version=$(VERSION)
ifneq ($(strip $(MANAGEMENT_KEY)),)
LD_FLAGS := $(LD_FLAGS) -X main.managementKey=$(MANAGEMENT_KEY)
endif

ifeq ($(GOOS),darwin)
	LIB_EXT := dylib
else ifeq ($(GOOS),windows)
	LIB_EXT := dll
else
	LIB_EXT := so
endif

DIST_DIR ?= dist
BUILD_DIR ?= $(DIST_DIR)
PLUGIN_BIN ?= $(BUILD_DIR)/$(PLUGIN_ID).$(LIB_EXT)
RELEASE_NAME := $(PLUGIN_ID)_$(VERSION)_$(GOOS)_$(GOARCH)
RELEASE_STAGING := $(DIST_DIR)/release/$(RELEASE_NAME)
RELEASE_ZIP := $(DIST_DIR)/release/$(RELEASE_NAME).zip
RELEASE_SHA256 := $(RELEASE_ZIP).sha256
CHECKSUMS_PATH := $(DIST_DIR)/release/checksums.txt

.PHONY: all test vet build package release checksums clean

all: test build

test:
	CGO_ENABLED=1 $(GO) test ./...

vet:
	$(GO) vet ./...

build:
	mkdir -p $(dir $(PLUGIN_BIN))
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) CC="$(CC)" $(GO) build -trimpath -buildmode=c-shared -ldflags "$(LD_FLAGS)" -o $(PLUGIN_BIN) .
	rm -f $(basename $(PLUGIN_BIN)).h

package: build
	CGO_ENABLED=0 GOOS=$(HOST_GOOS) GOARCH=$(HOST_GOARCH) $(GO) run ./.github/scripts/package-release.go -library "$(PLUGIN_BIN)" -archive "$(RELEASE_ZIP)" -checksum "$(RELEASE_SHA256)"
	@echo $(RELEASE_ZIP)

release: clean test vet package

checksums: package
	cat $(DIST_DIR)/release/*.zip.sha256 | sort -k 2 > "$(CHECKSUMS_PATH)"
	@echo $(CHECKSUMS_PATH)

clean:
	rm -rf $(DIST_DIR)
