PROJECT_ROOT ?= ../..
KYMA_CLI ?= "${PROJECT_ROOT}/${KYMA}"
CLUSTER_NAME ?= kyma
REGISTRY_PORT ?= 5001

.PHONY: install-k3d-tools
install-k3d-tools: ## Create k3d with kyma CRDs.
	curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | TAG=${K3D_VERSION} bash

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
	kyma alpha enable module nats -c fast -f --ci -n kyma-system

.PHONY: wait-until-nats-is-ready
wait-until-nats-is-ready:
