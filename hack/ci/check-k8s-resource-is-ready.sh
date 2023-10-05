#!/bin/bash

timeouttime=${1:-180}

check-nats-ready() {
  timeout ${timeouttime}  bash -c 'while [[ "$(kubectl get -n kyma-system nats -ojsonpath='{.items[0].status.state}')" != "Ready" ]]; do sleep 1; done'

  if [[ $? -ne 0 ]]; then
    echo "NATS was not 'Ready' after ${timeouttime} seconds"
    exit 1
  fi

  echo "NATS is 'Ready'"
}

check-nats-ready
