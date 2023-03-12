
# ==============================================================================
# Makefile helper functions for docker
#

DOCKER := docker
DOCKER_SUPPORTED_API_VERSION ?= 1.31

DOCKER_NETWORK ?= --network=host
DOCKER_ARGS ?= --build-arg VERSION=$(DOCKER_IMAGE_VERSION) --build-arg COMMIT=$(COMMIT)



.PHONY: docker.build.%
docker.build.%:
	@echo "===========> Docker Image Building ${HUB}/$*:$(DOCKER_IMAGE_VERSION)"
	@$(DOCKER) build -t ${HUB}/$*:$(DOCKER_IMAGE_VERSION) ${DOCKER_ARGS} ${DOCKER_NETWORK} -f build/dockerfile/$*/Dockerfile .


