
# ==============================================================================
# Build options

ROOT_PACKAGE=~/workspace/k8s-dev/scaffold/kubecontroller
VERSION_PACKAGE=github.com/rebirthmonkey/k8s-dev/scaffold/pkg/version

# ==============================================================================
# Includes

include hack/make-rules/common.mk
include hack/make-rules/golang.mk


VERSION ?= $(shell git describe --always --tags)
COMMIT = $(shell git rev-parse HEAD)


# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

version:
	@echo $(VERSION)

## run: Run the whole application.
.PHONY: run
run:
	@$(MAKE) go.run.manager

.PHONY: run.%
run.%:
	@$(MAKE) go.run.$*


## clean: Remove all files that are created by building.
.PHONY: clean
clean:
	@echo "===========> Cleaning all build output"
	@-rm -vrf $(OUTPUT_DIR)