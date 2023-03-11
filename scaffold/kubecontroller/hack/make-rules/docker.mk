
# ==============================================================================
# Makefile helper functions for docker
#

DOCKER := docker
DOCKER_SUPPORTED_API_VERSION ?= 1.31
DOCKER_IMAGE_VERSION := v1.0
#HUB_IMAGE ?= ${HUB}/hub:$(VERSION)

DOCKER_NETWORK ?= --network=host
DOCKER_ARGS ?= --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT)



.PHONY: docker.build.%
docker.build.%:
	@echo "===========> Docker Image Building ${HUB}/$*:$(DOCKER_IMAGE_VERSION)"
	@$(DOCKER) build -t ${HUB}/$*:$(DOCKER_IMAGE_VERSION) ${DOCKER_ARGS} ${DOCKER_NETWORK} -f build/dockerfile/$*/Dockerfile .


