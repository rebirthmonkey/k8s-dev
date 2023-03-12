
# ==============================================================================
# Makefile helper functions for docker
#

KUBECTL := kubectl
NAMESPACE ?= default


.PHONY: k8s.install.%
k8s.install.%: ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	@$(KUSTOMIZE) build configs/$*/crd | kubectl apply -f -


.PHONY: k8s.uninstall.%
k8s.uninstall.%:
	@$(KUSTOMIZE) build configs/$*/crd | kubectl delete --ignore-not-found=true -f -


.PHONY: k8s.deploy.%
k8s.deploy.%: kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd configs/$*/manager && $(KUSTOMIZE) edit set image controller=${HUB}/$*:$(DOCKER_IMAGE_VERSION)
	$(KUSTOMIZE) build configs/$*/default | kubectl apply -f -

.PHONY: k8s.undeploy.%
k8s.undeploy.%: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build configs/$*/default | kubectl delete --ignore-not-found=true -f -



