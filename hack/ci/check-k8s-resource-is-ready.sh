#!/bin/bash

timeouttime=${1:-180}
resource=${2:-nats}
namespace=${3:-kyma-system}

check-nats-ready() {
  echo "checking ${resource} in the namespace ${namespace} to become 'Ready' for ${timeouttime} seconds"

  timeout ${timeouttime}  bash -c 'while [[ "$(kubectl get -n ${namespace} ${resource} -ojsonpath='{.items[0].status.state}')" != "Ready" ]]; do sleep 1; done'

  if [[ $? -ne 0 ]]; then
    echo "NATS was not 'Ready' after ${timeouttime} seconds"
    exit 1
  fi

  echo "NATS is 'Ready'"
}

check-nats-ready
