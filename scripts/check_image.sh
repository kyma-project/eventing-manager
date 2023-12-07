#!/usr/bin/env bash

REF_NAME="${1:-"main"}"

#SHORT_EXPECTED_SHA=$(git rev-parse --short=8 "${REF_NAME}~")
#DATE="v$(git show ${SHORT_EXPECTED_SHA} --date=format:'%Y%m%d' --format=%ad -q)"

EXPECTED_TAG="${REF_NAME}"

IMAGE_TO_CHECK="${2:-europe-docker.pkg.dev/kyma-project/prod/eventing-manager}"
BUMPED_IMAGE_TAG=$(cat sec-scanners-config.yaml | grep "${IMAGE_TO_CHECK}" | cut -d : -f 2)

if [[ "$BUMPED_IMAGE_TAG" != "$EXPECTED_TAG" ]]; then
  echo "Tags are not correct: wanted $EXPECTED_TAG but got $BUMPED_IMAGE_TAG"
  exit 1
fi
echo "Tags are correct"
exit 0