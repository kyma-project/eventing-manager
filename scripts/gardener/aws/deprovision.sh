#!/usr/bin/env bash

# Usage: To run this script, set the following environment variables and run this script.
#  CLUSTER_NAME
#  GARDENER_PROJECT_NAME
#  GARDENER_KUBECONFIG - Path to kubeconfig for Gardener.
#  WAIT_FOR_DELETE_COMPLETION - "true" | "false"

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
        GARDENER_KUBECONFIG
        GARDENER_PROJECT_NAME
    )
    utils::check_required_vars "${requiredVars[@]}"

    # set default values
    if [ -z "$WAIT_FOR_DELETE_COMPLETION" ]; then
        export WAIT_FOR_DELETE_COMPLETION="false"
    fi
}

# gardener::deprovision_cluster removes a Gardener cluster
# Reference: https://github.com/gardener/gardener/blob/master/hack/usage/delete
function gardener::deprovision_cluster() {
  log::info "De-provisioning cluster: ${CLUSTER_NAME}"

  local namespace
  namespace="garden-${GARDENER_PROJECT_NAME}"

  kubectl annotate shoot "${CLUSTER_NAME}" confirmation.gardener.cloud/deletion=true \
    --overwrite \
    -n "${namespace}" \
    --kubeconfig "${GARDENER_KUBECONFIG}"
  kubectl delete shoot "${CLUSTER_NAME}" \
    --wait="${WAIT_FOR_DELETE_COMPLETION}" \
    --kubeconfig "${GARDENER_KUBECONFIG}" \
    -n "${namespace}"
}

## MAIN Logic

gardener::validate_and_default

gardener::deprovision_cluster
