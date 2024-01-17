#!/bin/bash

# This script will remove eventing related resources and all required modules.
# Usage: No parameters are required to run this script.

source "$(dirname "$0")/utils/log.sh"

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

set +o errexit  # exit immediately when a command fails.

log::banner "Deleting all Subscriptions"
kubectl delete --wait=false -A subscriptions --all

log::banner "Deleting all API Rules"
kubectl delete --wait=false -A apirules.gateway.kyma-project.io --all

log::banner "Uninstalling Eventing module"
make undeploy

log::banner "Uninstalling NATS module"
kubectl delete --ignore-not-found -f https://github.com/kyma-project/nats-manager/releases/latest/download/nats-default-cr.yaml
kubectl delete --wait=false --ignore-not-found -f https://github.com/kyma-project/nats-manager/releases/latest/download/nats-manager.yaml

log::banner "Uninstalling API Gateway module"
kubectl delete --ignore-not-found -f https://github.com/kyma-project/api-gateway/releases/latest/download/apigateway-default-cr.yaml
kubectl delete --wait=false --ignore-not-found -f https://github.com/kyma-project/api-gateway/releases/latest/download/api-gateway-manager.yaml

log::banner "Uninstalling Istio module"
kubectl delete --ignore-not-found -f https://github.com/kyma-project/istio/releases/latest/download/istio-default-cr.yaml
kubectl delete --wait=false --ignore-not-found -f https://github.com/kyma-project/istio/releases/latest/download/istio-manager.yaml
kubectl label namespace kyma-system istio-injection-

exit 0
