#!/usr/bin/env bash

##############################
# Check tags in security-scan-config.yaml
# Image Tag, rc-tag
##############################


# Get release version
DESIRED_TAG="${1:-"main"}"

# Get eventing-manager tag from sec-scanners-config.yaml
SEC_SCAN_TO_CHECK="${2:-europe-docker.pkg.dev/kyma-project/prod/eventing-manager}"
# Get rc-tag
RC_TAG_TO_CHECK="${3:-rc-tag}"

IMAGE_TAG=$(cat sec-scanners-config.yaml | grep "${SEC_SCAN_TO_CHECK}" | cut -d : -f 2)
RC_TAG=$(cat sec-scanners-config.yaml | grep "${RC_TAG_TO_CHECK}" | cut -d : -f 2)

echo $RC_TAG
echo $IMAGE_TAG

# Check IMAGE_TAG and required image tag
if [[ "$IMAGE_TAG" != "$DESIRED_TAG" ]] || [[ "$RC_TAG" != "$DESIRED_TAG" ]]; then
  # ERROR: Tag issue
  echo "Tags are not correct:
  - wanted $DESIRED_TAG
  - securoty-scanner image tag: $IMAGE_TAG
  - rc-tag: $RC_TAG"
  exit 1
fi

# OK: Everything is fine
echo "Tags are correct"
exit 0
