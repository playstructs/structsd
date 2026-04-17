#!/usr/bin/make -f

BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')

ifeq (,$(VERSION))
  VERSION := $(shell git describe --exact-match 2>/dev/null)
  ifeq (,$(VERSION))
    VERSION := $(BRANCH)-$(COMMIT)
  endif
endif

LEDGER_ENABLED ?= true
TM_VERSION := $(shell go list -m github.com/cometbft/cometbft | sed 's:.* ::')
BUILDDIR ?= $(CURDIR)/build

GO_SYSTEM_VERSION = $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1-2)
REQUIRE_GO_VERSION = 1.23

export GO111MODULE = on

###############################################################################
###                               Build Tags                                ###
###############################################################################

build_tags = netgo
ifeq ($(LEDGER_ENABLED),true)
  ifeq ($(OS),Windows_NT)
    GCCEXE = $(shell where gcc.exe 2> NUL)
    ifeq ($(GCCEXE),)
      $(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
    else
      build_tags += ledger
    endif
  else
    UNAME_S = $(shell uname -s)
    ifeq ($(UNAME_S),OpenBSD)
      $(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
    else
      GCC = $(shell command -v gcc 2> /dev/null)
      ifeq ($(GCC),)
        $(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
      else
        build_tags += ledger
      endif
    endif
  endif
endif

build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

whitespace :=
whitespace := $(whitespace) $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

###############################################################################
###                              Linker Flags                               ###
###############################################################################

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=structs \
		  -X github.com/cosmos/cosmos-sdk/version.AppName=structsd \
		  -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
		  -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
		  -X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)" \
		  -X github.com/cometbft/cometbft/version.TMCoreSemVer=$(TM_VERSION)

ifeq ($(LINK_STATICALLY),true)
  ldflags += -linkmode=external -extldflags "-Wl,-z,muldefs -static"
endif
ldflags += -w -s
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)' -trimpath

###############################################################################
###                                  Help                                   ###
###############################################################################

.DEFAULT_GOAL := help
help:
	@echo "Structs Chain Makefile"
	@echo ""
	@echo "Usage:  make [target]"
	@echo ""
	@echo "Build:"
	@echo "  build                Build structsd for current platform"
	@echo "  install              Install structsd to GOPATH/bin"
	@echo "  build-all            Build for all supported platforms"
	@echo "  build-linux-amd64    Cross-compile for linux/amd64"
	@echo "  build-linux-arm64    Cross-compile for linux/arm64"
	@echo "  build-darwin-amd64   Cross-compile for darwin/amd64 (Intel Mac)"
	@echo "  build-darwin-arm64   Cross-compile for darwin/arm64 (Apple Silicon)"
	@echo "  build-windows-amd64  Cross-compile for windows/amd64"
	@echo "  clean                Remove build artifacts"
	@echo ""
	@echo "Proto:"
	@echo "  proto-all            Format, lint, and generate all proto outputs"
	@echo "  proto-gen            Generate Go protobuf files (gogo + pulsar)"
	@echo "  proto-gen-ts         Generate TypeScript proto bindings"
	@echo "  proto-swagger        Generate OpenAPI/Swagger spec"
	@echo "  proto-format         Format .proto files"
	@echo "  proto-lint           Lint .proto files"
	@echo ""
	@echo "Test:"
	@echo "  test                 Run all tests"
	@echo "  test-unit            Run unit tests with timeout"
	@echo "  test-race            Run tests with race detector"
	@echo "  test-cover           Run tests with coverage report"
	@echo "  test-integration     Run integration test script"
	@echo ""
	@echo "Lint:"
	@echo "  lint                 Run golangci-lint"
	@echo "  lint-fix             Run golangci-lint with auto-fix"
	@echo "  format               Format Go source files"
	@echo ""
	@echo "Dev:"
	@echo "  serve                Start chain via Ignite CLI"
	@echo "  serve-reset          Start chain via Ignite CLI (reset state)"
	@echo "  serve-reset-verbose  Start chain via Ignite CLI (reset + verbose)"
	@echo ""
	@echo "Release:"
	@echo "  release-dry-run      Test release locally (no publish)"
	@echo "  release              Create a release (requires GITHUB_TOKEN)"
	@echo ""
	@echo "Misc:"
	@echo "  go.sum               Verify and tidy Go dependencies"

###############################################################################
###                                 Build                                   ###
###############################################################################

check_version:
ifneq ($(shell [ "$(GO_SYSTEM_VERSION)" \< "$(REQUIRE_GO_VERSION)" ] && echo true),)
	@echo "ERROR: Go version $(REQUIRE_GO_VERSION)+ is required (found $(GO_SYSTEM_VERSION))"
	@exit 1
endif

all: build lint test

build: check_version go.sum
	@mkdir -p $(BUILDDIR)
	go build -mod=readonly $(BUILD_FLAGS) -o $(BUILDDIR)/structsd ./cmd/structsd
	@echo "Built: $(BUILDDIR)/structsd"

install: check_version go.sum
	go install -mod=readonly $(BUILD_FLAGS) ./cmd/structsd
	@echo "Installed: structsd"

clean:
	rm -rf $(BUILDDIR)/

###############################################################################
###                          Cross-Compilation                              ###
###############################################################################

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

build-all: $(foreach p,$(PLATFORMS),build-$(subst /,-,$(p)))
	@echo "All platform builds complete. Artifacts in $(BUILDDIR)/"

define BUILD_PLATFORM
build-$(subst /,-,$(1)): check_version go.sum
	@echo "Building structsd for $(1)..."
	@mkdir -p $(BUILDDIR)
	CGO_ENABLED=0 GOOS=$(word 1,$(subst /, ,$(1))) GOARCH=$(word 2,$(subst /, ,$(1))) \
		go build -mod=readonly $(BUILD_FLAGS) \
		-o $(BUILDDIR)/structsd-$(subst /,-,$(1))$(if $(findstring windows,$(1)),.exe) \
		./cmd/structsd
	@echo "Built: $(BUILDDIR)/structsd-$(subst /,-,$(1))$(if $(findstring windows,$(1)),.exe)"
endef
$(foreach p,$(PLATFORMS),$(eval $(call BUILD_PLATFORM,$(p))))

###############################################################################
###                             Dependencies                                ###
###############################################################################

go.sum: go.mod
	@echo "--> Verifying dependencies"
	go mod verify
	go mod tidy
	@echo "--> Downloading dependencies"
	go mod download

###############################################################################
###                               Protobuf                                  ###
###############################################################################

proto-all: proto-format proto-lint proto-gen proto-gen-ts
	@echo "Proto generation complete."

proto-gen:
	@echo "--> Generating Go protobuf files (gogo)..."
	buf generate --template proto/buf.gen.gogo.yaml
	@echo "--> Generating Go protobuf files (pulsar)..."
	buf generate --template proto/buf.gen.pulsar.yaml

proto-gen-ts:
	@echo "--> Generating TypeScript proto bindings..."
	buf generate --template proto/buf.gen.ts.yaml

proto-swagger:
	@echo "--> Generating OpenAPI/Swagger spec..."
	buf generate --template proto/buf.gen.swagger.yaml

proto-format:
	@echo "--> Formatting proto files..."
	buf format -w

proto-lint:
	@echo "--> Linting proto files..."
	buf lint

.PHONY: proto-all proto-gen proto-gen-ts proto-swagger proto-format proto-lint

###############################################################################
###                                 Tests                                   ###
###############################################################################

TEST_PACKAGES := ./...

test:
	@echo "--> Running all tests"
	go test -mod=readonly $(TEST_PACKAGES)

test-unit:
	@echo "--> Running unit tests"
	go test -mod=readonly -timeout=5m -tags='norace' $(TEST_PACKAGES)

test-race:
	@echo "--> Running tests with race detector"
	go test -mod=readonly -timeout=5m -race $(TEST_PACKAGES)

test-cover:
	@echo "--> Running tests with coverage"
	go test -mod=readonly -timeout=5m -tags='norace' \
		-coverprofile=coverage.txt -covermode=atomic $(TEST_PACKAGES)
	@echo "Coverage report: coverage.txt"

test-integration:
	@echo "--> Running integration tests"
	bash tests/test_chain.sh

.PHONY: test test-unit test-race test-cover test-integration

###############################################################################
###                            Linting & Format                             ###
###############################################################################

golangci_version = v2.1.6

lint:
	@echo "--> Running linter"
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(golangci_version)
	golangci-lint run --timeout=10m

lint-fix:
	@echo "--> Running linter with auto-fix"
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(golangci_version)
	golangci-lint run --fix --issues-exit-code=0

format:
	@echo "--> Formatting Go source files"
	@go install mvdan.cc/gofumpt@latest
	find . -name '*.go' -type f \
		-not -path "./vendor/*" \
		-not -path "./.git/*" \
		-not -name "*.pb.go" \
		-not -name "*.pb.gw.go" \
		-not -name "*.pulsar.go" \
		| xargs gofumpt -w -l

.PHONY: lint lint-fix format

###############################################################################
###                          Local Development                              ###
###############################################################################

serve:
	ignite chain serve

serve-reset:
	ignite chain serve --reset-once

serve-reset-verbose:
	ignite chain serve --reset-once --verbose

.PHONY: serve serve-reset serve-reset-verbose

###############################################################################
###                                Release                                  ###
###############################################################################

release-dry-run:
	@echo "--> Running release dry run..."
	TM_VERSION=$(TM_VERSION) goreleaser release --snapshot --clean
	@echo "Dry run complete. Artifacts in dist/"

ifdef GITHUB_TOKEN
release:
	@echo "--> Creating release..."
	TM_VERSION=$(TM_VERSION) goreleaser release --clean
else
release:
	@echo "Error: GITHUB_TOKEN is not set."
	@echo "Usage: GITHUB_TOKEN=<token> make release"
	@echo ""
	@echo "Or push a tag to trigger the GitHub Actions release workflow:"
	@echo "  git tag v1.0.0 && git push origin v1.0.0"
	@exit 1
endif

.PHONY: release release-dry-run

###############################################################################
###                              Phony Targets                              ###
###############################################################################

.PHONY: all build install clean check_version help go.sum build-all
