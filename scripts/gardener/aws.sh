#!/usr/bin/env bash

#Permissions: In order to run this script you need to use a service account with permissions equivalent to the following GCP roles:
# - Compute Admin
# - Service Account User
# - Service Account Admin
# - Service Account Token Creator
# - Make sure the service account is enabled for the Google Identity and Access Management API.

source "${PROJECT_ROOT}/scripts/utils/log.sh"
source "${PROJECT_ROOT}/scripts/utils/utils.sh"

gardener::init() {
    requiredVars=(
        GARDENER_REGION
        GARDENER_ZONES
        GARDENER_KUBECONFIG
        GARDENER_PROJECT_NAME
        GARDENER_PROVIDER_SECRET_NAME
        GARDENER_CLUSTER_VERSION
    )
    utils::check_required_vars "${requiredVars[@]}"

    # set default values
    if [ -z "$MACHINE_TYPE" ]; then
        export MACHINE_TYPE="m5.xlarge"
    fi
}

gardener::provision_cluster() {
    log::info "Provision cluster: \"${CLUSTER_NAME}\""
    if [ "${#CLUSTER_NAME}" -gt 9 ]; then
        log::error "Provided cluster name is too long"
        return 1
    fi

    # decreasing attempts to 2 because we will try to create new cluster from scratch on exit code other than 0
    ${KYMA_CLI} provision gardener aws \
      --secret "${GARDENER_PROVIDER_SECRET_NAME}" \
      --name "${CLUSTER_NAME}" \
      --project "${GARDENER_PROJECT_NAME}" \
      --credentials "${GARDENER_KUBECONFIG}" \
      --region "${GARDENER_REGION}" \
      --zones "${GARDENER_ZONES}" \
      --type "${MACHINE_TYPE}" \
      --scaler-max 4 \
      --scaler-min 2 \
      --kube-version="${GARDENER_CLUSTER_VERSION}" \
      --attempts 1 \
      --verbose \
      --hibernation-start ""
}

gardener::cleanup() {
    log::info "Deprovision cluster: \"${CLUSTER_NAME}\""
        gardener::deprovision_cluster \
            -p "${GARDENER_PROJECT_NAME}" \
            -c "${CLUSTER_NAME}" \
            -f "${GARDENER_KUBECONFIG}"
}

# gardener::deprovision_cluster removes a Gardener cluster
#
# Arguments:
#
# required:
# p - project name
# c - cluster name
# f - kubeconfig file path
function gardener::deprovision_cluster() {
  local OPTIND
  local projectName
  local clusterName
  local kubeconfigFile
  local namespace
  local wait="false"

  while getopts ":p:c:f:w:" opt; do
      case $opt in
          p)
            projectName="$OPTARG" ;;
          c)
            clusterName="$OPTARG" ;;
          f)
            kubeconfigFile="$OPTARG" ;;
          w)
            wait=${OPTARG:-$wait} ;;
          \?)
              echo "Invalid option: -$OPTARG" >&2; exit 1 ;;
          :)
              echo "Option -$OPTARG argument not provided" >&2 ;;
      esac
  done


  utils::check_empty_arg "$projectName" "Project name is empty. Exiting..."
  utils::check_empty_arg "$clusterName" "Cluster name is empty. Exiting..."
  utils::check_empty_arg "$kubeconfigFile" "Kubeconfig file path is empty. Exiting..."

  log::info "Deprovision cluster: ${clusterName}"

  namespace="garden-${projectName}"

  kubectl annotate shoot "${clusterName}" confirmation.gardener.cloud/deletion=true \
    --overwrite \
    -n "${namespace}" \
    --kubeconfig "${kubeconfigFile}"
  kubectl delete shoot "${clusterName}" \
    --wait="${wait}" \
    --kubeconfig "${kubeconfigFile}" \
    -n "${namespace}"
}
