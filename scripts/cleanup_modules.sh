#!/bin/bash

# This script will remove eventing related resources and all required modules.
# Usage: No parameters are required to run this script.

source "$(dirname "$0")/utils/log.sh"

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked
# NOTE: This script will continue on errors and will always return exit code 0.
set +o errexit  # continue immediately when a command fails.

log::banner "Deleting all Subscriptions"
kubectl delete --timeout=120s --wait=false -A subscriptions --all

log::banner "Deleting all API Rules"
kubectl delete --timeout=120s --wait=false -A apirules.gateway.kyma-project.io --all

log::banner "Uninstalling Eventing module"
kubectl delete --timeout=120s --ignore-not-found -f https://github.com/kyma-project/eventing-manager/releases/latest/download/eventing-default-cr.yaml
make undeploy

log::banner "Uninstalling NATS module"
kubectl delete --timeout=120s --ignore-not-found -f https://github.com/kyma-project/nats-manager/releases/latest/download/nats-default-cr.yaml
kubectl delete --timeout=120s --wait=false --ignore-not-found -f https://github.com/kyma-project/nats-manager/releases/latest/download/nats-manager.yaml

log::banner "Uninstalling API Gateway module"
kubectl delete --timeout=120s --ignore-not-found -f https://github.com/kyma-project/api-gateway/releases/latest/download/apigateway-default-cr.yaml
kubectl delete --timeout=120s --wait=false --ignore-not-found -f https://github.com/kyma-project/api-gateway/releases/latest/download/api-gateway-manager.yaml
# Delete peer auth
kubectl delete --timeout=120s --wait=false peerauthentication -n kyma-system ory-oathkeeper-maester-metrics

log::banner "Uninstalling Istio module"
kubectl delete --timeout=120s --ignore-not-found -f https://github.com/kyma-project/istio/releases/latest/download/istio-default-cr.yaml
kubectl delete --timeout=120s --wait=false --ignore-not-found -f https://github.com/kyma-project/istio/releases/latest/download/istio-manager.yaml
kubectl label namespace kyma-system istio-injection-

exit 0
