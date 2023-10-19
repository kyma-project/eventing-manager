#!/usr/bin/env bash

set -ue

DESIRED_TAG=$1

source .version

if [[ "$DESIRED_TAG" != "$MODULE_VERSION" ]]; then
  echo "Tags mismatch: expected ${MODULE_VERSION}, got $DESIRED_TAG"
  exit 1
fi
echo "Tags are correct"
exit 0