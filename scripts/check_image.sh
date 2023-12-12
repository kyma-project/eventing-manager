#!/usr/bin/env bash

# Get release version
REF_NAME="${1:-"main"}"

# Get eventing-manager tag from sec-scanners-config.yaml
IMAGE_TO_CHECK="${2:-europe-docker.pkg.dev/kyma-project/prod/eventing-manager}"
BUMPED_IMAGE_TAG=$(cat sec-scanners-config.yaml | grep "${IMAGE_TO_CHECK}" | cut -d : -f 2)

# Check BUMPED_IMAGE_TAG and required image tag
if [[ "$BUMPED_IMAGE_TAG" != "$REF_NAME" ]]; then
  # ERROR: Tag issue
  echo "Tags are not correct: wanted $REF_NAME but got $BUMPED_IMAGE_TAG"
  exit 1
fi

# OK: Everything is fine
echo "Tags are correct"
exit 0
