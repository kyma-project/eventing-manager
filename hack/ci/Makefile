PROJECT_ROOT ?= ../..
KYMA_CLI ?= "${PROJECT_ROOT}/${KYMA}"
CLUSTER_NAME ?= kyma
REGISTRY_PORT ?= 5001

.PHONY: create-kyma-system-ns
create-kyma-system-ns: ## Create kyma-system namespace.
	kubectl create ns kyma-system

.PHONY: create-k3d
create-k3d: ## Create k3d with kyma CRDs.
	"${KYMA_CLI}" provision k3d -p 8081:80@loadbalancer -p 8443:443@loadbalancer --registry-port ${REGISTRY_PORT} --name ${CLUSTER_NAME} --ci

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
