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
      export API_RULE_GATEWAY="kyma-system/kyma-gateway"
  fi

  if [ -z "$DOMAIN_TO_EXPOSE_WORKLOADS" ]; then
      export DOMAIN_TO_EXPOSE_WORKLOADS="$(kubectl get cm -n kube-system shoot-info -o=jsonpath='{.data.domain}')"
  fi
}

create_api_rule_for_epp() {
  cat <<EOF | kubectl apply -f -
apiVersion: gateway.kyma-project.io/v2
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
    - path: /*
      methods: ["GET", "POST"]
      noAuth: true
EOF
}

create_api_rule_for_sink() {
  cat <<EOF | kubectl apply -f -
apiVersion: gateway.kyma-project.io/v2
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
    - path: /*
      methods: ["GET", "POST"]
      noAuth: true
EOF
}

create_authorization_policy_for_sink() {
  cat <<EOF | kubectl apply -f -
apiVersion: security.istio.io/v1
kind: AuthorizationPolicy
metadata:
  name: allow-internal-sink
  namespace: eventing-tests
spec:
  selector:
    matchLabels:
      name: test-sink
  action: ALLOW
  rules:
  - from:
    - source:
        notPrincipals: ["cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account"]
EOF
}

wait_for_endpoint_status() {
  local endpoint="$1"
  local expected_status="$2"
  local timeout=300  # Timeout in seconds
  local interval=5   # Interval between checks in seconds
  local elapsed=0

  echo "Waiting for endpoint $endpoint to return status $expected_status..."

  while true; do
    # Get the HTTP status code or capture curl errors
    response=$(curl -s -o /dev/null -w "%{http_code}" "$endpoint" 2>&1)
    status_code=$(echo "$response" | grep -Eo '^[0-9]{3}')

    if [ "$status_code" -eq "$expected_status" ]; then
      echo "Endpoint $endpoint is now returning status $expected_status."
      return 0
    fi

    # Check for specific curl errors and retry
    if echo "$response" | grep -qE "(dial tcp|i/o timeout|no such host)"; then
      echo "Retrying due to error: $response"
    else
      echo "Unexpected response: $response"
    fi

    # Check if timeout is reached
    if [ "$elapsed" -ge "$timeout" ]; then
      echo "Timeout reached while waiting for $endpoint to return status $expected_status."
      return 1
    fi

    # Wait and increment elapsed time
    sleep "$interval"
    elapsed=$((elapsed + interval))
  done
}

wait_for_api_rules_readiness() {
  echo "waiting for EPP APIRule to be ready..."
  kubectl wait apirules.gateway.kyma-project.io -n default epp-e2e-tests --timeout=150s --for=jsonpath='{.status.state}'=Ready

  echo "waiting for sink APIRule to be ready..."
  kubectl wait apirules.gateway.kyma-project.io -n default sink-e2e-tests --timeout=150s --for=jsonpath='{.status.state}'=Ready

  echo "wait for endpoints to be reachable"
  wait_for_endpoint_status "https://sink-e2e-tests.$DOMAIN_TO_EXPOSE_WORKLOADS/events/190d697d-e842-4b0e-bc57-e7731dba6031/" 404
  wait_for_endpoint_status "https://epp-e2e-tests.$DOMAIN_TO_EXPOSE_WORKLOADS/events" 404
}

## Main logic

validate_and_default

create_api_rule_for_epp

create_api_rule_for_sink

create_authorization_policy_for_sink

wait_for_api_rules_readiness
