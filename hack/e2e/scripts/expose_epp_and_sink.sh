#!/usr/bin/env bash

# This script will create APIRules for publisher proxy and sink for e2e-tests.
#
# Usage: To run this script, set the following environment variables and run this script.
# - API_RULE_GATEWAY (optional)
# - DOMAIN_TO_EXPOSE_WORKLOADS (optional)

# standard bash error handling
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

validate_and_default() {
  # set default values
  if [ -z "$API_RULE_GATEWAY" ]; then
      export API_RULE_GATEWAY="kyma-gateway.kyma-system.svc.cluster.local"
  fi

  if [ -z "$DOMAIN_TO_EXPOSE_WORKLOADS" ]; then
      export DOMAIN_TO_EXPOSE_WORKLOADS="$(kubectl get cm -n kube-system shoot-info -o=jsonpath='{.data.domain}')"
  fi
}

create_api_rule_for_epp() {
  cat <<EOF | kubectl apply -f -
apiVersion: gateway.kyma-project.io/v2alpha1
kind: APIRule
metadata:
  name: epp-e2e-tests
  namespace: default
spec:
  hosts:
    - epp-e2e-tests.$DOMAIN_TO_EXPOSE_WORKLOADS
  service:
    name: eventing-publisher-proxy
    namespace: kyma-system
    port: 80
  gateway: $API_RULE_GATEWAY
  rules:
    - path: /.*
      methods: ["GET"]
      noAuth: true
    - path: /post
      methods: ["POST"]
      noAuth: true
EOF
}

create_api_rule_for_sink() {
  cat <<EOF | kubectl apply -f -
apiVersion: gateway.kyma-project.io/v2alpha1
kind: APIRule
metadata:
  name: sink-e2e-tests
  namespace: default
spec:
  hosts:
    - sink-e2e-tests.$DOMAIN_TO_EXPOSE_WORKLOADS
  service:
    name: test-sink
    namespace: eventing-tests
    port: 80
  gateway: $API_RULE_GATEWAY
  rules:
    - path: /.*
      methods: ["GET"]
      noAuth: true
    - path: /post
      methods: ["POST"]
      noAuth: true
EOF
}

wait_for_api_rules_readiness() {
  echo "waiting for EPP APIRule to be ready..."
  kubectl wait apirules.gateway.kyma-project.io -n default epp-e2e-tests --timeout=150s --for=jsonpath='{.status.APIRuleStatus.code}'=OK

  echo "waiting for sink APIRule to be ready..."
  kubectl wait apirules.gateway.kyma-project.io -n default sink-e2e-tests --timeout=150s --for=jsonpath='{.status.APIRuleStatus.code}'=OK
}

## Main logic

validate_and_default

create_api_rule_for_epp

create_api_rule_for_sink

wait_for_api_rules_readiness
