PROJECT_ROOT ?= ../..
KYMA_CLI ?= "${PROJECT_ROOT}/${KYMA}"
CLUSTER_NAME ?= kyma
REGISTRY_PORT ?= 5001
K3D_VERSION ?= 1.28.4
GARDENER_KUBE_VERSION ?= 1.28.4

.PHONY: create-kyma-system-ns
create-kyma-system-ns: ## Create kyma-system namespace.
	kubectl create ns kyma-system

.PHONY: create-k3d
create-k3d: ## Create k3d with kyma CRDs.
	"${KYMA_CLI}" provision k3d -p 8081:80@loadbalancer -p 8443:443@loadbalancer --registry-port ${REGISTRY_PORT} --name ${CLUSTER_NAME} -k ${K3D_VERSION} --ci

.PHONY: install-eventing-module
install-eventing-module:
	$(eval LATEST_EVENTING_VERSION := $(shell curl -s https://api.github.com/repos/kyma-project/eventing-manager/releases/latest | jq -r '.tag_name'))
	@echo "Deploying Eventing Manager: ${LATEST_EVENTING_VERSION}"
	kubectl apply -f https://github.com/kyma-project/eventing-manager/releases/latest/download/eventing-manager.yaml
	kubectl apply -f https://github.com/kyma-project/eventing-manager/releases/latest/download/eventing-default-cr.yaml

.PHONY: install-nats-module
install-nats-module:
	$(eval LATEST_NATS_VERSION := $(shell curl -s https://api.github.com/repos/kyma-project/nats-manager/releases/latest | jq -r '.tag_name'))
	@echo "Deploying NATS Manager: ${LATEST_NATS_VERSION}"
	kubectl apply -f https://github.com/kyma-project/nats-manager/releases/latest/download/nats-manager.yaml
	kubectl apply -f https://github.com/kyma-project/nats-manager/releases/latest/download/nats-default-cr.yaml

.PHONY: install-istio-module
install-istio-module:
	$(eval LATEST_ISTIO_VERSION := $(shell curl -s https://api.github.com/repos/kyma-project/istio-manager/releases/latest | jq -r '.tag_name'))
	@echo "Deploying Istio Manager: ${LATEST_ISTIO_VERSION}"
	kubectl label namespace kyma-system istio-injection=enabled --overwrite
	kubectl apply -f https://github.com/kyma-project/istio/releases/latest/download/istio-manager.yaml
	kubectl apply -f https://github.com/kyma-project/istio/releases/latest/download/istio-default-cr.yaml

.PHONY: install-api-gateway-module
install-api-gateway-module:
	$(eval LATEST_API_GATEWAY_VERSION := $(shell curl -s https://api.github.com/repos/kyma-project/api-gateway/releases/latest | jq -r '.tag_name'))
	@echo "Deploying API Gateway module: ${LATEST_API_GATEWAY_VERSION}"
	kubectl apply -f https://github.com/kyma-project/api-gateway/releases/latest/download/api-gateway-manager.yaml
	kubectl apply -f https://github.com/kyma-project/api-gateway/releases/latest/download/apigateway-default-cr.yaml

.PHONY: verify-kyma
verify-kyma: ## Wait for Kyma CR to be in Ready state.
	../verify_kyma_status.sh

.PHONY: install-k3d-tools
install-k3d-tools: ## Create k3d with kyma CRDs.
	curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash

.PHONY: apply-peerauthentication-crd
apply-peerauthentication-crd:
	kubectl apply -f ../../config/crd/for-tests/security.istio.io_peerauthentication.yaml

.PHONY: wait-nats-cr-ready
wait-nats-cr-ready:
	kubectl wait nats.operator.kyma-project.io -n kyma-system eventing-nats --timeout=300s --for=jsonpath='{.status.state}'=Ready

.PHONY: wait-nats-cr-warning
wait-nats-cr-warning:
	kubectl wait nats.operator.kyma-project.io -n kyma-system eventing-nats --timeout=300s --for=jsonpath='{.status.state}'=Warning

.PHONY: wait-eventing-cr-ready
wait-eventing-cr-ready:
	kubectl wait eventing.operator.kyma-project.io -n kyma-system eventing --timeout=300s --for=jsonpath='{.status.state}'=Ready

.PHONY: wait-eventing-cr-ready-with-backend
wait-eventing-cr-ready-with-backend:
	kubectl wait eventing.operator.kyma-project.io -n kyma-system eventing --timeout=300s --for=jsonpath='{.status.state}'=Ready --for=jsonpath='{.status.activeBackend}'=${ACTIVE_BACKEND}

.PHONY: wait-istio-cr-ready
wait-istio-cr-ready:
	kubectl wait istios.operator.kyma-project.io -n kyma-system default --timeout=300s --for=jsonpath='{.status.state}'=Ready

.PHONY: wait-api-gateway-cr-ready
wait-api-gateway-cr-ready:
	kubectl wait apigateways.operator.kyma-project.io -n kyma-system default --timeout=300s --for=jsonpath='{.status.state}'=Ready

.PHONY: provision-gardener-cluster
provision-gardener-cluster:
	GARDENER_CLUSTER_VERSION="${GARDENER_KUBE_VERSION}" PROJECT_ROOT="${PROJECT_ROOT}" ${PROJECT_ROOT}/scripts/gardener/aws/provision.sh

.PHONY: deprovision-gardener-cluster
deprovision-gardener-cluster:
	PROJECT_ROOT="${PROJECT_ROOT}" ${PROJECT_ROOT}/scripts/gardener/aws/deprovision.sh

.PHONY: create-ias-app
create-ias-app:
	PROJECT_ROOT="${PROJECT_ROOT}" ${PROJECT_ROOT}/scripts/ias/create-ias-app.sh

.PHONY: delete-ias-app
delete-ias-app:
	PROJECT_ROOT="${PROJECT_ROOT}" ${PROJECT_ROOT}/scripts/ias/delete-ias-app.sh
