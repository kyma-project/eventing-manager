#!/usr/bin/env bash

# Usage: To run this script, set the following environment variables and run this script.
#  CLUSTER_NAME
#  GARDENER_REGION
#  GARDENER_ZONES
#  GARDENER_PROJECT_NAME
#  GARDENER_PROVIDER_SECRET_NAME
#  GARDENER_KUBECONFIG - Path to kubeconfig for Gardener.
#  MACHINE_TYPE - default: "m5.xlarge"
#  SCALER_MIN - default: 1
#  SCALER_MAX - default: 2
#  RETRY_ATTEMPTS - default: 1
#  GARDENER_CLUSTER_VERSION - default as defined in kyma CLI.

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
        # grep the default kube-version defined in kyma CLI.
        export GARDENER_CLUSTER_VERSION="$(${KYMA_CLI} provision gardener aws --help | grep "kube-version string" | awk -F "\"" '{print $2}')"
        log::info "Using GARDENER_CLUSTER_VERSION=${GARDENER_CLUSTER_VERSION}"
    fi
}

gardener::provision_cluster() {
    log::info "Provision cluster: \"${CLUSTER_NAME}\""

    # decreasing attempts to 2 because we will try to create new cluster from scratch on exit code other than 0
    ${KYMA_CLI} provision gardener aws \
      --secret "${GARDENER_PROVIDER_SECRET_NAME}" \
      --name "${CLUSTER_NAME}" \
      --project "${GARDENER_PROJECT_NAME}" \
      --credentials "${GARDENER_KUBECONFIG}" \
      --region "${GARDENER_REGION}" \
      --zones "${GARDENER_ZONES}" \
      --type "${MACHINE_TYPE}" \
      --scaler-min ${SCALER_MIN} \
      --scaler-max ${SCALER_MAX} \
      --kube-version="${GARDENER_CLUSTER_VERSION}" \
      --attempts ${RETRY_ATTEMPTS} \
      --verbose \
      --hibernation-start ""
}

## MAIN Logic

gardener::validate_and_default

gardener::provision_cluster
