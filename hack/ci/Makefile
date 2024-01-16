PROJECT_ROOT ?= ../..
KYMA_CLI ?= "${PROJECT_ROOT}/${KYMA}"
CLUSTER_NAME ?= kyma
REGISTRY_PORT ?= 5001

ifndef MODULE_VERSION
	include ${PROJECT_ROOT}/.version
endif

ifndef MODULE_SHA
    MODULE_SHA = ${shell git rev-parse --short HEAD}
endif

.PHONY: module-build
module-build: ## Build the Module, push it to a registry and print it based on the commit hash.
	@make -C ${PROJECT_ROOT} module-build \
		IMG=${IMG} \
		MODULE_REGISTRY=${MODULE_REGISTRY} \
		MODULE_VERSION=${MODULE_VERSION}-${MODULE_SHA}
	cp -f ${PROJECT_ROOT}/module-template.yaml ${ARTIFACTS}/module-template.yaml
	@echo "Added template to job artifacts: ${ARTIFACTS}/module-template.yaml"
	@echo "\n~~~~~~~~~~~~BEGINNING OF MODULE TEMPLATE~~~~~~~~~~~~~~"
	@cat ${ARTIFACTS}/module-template.yaml
	@echo "\n~~~~~~~~~~~~~~~END OF MODULE TEMPLATE~~~~~~~~~~~~~~~~"

.PHONY: download-module-template
download-module-template: ## Downloads the module-template from the module-build prow job.
	export PROJECT_ROOT="${PROJECT_ROOT}" \
	&& ../get_module_template_from_build_job.sh

.PHONY: install-latest-module-template-fast
install-latest-module-template-fast: ## Downloads and applies the latest released module-template from fast channel.
	kubectl apply -f https://github.com/kyma-project/eventing-manager/releases/latest/download/module-template.yaml

.PHONY: create-kyma-system-ns
create-kyma-system-ns: ## Create kyma-system namespace.
	kubectl create ns kyma-system

.PHONY: install-module-template
install-module-template: ## Apply Eventing moduletemplate.
	kubectl apply -f ${PROJECT_ROOT}/module-template.yaml

.PHONY: enable-module-without-default-cr
enable-module-without-default-cr: ## Enable Eventing module without creating default Eventing CR.
	"${KYMA_CLI}" alpha enable module eventing -c fast -n kyma-system -p Ignore

.PHONY: install-lifecycle-manager
install-lifecycle-manager: ## Deploys lifecycle-manager.
	"${KYMA_CLI}" alpha deploy \
		--ci \
		--force-conflicts

.PHONY: create-k3d
create-k3d: ## Create k3d with kyma CRDs.
	"${KYMA_CLI}" provision k3d -p 8081:80@loadbalancer -p 8443:443@loadbalancer --registry-port ${REGISTRY_PORT} --name ${CLUSTER_NAME} --ci

.PHONY: install-lifecycle-manager
install-lifecycle-manager: ## Deploys lifecycle-manager.
	"${KYMA_CLI}" alpha deploy \
		--ci \
		--force-conflicts

.PHONY: install-latest-nats-module-template-fast
install-latest-nats-module-template-fast: ## Downloads and applies the latest nats-manager released module-template from fast channel.
	kubectl apply -f https://github.com/kyma-project/nats-manager/releases/latest/download/module-template.yaml

.PHONY: enable-nats-module
enable-nats-module:
	"${KYMA_CLI}" alpha enable module nats -c fast -f --ci -n kyma-system

.PHONY: get-nats-via-lifecycle-manager
get-nats-via-lifecycle-manager: install-lifecycle-manager install-latest-nats-module-template-fast enable-nats-module
	./check-k8s-resource-is-ready.sh 180 nats kyma-system

.PHONY: install-nats-manager
install-nats-manager:
	kubectl apply -f https://github.com/kyma-project/nats-manager/releases/latest/download/nats-manager.yaml
	kubectl apply -f https://github.com/kyma-project/nats-manager/releases/latest/download/nats-default-cr.yaml

.PHONY: verify-kyma
verify-kyma: ## Wait for Kyma CR to be in Ready state.
	../verify_kyma_status.sh

.PHONY: install-k3d-tools
install-k3d-tools: ## Create k3d with kyma CRDs.
	curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | TAG=${K3D_VERSION} bash

.PHONY: apply-peerauthentication-crd
apply-peerauthentication-crd:
	kubectl apply -f ../../config/crd/for-tests/security.istio.io_peerauthentication.yaml
