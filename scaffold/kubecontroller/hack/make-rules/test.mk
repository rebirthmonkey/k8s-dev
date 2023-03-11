
# ==============================================================================
# Makefile helper functions for test
#


.PHONY: test.k8s.%
test.k8s.%:
	@echo "===========> Testing $*"
	@-$(KUBECTL) -n $(NAMESPACE) delete -f manifests/$*/cr.yaml
	@$(KUBECTL) -n $(NAMESPACE) apply -f manifests/$*/cr.yaml


