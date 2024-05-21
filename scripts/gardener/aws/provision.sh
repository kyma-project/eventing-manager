#!/usr/bin/env bash

# Usage: To run this script, set the required environment variables and run this script.
#  CLUSTER_NAME - (required)
#  GARDENER_REGION - (required)
#  GARDENER_ZONES - (required)
#  GARDENER_PROJECT_NAME - (required)
#  GARDENER_PROVIDER_SECRET_NAME - (required)
#  GARDENER_KUBECONFIG - (required, description: Path to kubeconfig for Gardener.)
#  MACHINE_TYPE - (optional, default: "m5.xlarge")
#  SCALER_MIN - (optional, default: 1)
#  SCALER_MAX - (optional, default: 2)
#  RETRY_ATTEMPTS - (optional, default: 1)
#  GARDENER_CLUSTER_VERSION - (optional, default: kyma CLI default value for kube-version.)

# Permissions: In order to run this script you need to use a service account with permissions equivalent to the following GCP roles:
# - Compute Admin
# - Service Account User
# - Service Account Admin
# - Service Account Token Creator
# - Make sure the service account is enabled for the Google Identity and Access Management API.

source "${PROJECT_ROOT}/scripts/utils/log.sh"
source "${PROJECT_ROOT}/scripts/utils/utils.sh"

gardener::validate_and_default() {
    requiredVars=(
        CLUSTER_NAME
        GARDENER_REGION
        GARDENER_ZONES
        GARDENER_KUBECONFIG
        GARDENER_PROJECT_NAME
        GARDENER_PROVIDER_SECRET_NAME
    )
    utils::check_required_vars "${requiredVars[@]}"

    # validations
    if [ "${#CLUSTER_NAME}" -gt 9 ]; then
        log::error "Provided cluster name is too long"
        return 1
    fi

    # set default values
    if [ -z "$MACHINE_TYPE" ]; then
        export MACHINE_TYPE="m5.xlarge"
    fi

    if [ -z "$SCALER_MIN" ]; then
        export SCALER_MIN="1"
    fi

    if [ -z "$SCALER_MAX" ]; then
        export SCALER_MAX="2"
    fi

    if [ -z "$RETRY_ATTEMPTS" ]; then
        export RETRY_ATTEMPTS="1"
    fi

    if [ -z "$GARDENER_CLUSTER_VERSION" ]; then
        # Detect supported kube-version.
        export GARDENER_CLUSTER_VERSION=$(kubectl --kubeconfig="${GARDENER_KUBECONFIG}" get cloudprofiles.core.gardener.cloud aws -o go-template='{{range .spec.kubernetes.versions}}{{if eq .classification "supported"}}{{.version}}{{break}}{{end}}{{end}}')
    fi

    # print configurations for debugging purposes:
    log::banner "Configurations:"
    echo "CLUSTER_NAME: ${CLUSTER_NAME}"
    echo "GARDENER_REGION: ${GARDENER_REGION}"
    echo "GARDENER_ZONES: ${GARDENER_ZONES}"
    echo "MACHINE_TYPE: ${MACHINE_TYPE}"
    echo "SCALER_MIN: ${SCALER_MIN}"
    echo "SCALER_MAX: ${SCALER_MAX}"
    echo "GARDENER_CLUSTER_VERSION: ${GARDENER_CLUSTER_VERSION}"
    echo "RETRY_ATTEMPTS ${RETRY_ATTEMPTS}"
}

gardener::provision_cluster() {
    log::banner "Provision cluster: \"${CLUSTER_NAME}\""

    cat << EOF | kubectl apply --kubeconfig="${GARDENER_KUBECONFIG}" -f -
apiVersion: core.gardener.cloud/v1beta1
kind: Shoot
metadata:
  name: ${CLUSTER_NAME}
spec:
  secretBindingName: ${GARDENER_PROVIDER_SECRET_NAME}
  cloudProfileName: aws
  region: ${GARDENER_REGION}
  purpose: evaluation
  provider:
    type: aws
    infrastructureConfig:
      apiVersion: aws.provider.extensions.gardener.cloud/v1alpha1
      kind: InfrastructureConfig
      networks:
        vpc:
          cidr: 10.250.0.0/16
        zones:
          - name: ${GARDENER_REGION}a
            internal: 10.250.112.0/22
            public: 10.250.96.0/22
            workers: 10.250.0.0/19
    workers:
    - name: cpu-worker
      minimum: ${SCALER_MIN}
      maximum: ${SCALER_MAX}
      machine:
        type: ${MACHINE_TYPE}
      volume:
        type: gp2
        size: 50Gi
      zones:
        - ${GARDENER_REGION}a
  networking:
    type: calico
    pods: 100.96.0.0/11
    nodes: 10.250.0.0/16
    services: 100.64.0.0/13
  kubernetes:
    version: ${GARDENER_CLUSTER_VERSION}
  hibernation:
    enabled: false
  addons:
    nginxIngress:
      enabled: false
EOF

    echo "waiting fo cluster to be ready..."
    kubectl wait --kubeconfig="${GARDENER_KUBECONFIG}"--for=condition=EveryNodeReady shoot/${CLUSTER_NAME} --timeout=17m

    # create kubeconfig request, that creates a Kubeconfig, which is valid for one day
    kubectl create --kubeconfig="${GARDENER_KUBECONFIG}" \
        -f <(printf '{"spec":{"expirationSeconds":86400}}') \
        --raw /apis/core.gardener.cloud/v1beta1/namespaces/garden-${GARDENER_PROJECT_NAME}/shoots/${CLUSTER_NAME}/adminkubeconfig | \
        jq -r ".status.kubeconfig" | \
        base64 -d > ${CLUSTER_NAME}_kubeconfig.yaml

    # merge with the existing kubeconfig settings
    mkdir -p ~/.kube
    KUBECONFIG="~/.kube/config:${CLUSTER_NAME}_kubeconfig.yaml" kubectl config view --merge > merged_kubeconfig.yaml
    mv merged_kubeconfig.yaml ~/.kube/config
}

## MAIN Logic

gardener::validate_and_default

gardener::provision_cluster
