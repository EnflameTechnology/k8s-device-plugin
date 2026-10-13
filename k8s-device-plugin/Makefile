# Copyright (c) 2020-2022, NVIDIA CORPORATION.  All rights reserved.
# Copyright (c) 2024, ENFLAME INC.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

WITH_ISGCU   ?= yes

ifeq ($(WITH_ISGCU), yes)
##### Read Release Version #####
VERSION_TXT ?= $(shell if [ -f "../../../g_version.txt" ]; then echo "../../../g_version.txt"; fi)
RELEASE_VERSION ?= $(shell if [ -n "$(VERSION_TXT)" ]; then grep -w 'ENFLAME_DEVICE_PLUGIN_VERSION' $(VERSION_TXT) | awk -F '=' '{print $$2}'; fi)
VERSION = $(RELEASE_VERSION)
ifeq ($(VERSION),)
    VERSION := 0.0.0
endif

# all: binaries  # compile all components
all: cmd-enflame-device-plugin cmd-config-manager  # compile enflame-device-plugin and config-manager

ENFLAME_DEVICE_PLUGIN        := enflame-device-plugin
CONFIG_MANAGER               := config-manager
GCU_FEATURE_DISCOVERY        := gcu-feature-discovery
MPS_CONTROL_DAEMON           := mps-control-daemon

clean:
	$(info Cleaning up build artifacts...)
	@if [ -f ./${ENFLAME_DEVICE_PLUGIN} ]; then $(RM) ${ENFLAME_DEVICE_PLUGIN}; fi
	@if [ -f ./${CONFIG_MANAGER} ]; then $(RM) ${CONFIG_MANAGER}; fi
	@if [ -f ./${GCU_FEATURE_DISCOVERY} ]; then $(RM) ${GCU_FEATURE_DISCOVERY}; fi
	@if [ -f ./${MPS_CONTROL_DAEMON} ]; then $(RM) ${MPS_CONTROL_DAEMON}; fi

gomod:
	go list -mod=readonly -m -json all

endif

DOCKER   ?= docker
MKDIR    ?= mkdir
DIST_DIR ?= $(CURDIR)/dist

include $(CURDIR)/versions.mk

ifeq ($(IMAGE_NAME),)
REGISTRY ?= enflame
IMAGE_NAME = $(REGISTRY)/k8s-device-plugin
endif

EXAMPLES := $(patsubst ./examples/%/,%,$(sort $(dir $(wildcard ./examples/*/))))
EXAMPLE_TARGETS := $(patsubst %,example-%, $(EXAMPLES))

CMDS := $(patsubst ./cmd/%/,%,$(sort $(dir $(wildcard ./cmd/*/))))
CMD_TARGETS := $(patsubst %,cmd-%, $(CMDS))

CHECK_TARGETS := lint
MAKE_TARGETS := binaries build check fmt lint-internal test examples cmds coverage generate vendor check-vendor $(CHECK_TARGETS)

TARGETS := $(MAKE_TARGETS) $(EXAMPLE_TARGETS) $(CMD_TARGETS)

DOCKER_TARGETS := $(patsubst %,docker-%, $(TARGETS))
.PHONY: $(TARGETS) $(DOCKER_TARGETS)

ifeq ($(VERSION),)
CLI_VERSION = $(LIB_VERSION)$(if $(LIB_TAG),-$(LIB_TAG))
else
CLI_VERSION = $(VERSION)
endif
CLI_VERSION_PACKAGE = enflame/k8s-device-plugin/internal/info

binaries: cmds
ifneq ($(PREFIX),)
cmd-%: COMMAND_BUILD_OPTIONS = -o $(PREFIX)/$(*)
endif
ifneq ($(shell uname),Darwin)
EXTLDFLAGS = -Wl,--export-dynamic -Wl,--unresolved-symbols=ignore-in-object-files
else
EXTLDFLAGS = -Wl,-undefined,dynamic_lookup
endif
BUILDFLAGS = -ldflags "-s -w '-extldflags=$(EXTLDFLAGS)' -X $(CLI_VERSION_PACKAGE).gitCommit=$(GIT_COMMIT) -X $(CLI_VERSION_PACKAGE).version=$(CLI_VERSION)"
build:
	go build $(BUILDFLAGS) ./...

cmds: $(CMD_TARGETS)
$(CMD_TARGETS): cmd-%:
	go build $(BUILDFLAGS) $(COMMAND_BUILD_OPTIONS) $(MODULE)/cmd/$(*)

examples: $(EXAMPLE_TARGETS)
$(EXAMPLE_TARGETS): example-%:
	go build $(BUILDFLAGS) $(COMMAND_BUILD_OPTIONS) ./examples/$(*)

ifeq ($(WITH_ISGCU), no)
all: check test build binaries
endif
check: $(CHECK_TARGETS)

# Apply go fmt to the codebase
fmt:
	go list -f '{{.Dir}}' $(MODULE)/... \
		| xargs gofmt -s -l -w

goimports:
	go list -f {{.Dir}} $(MODULE)/... \
		| xargs goimports -local $(MODULE) -w

lint:
	golangci-lint run ./...

vendor:
	go mod tidy
	go mod vendor
	go mod verify

check-vendor: vendor
	git diff --quiet HEAD -- go.mod go.sum vendor

COVERAGE_FILE := coverage.out
test: build cmds
	go test -coverprofile=$(COVERAGE_FILE) $(MODULE)/cmd/... $(MODULE)/internal/... $(MODULE)/api/...

coverage: test
	cat $(COVERAGE_FILE) | grep -v "_mock.go" > $(COVERAGE_FILE).no-mocks
	go tool cover -func=$(COVERAGE_FILE).no-mocks

generate:
	go generate $(MODULE)/...

$(DOCKER_TARGETS): docker-%:
	@echo "Running 'make $(*)' in container image $(BUILDIMAGE)"
	$(DOCKER) run \
		--rm \
		-e GOCACHE=/tmp/.cache/go \
		-e GOMODCACHE=/tmp/.cache/gomod \
		-v $(PWD):/work \
		-w /work \
		--user $$(id -u):$$(id -g) \
		$(BUILDIMAGE) \
			make $(*)

# Start an interactive shell using the development image.
PHONY: .shell
.shell:
	$(DOCKER) run \
		--rm \
		-ti \
		-e GOCACHE=/tmp/.cache/go \
		-e GOMODCACHE=/tmp/.cache/gomod \
		-v $(PWD):/work \
		-w /work \
		--user $$(id -u):$$(id -g) \
		$(BUILDIMAGE)
