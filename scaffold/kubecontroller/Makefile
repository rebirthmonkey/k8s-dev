# ==============================================================================
# Build options

ROOT_PACKAGE=~/workspace/k8s-dev/scaffold/kubecontroller
VERSION_PACKAGE=github.com/rebirthmonkey/go/pkg/version
HUB ?= hub.docker.com/rebirthmonkey
DOCKER_IMAGE_VERSION := v1.1
IMG ?= hub.docker.com/rebirthmonkey/hub:v1.1


# ==============================================================================
# Includes

include hack/make-rules/common.mk
include hack/make-rules/tools.mk
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

.PHONY: install.%
install.%:
	@$(MAKE) k8s.install.$*

.PHONY: uninstall.%
uninstall.%:
	@$(MAKE) k8s.uninstall.$*


.PHONY: deploy.%
deploy.%:
	@$(MAKE) k8s.deploy.$*

.PHONY: undeploy.%
undeploy.%:
	@$(MAKE) k8s.undeploy.$*

.PHONY: clean
clean:
	@echo "===========> Cleaning all build output"
	@-rm -vrf $(OUTPUT_DIR)