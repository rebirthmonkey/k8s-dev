# ==============================================================================
# Build options

ROOT_PACKAGE=~/workspace/k8s-dev/scaffold/kubecontroller
VERSION_PACKAGE=github.com/rebirthmonkey/go/pkg/version
HUB ?= hub.docker.com/rebirthmonkey


# ==============================================================================
# Includes

include hack/make-rules/common.mk
include hack/make-rules/golang.mk
include hack/make-rules/k8s.mk
include hack/make-rules/test.mk
include hack/make-rules/docker.mk


version:
	@echo $(VERSION)

.PHONY: run
run:
	@$(MAKE) go.run

.PHONY: run.%
run.%:
	@$(MAKE) go.run.$*

.PHONY: test.%
test.%:
	@$(MAKE) test.k8s.$*


.PHONY: docker-build.%
docker-build.%:
	@$(MAKE) docker.build.$*


.PHONY: clean
clean:
	@echo "===========> Cleaning all build output"
	@-rm -vrf $(OUTPUT_DIR)