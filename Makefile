PLUGIN_ID := codex-auth-importer
VERSION := $(shell tr -d '[:space:]' < VERSION)
GO ?= go
GOOS_VALUE := $(shell $(GO) env GOOS)
GOARCH_VALUE := $(shell $(GO) env GOARCH)

ifeq ($(GOOS_VALUE),darwin)
	LIB_EXT := dylib
else ifeq ($(GOOS_VALUE),windows)
	LIB_EXT := dll
else
	LIB_EXT := so
endif

DIST_DIR := dist
PLUGIN_BIN := $(DIST_DIR)/$(PLUGIN_ID).$(LIB_EXT)
RELEASE_NAME := $(PLUGIN_ID)_$(VERSION)_$(GOOS_VALUE)_$(GOARCH_VALUE)
RELEASE_STAGING := $(DIST_DIR)/release/$(RELEASE_NAME)
RELEASE_ZIP := $(DIST_DIR)/release/$(RELEASE_NAME).zip

.PHONY: all test build release clean

all: test build

test:
	CGO_ENABLED=1 $(GO) test ./...

build:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=1 $(GO) build -trimpath -buildmode=c-shared -ldflags "-s -w -X main.version=$(VERSION)" -o $(PLUGIN_BIN) .
	rm -f $(DIST_DIR)/$(PLUGIN_ID).h

release: clean test build
	mkdir -p $(RELEASE_STAGING)
	cp $(PLUGIN_BIN) $(RELEASE_STAGING)/
	cp README.md CHANGELOG.md VERSION $(RELEASE_STAGING)/
	cd $(RELEASE_STAGING) && shasum -a 256 * > checksums.txt
	cd $(RELEASE_STAGING) && zip -q -r ../$(RELEASE_NAME).zip .
	@echo $(RELEASE_ZIP)

clean:
	rm -rf $(DIST_DIR)
